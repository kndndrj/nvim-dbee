CREATE SCHEMA IF NOT EXISTS test;

CREATE TABLE IF NOT EXISTS test.test_table (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

INSERT INTO test.test_table (id, username, email, created_at) VALUES
    (1, 'john_doe', 'john@example.com', '2023-01-01 10:00:00'),
    (2, 'jane_smith', 'jane@example.com', '2023-01-02 11:30:00'),
    (3, 'bob_wilson', 'bob@example.com', '2023-01-03 09:15:00');

CREATE OR REPLACE VIEW test.test_view AS
SELECT id, username, email
FROM test.test_table
WHERE id = 2;
