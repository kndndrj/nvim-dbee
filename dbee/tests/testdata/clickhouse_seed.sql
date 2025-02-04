CREATE DATABASE IF NOT EXISTS test;

CREATE TABLE IF NOT EXISTS test.test_table
(
    id UInt32,
    username String,
    email String,
    created_at DateTime,
    is_active UInt8
) ENGINE = MergeTree()
ORDER BY id
;

INSERT INTO test.test_table (id, username, email, created_at, is_active) VALUES
    (1, 'john_doe', 'john@example.com', '2023-01-01 10:00:00', 1),
    (2, 'jane_smith', 'jane@example.com', '2023-01-02 11:30:00', 1),
    (3, 'bob_wilson', 'bob@example.com', '2023-01-03 09:15:00', 0)
;

CREATE VIEW IF NOT EXISTS test.test_view AS
SELECT id, username, email, created_at
FROM test.test_table
WHERE is_active = 1
;

