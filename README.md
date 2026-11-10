# Modul 6 Backend — Modul 9

The application uses PostgreSQL for persistent data and transactional KRS/supervision operations.

## Local setup

1. Create a PostgreSQL database and set the connection string in PowerShell:

   ```powershell
   $env:DATABASE_URL = "postgres://USER:PASSWORD@localhost:5432/DATABASE?sslmode=disable"
   $env:JWT_SECRET = "replace-with-a-long-random-secret"
   ```

2. Apply the numbered migrations and create an administrator. The migrations
   seed demo students, courses, and supervisors, plus separate load-test users,
   students, courses, and supervisors. Load-test users cannot log in with their
   seeded placeholder passwords.

   ```powershell
   go run ./cmd/migrate
   $env:ADMIN_USERNAME = "admin"
   $env:ADMIN_PASSWORD = "choose-a-strong-password-with-digits"
   go run ./cmd/create-admin
   ```

   Set `ADMIN_PASSWORD` only in the environment. The command stores a bcrypt hash
   and does not print the password. Public registration always creates a `user`;
   it cannot grant administrator permissions.

3. Start the server:

   ```powershell
   go run .
   ```

   `DB_MAX_CONNS` defaults to 10 and `LISTEN_ADDR` defaults to `:3000`.
   Migrations are explicit; apply them before starting the application.

## Module 9 API

All routes below are under `/api/v1` and require a bearer token unless noted.

| Method and path | Behavior |
|---|---|
| `GET /health` | Checks PostgreSQL connectivity |
| `GET /courses`, `GET /courses/:id` | List or retrieve courses |
| `GET /courses/mine` | Current user's enrollment and 24-credit limit |
| `POST /courses/:id/enroll` | Enroll atomically, enforcing course quota and credit limit |
| `DELETE /courses/:id/enroll` | Drop an enrollment |
| `POST /enrollments/swap` | Atomically swap courses; body has `from_course_id` and `to_course_id` |
| `PATCH /courses/:id` | Optimistically update course quota with `quota` and `version`; requires `course:manage` |
| `GET /supervisors` | List supervisor quotas |
| `POST /supervisors/:id/assign` | Assign a student; body has `student_id`; requires `supervision:manage` |
| `POST /supervisions/switch` | Atomically switch a student's supervisor; body has `student_id`, `from_supervisor_id`, and `to_supervisor_id` |
| `PATCH /supervisors/:id` | Optimistically update supervisor quota with `quota` and `version`; requires `supervision:manage` |

Course and supervisor counters have database `CHECK` constraints as a final
invariant. Enrollment/assignment and counter changes use one transaction.
Student rows are locked before checking per-student rules, shared course or
supervisor rows are locked in ascending ID order, and quota updates increment
their version in the same SQL statement.

## Concurrent load checks

Keep the server running in one terminal. In another terminal, with the same
`DATABASE_URL` and `JWT_SECRET`, run one mode at a time:

```powershell
go run ./tools/loadtest -mode crowd
go run ./tools/loadtest -mode credits
go run ./tools/loadtest -mode swap
go run ./tools/loadtest -mode supervision
```

Load-test fixtures are marked in the database. Each mode resets only marked
fixtures and their course/supervisor counters, leaving ordinary records intact.
`supervision` requires an administrator account. For repeatable results, run
each mode three times; each invocation resets its own test data.

Run local checks with:

```powershell
go test ./...
go build ./...
go vet ./...
```
