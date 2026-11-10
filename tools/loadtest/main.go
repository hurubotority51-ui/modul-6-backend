package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"modul6/helper"
	"modul6/model"
	"modul6/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type requestJob struct {
	method string
	path   string
	token  string
	body   any
}

func main() {
	mode := flag.String("mode", "", "load mode: crowd, credits, swap, or supervision")
	apiURL := flag.String("api", envOr("API_URL", "http://localhost:3000/api/v1"), "API base URL")
	flag.Parse()
	if *mode == "" {
		fatal(errors.New("-mode is required"))
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fatal(errors.New("DATABASE_URL is required"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fatal(err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	var runErr error
	switch *mode {
	case "crowd":
		runErr = crowd(ctx, pool, client, *apiURL)
	case "credits":
		runErr = credits(ctx, pool, client, *apiURL)
	case "swap":
		runErr = swap(ctx, pool, client, *apiURL)
	case "supervision":
		runErr = supervisions(ctx, pool, client, *apiURL)
	default:
		runErr = fmt.Errorf("unknown mode %q", *mode)
	}
	if runErr != nil {
		fatal(runErr)
	}
}

func crowd(ctx context.Context, pool *pgxpool.Pool, client *http.Client, api string) error {
	if err := resetEnrollments(ctx, pool); err != nil {
		return err
	}
	courseID, err := courseID(ctx, pool, "LT399")
	if err != nil {
		return err
	}
	users, err := usersByPrefix(ctx, pool, "loadtest", 60)
	if err != nil {
		return err
	}
	jobs := make([]requestJob, 0, len(users))
	for _, user := range users {
		token, err := service.GenerateAccessToken(user)
		if err != nil {
			return err
		}
		jobs = append(jobs, requestJob{
			method: http.MethodPost, path: fmt.Sprintf("/courses/%d/enroll", courseID), token: token,
		})
	}
	results, elapsed := fire(client, api, jobs)
	printResults("60 mahasiswa berebut LT399", results, elapsed)
	var quota, enrolledCount, enrollmentRows int
	err = pool.QueryRow(ctx,
		`SELECT quota, enrolled_count, COUNT(e.id)::INTEGER
		 FROM courses c LEFT JOIN enrollments e ON e.course_id = c.id
		 WHERE c.id = $1 AND c.is_load_test GROUP BY c.id`, courseID,
	).Scan(&quota, &enrolledCount, &enrollmentRows)
	if err != nil {
		return err
	}
	fmt.Printf("LT399 kuota=%d enrolled_count=%d baris=%d\n", quota, enrolledCount, enrollmentRows)
	if results["201"] != 5 || results["409 COURSE_FULL"] != 55 ||
		enrolledCount != 5 || enrollmentRows != 5 {
		return errors.New("crowd mode failed the expected capacity/concurrency invariant")
	}
	return nil
}

func credits(ctx context.Context, pool *pgxpool.Pool, client *http.Client, api string) error {
	if err := resetEnrollments(ctx, pool); err != nil {
		return err
	}
	users, err := usersByPrefix(ctx, pool, "loadtest", 10)
	if err != nil {
		return err
	}
	courses, err := courseIDs(ctx, pool, 12)
	if err != nil {
		return err
	}
	jobs := make([]requestJob, 0, len(users)*len(courses))
	for _, user := range users {
		token, err := service.GenerateAccessToken(user)
		if err != nil {
			return err
		}
		for _, courseID := range courses {
			jobs = append(jobs, requestJob{
				method: http.MethodPost, path: fmt.Sprintf("/courses/%d/enroll", courseID), token: token,
			})
		}
	}
	results, elapsed := fire(client, api, jobs)
	printResults("10 mahasiswa mengambil 12 kelas secara serentak", results, elapsed)
	rows, err := pool.Query(ctx,
		`SELECT u.username, COALESCE(SUM(c.credits), 0)::INTEGER
		 FROM users u LEFT JOIN enrollments e ON e.user_id = u.id
		 LEFT JOIN courses c ON c.id = e.course_id
		 WHERE u.is_load_test GROUP BY u.id, u.username ORDER BY u.username LIMIT 10`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var username string
		var total int
		if err := rows.Scan(&username, &total); err != nil {
			return err
		}
		if total > service.MaxCreditsPerSemester {
			return fmt.Errorf("%s has %d credits, limit is %d", username, total, service.MaxCreditsPerSemester)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if results["201"] != 80 || results["409 CREDIT_LIMIT_EXCEEDED"] != 40 {
		return errors.New("credits mode failed the expected credit limit")
	}
	return nil
}

func swap(ctx context.Context, pool *pgxpool.Pool, client *http.Client, api string) error {
	if err := resetEnrollments(ctx, pool); err != nil {
		return err
	}
	if101, err := courseID(ctx, pool, "LT101")
	if err != nil {
		return err
	}
	if102, err := courseID(ctx, pool, "LT102")
	if err != nil {
		return err
	}
	users, err := usersByPrefix(ctx, pool, "loadtest", 40)
	if err != nil {
		return err
	}
	if err := seedSwaps(ctx, pool, if101, if102, users); err != nil {
		return err
	}
	jobs := make([]requestJob, 0, len(users))
	for i, user := range users {
		token, err := service.GenerateAccessToken(user)
		if err != nil {
			return err
		}
		fromID, toID := if101, if102
		if i >= 20 {
			fromID, toID = if102, if101
		}
		jobs = append(jobs, requestJob{
			method: http.MethodPost, path: "/enrollments/swap", token: token,
			body: model.SwapRequest{FromCourseID: fromID, ToCourseID: toID},
		})
	}
	results, elapsed := fire(client, api, jobs)
	printResults("20 pasang mahasiswa pindah LT101/LT102", results, elapsed)
	for code, expected := range map[string]int{"LT101": 20, "LT102": 20} {
		id, err := courseID(ctx, pool, code)
		if err != nil {
			return err
		}
		var enrolled, actual int
		if err := pool.QueryRow(ctx,
			`SELECT c.enrolled_count, COUNT(e.id)::INTEGER
			 FROM courses c LEFT JOIN enrollments e ON e.course_id = c.id
			 WHERE c.id = $1 AND c.is_load_test GROUP BY c.id`, id,
		).Scan(&enrolled, &actual); err != nil {
			return err
		}
		if enrolled != expected || actual != expected {
			return fmt.Errorf("%s inconsistent counts: enrolled_count=%d rows=%d", code, enrolled, actual)
		}
	}
	if results["200"] != 40 {
		return errors.New("swap mode did not complete all 40 swaps")
	}
	return nil
}

func supervisions(ctx context.Context, pool *pgxpool.Pool, client *http.Client, api string) error {
	supervisorID, err := firstSupervisorID(ctx, pool)
	if err != nil {
		return err
	}
	admin, err := firstAdmin(ctx, pool)
	if err != nil {
		return err
	}
	token, err := service.GenerateAccessToken(admin)
	if err != nil {
		return err
	}
	if err := resetSupervisions(ctx, pool); err != nil {
		return err
	}
	studentIDs, err := seedStudentIDs(ctx, pool, 30)
	if err != nil {
		return err
	}
	jobs := make([]requestJob, 0, len(studentIDs))
	for _, studentID := range studentIDs {
		jobs = append(jobs, requestJob{
			method: http.MethodPost,
			path:   fmt.Sprintf("/supervisors/%d/assign", supervisorID),
			token:  token,
			body:   model.AssignSupervisionRequest{StudentID: studentID},
		})
	}
	results, elapsed := fire(client, api, jobs)
	printResults("30 mahasiswa mendapat pembimbing berkuota 3", results, elapsed)
	var quota, count, actual int
	err = pool.QueryRow(ctx,
		`SELECT s.quota, s.supervised_count, COUNT(p.student_id)::INTEGER
		 FROM supervisors s LEFT JOIN supervisions p ON p.supervisor_id = s.id
		 WHERE s.id = $1 AND s.is_load_test GROUP BY s.id`, supervisorID,
	).Scan(&quota, &count, &actual)
	if err != nil {
		return err
	}
	fmt.Printf("supervisor=%d kuota=%d supervised_count=%d baris=%d\n", supervisorID, quota, count, actual)
	if results["201"] != 3 || results["409 SUPERVISOR_FULL"] != 27 ||
		count != 3 || actual != 3 {
		return errors.New("supervision mode failed the expected capacity/concurrency invariant")
	}
	return nil
}

func usersByPrefix(ctx context.Context, pool *pgxpool.Pool, prefix string, limit int) ([]model.User, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, username, role FROM users
		 WHERE is_load_test AND username LIKE $1 ORDER BY username LIMIT $2`, prefix+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]model.User, 0, limit)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(users) != limit {
		return nil, fmt.Errorf("expected %d load-test users with prefix %q, found %d; apply migrations/004_load_test_fixtures.sql",
			limit, prefix, len(users))
	}
	return users, nil
}

func firstAdmin(ctx context.Context, pool *pgxpool.Pool) (model.User, error) {
	var user model.User
	err := pool.QueryRow(ctx,
		"SELECT id, username, role FROM users WHERE role = 'admin' ORDER BY id LIMIT 1",
	).Scan(&user.ID, &user.Username, &user.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, errors.New("no admin account; provision one before supervision load testing")
	}
	return user, err
}

func courseID(ctx context.Context, pool *pgxpool.Pool, code string) (int, error) {
	var id int
	err := pool.QueryRow(ctx,
		"SELECT id FROM courses WHERE code = $1 AND is_load_test", code,
	).Scan(&id)
	return id, err
}

func courseIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]int, error) {
	rows, err := pool.Query(ctx,
		"SELECT id FROM courses WHERE is_load_test AND code <> 'LT399' ORDER BY code LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int, 0, limit)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) != limit {
		return nil, fmt.Errorf("expected %d test courses, found %d", limit, len(ids))
	}
	return ids, nil
}

func firstSupervisorID(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var id int
	err := pool.QueryRow(ctx,
		"SELECT id FROM supervisors WHERE is_load_test ORDER BY id LIMIT 1").Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New("no load-test supervisor seeded; apply migrations/004_load_test_fixtures.sql")
	}
	return id, err
}

func seedStudentIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]int, error) {
	rows, err := pool.Query(ctx,
		"SELECT id FROM students WHERE is_load_test ORDER BY nim LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int, 0, limit)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func resetEnrollments(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx,
		"DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE is_load_test)",
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE courses c SET enrolled_count =
		 (SELECT COUNT(*)::INTEGER FROM enrollments e WHERE e.course_id = c.id)
		 WHERE c.is_load_test`,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func seedSwaps(ctx context.Context, pool *pgxpool.Pool, if101, if102 int, users []model.User) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	for i, user := range users {
		courseID := if101
		if i >= 20 {
			courseID = if102
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)", courseID, user.ID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE courses c SET enrolled_count =
		 (SELECT COUNT(*)::INTEGER FROM enrollments e WHERE e.course_id = c.id)
		 WHERE c.id = ANY($1::INTEGER[])`, []int{if101, if102}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func resetSupervisions(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx,
		`DELETE FROM supervisions
		 WHERE student_id IN (
		   SELECT id FROM students WHERE is_load_test
		 )
		 AND supervisor_id IN (
		   SELECT id FROM supervisors WHERE is_load_test
		 )`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE supervisors s SET supervised_count =
		 (SELECT COUNT(*)::INTEGER FROM supervisions p WHERE p.supervisor_id = s.id)
		 WHERE s.is_load_test`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func fire(client *http.Client, api string, jobs []requestJob) (map[string]int, time.Duration) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, len(jobs))
	for _, job := range jobs {
		wg.Add(1)
		go func(job requestJob) {
			defer wg.Done()
			<-start
			result, err := send(client, api, job)
			if err != nil {
				results <- "ERROR " + err.Error()
				return
			}
			results <- result
		}(job)
	}
	startTime := time.Now()
	close(start)
	wg.Wait()
	elapsed := time.Since(startTime)
	close(results)
	counts := make(map[string]int)
	for result := range results {
		counts[result]++
	}
	return counts, elapsed
}

func send(client *http.Client, api string, job requestJob) (string, error) {
	var body io.Reader
	if job.body != nil {
		encoded, err := json.Marshal(job.body)
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(job.method, strings.TrimRight(api, "/")+job.path, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+job.token)
	if job.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 400 {
		_, _ = io.Copy(io.Discard, response.Body)
		return fmt.Sprint(response.StatusCode), nil
	}
	var apiError helper.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
		return "", fmt.Errorf("HTTP %d: decode error response: %w", response.StatusCode, err)
	}
	if apiError.Code == "" {
		return fmt.Sprintf("%d %s", response.StatusCode, apiError.Message), nil
	}
	return fmt.Sprintf("%d %s", response.StatusCode, apiError.Code), nil
}

func printResults(title string, results map[string]int, elapsed time.Duration) {
	fmt.Printf("%s...\n", title)
	keys := make([]string, 0, len(results))
	for key := range results {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  %-32s %d\n", key, results[key])
	}
	fmt.Printf("total time: %s\n", elapsed.Round(time.Millisecond))
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
