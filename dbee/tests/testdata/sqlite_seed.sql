CREATE TABLE IF NOT EXISTS test_table (
    id INTEGER PRIMARY KEY,
    username TEXT,
    email TEXT
);

INSERT INTO test_table (id, username, email) VALUES
    (1, 'john_doe', 'john@example.com'),
    (2, 'jane_smith', 'jane@example.com'),
    (3, 'bob_wilson', 'bob@example.com');

CREATE VIEW IF NOT EXISTS test_view AS
    SELECT id, username, email
    FROM test_table
    WHERE id = 2;

