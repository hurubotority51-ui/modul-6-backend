CREATE TABLE IF NOT EXISTS roles (
    name TEXT PRIMARY KEY
);

INSERT INTO roles (name) VALUES ('admin'), ('user')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(30) NOT NULL UNIQUE,
    email VARCHAR(120) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role TEXT NOT NULL REFERENCES roles(name),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name TEXT NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    permission_name TEXT NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

INSERT INTO permissions (name, description) VALUES
    ('course:manage', 'Mengubah kuota kelas'),
    ('supervision:manage', 'Mengelola penetapan dosen pembimbing')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'course:manage'),
    ('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    nim VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    grade DOUBLE PRECISION NOT NULL CHECK (grade BETWEEN 0 AND 4),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS students_created_at_id_desc_idx
    ON students (created_at DESC, id DESC);

INSERT INTO students (nim, name, grade, is_active)
SELECT LPAD(g::TEXT, 8, '0'), 'Mahasiswa ' || LPAD(g::TEXT, 3, '0'), 0, TRUE
FROM generate_series(1, 60) AS g
ON CONFLICT (nim) DO NOTHING;
