ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_load_test BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE students
    ADD COLUMN IF NOT EXISTS is_load_test BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS is_load_test BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE supervisors
    ADD COLUMN IF NOT EXISTS is_load_test BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO users (username, email, password, role, is_load_test)
SELECT 'loadtest' || LPAD(g::TEXT, 3, '0'),
       'loadtest' || LPAD(g::TEXT, 3, '0') || '@loadtest.invalid',
       '!', 'user', TRUE
FROM generate_series(1, 60) AS g
ON CONFLICT (username) DO NOTHING;

INSERT INTO students (nim, name, grade, is_active, is_load_test)
SELECT 'LT' || LPAD(g::TEXT, 6, '0'),
       'Mahasiswa Load Test ' || LPAD(g::TEXT, 3, '0'),
       0, TRUE, TRUE
FROM generate_series(1, 60) AS g
ON CONFLICT (nim) DO NOTHING;

INSERT INTO courses (code, name, credits, quota, is_load_test) VALUES
    ('LT101', 'Load Test Course 01', 3, 40, TRUE),
    ('LT102', 'Load Test Course 02', 3, 40, TRUE),
    ('LT103', 'Load Test Course 03', 3, 40, TRUE),
    ('LT104', 'Load Test Course 04', 3, 40, TRUE),
    ('LT105', 'Load Test Course 05', 3, 40, TRUE),
    ('LT106', 'Load Test Course 06', 3, 40, TRUE),
    ('LT107', 'Load Test Course 07', 3, 40, TRUE),
    ('LT108', 'Load Test Course 08', 3, 40, TRUE),
    ('LT109', 'Load Test Course 09', 3, 40, TRUE),
    ('LT110', 'Load Test Course 10', 3, 40, TRUE),
    ('LT111', 'Load Test Course 11', 3, 40, TRUE),
    ('LT112', 'Load Test Course 12', 3, 40, TRUE),
    ('LT399', 'Load Test Crowded Course', 3, 5, TRUE)
ON CONFLICT (code) DO NOTHING;

INSERT INTO supervisors (name, quota, is_load_test)
VALUES ('Dosen Load Test', 3, TRUE)
ON CONFLICT (name) DO NOTHING;
