CREATE SCHEMA IF NOT EXISTS test;

CREATE TABLE IF NOT EXISTS test.test_table (
    id INT UNSIGNED,
    username VARCHAR(255),
    email VARCHAR(255),
    PRIMARY KEY (id)
);

INSERT INTO test.test_table (id, username, email) VALUES
    (1, 'john_doe', 'john@example.com'),
    (2, 'jane_smith', 'jane@example.com'),
    (3, 'bob_wilson', 'bob@example.com');

CREATE OR REPLACE VIEW test.test_view AS
SELECT id, username, email
FROM test.test_table
WHERE id = 2;

