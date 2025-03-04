-- Connect as system to grant privileges
ALTER SESSION SET CONTAINER = FREEPDB1;
grant create session, create table, create view, unlimited tablespace
to tester
;

-- Must match the APP_USER env in testcontainer
ALTER SESSION SET CURRENT_SCHEMA = tester;

CREATE TABLE test_table (
    id NUMBER,
    username VARCHAR2(255),
    email VARCHAR2(255),
    CONSTRAINT test_table_pk PRIMARY KEY (id)
);

INSERT INTO test_table (id, username, email) VALUES
    (1, 'john_doe', 'john@example.com');
INSERT INTO test_table (id, username, email) VALUES
    (2, 'jane_smith', 'jane@example.com');
INSERT INTO test_table (id, username, email) VALUES
    (3, 'bob_wilson', 'bob@example.com');

CREATE OR REPLACE VIEW test_view AS
    SELECT id, username, email
    FROM test_table
    WHERE id = 2;

commit
;

