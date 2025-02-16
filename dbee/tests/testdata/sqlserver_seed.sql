/*
Each transaction need to be separated by GO,
see more how t-sql works:
http://learn.microsoft.com/en-us/sql/linux/sql-server-linux-docker-container-deployment?view=sql-server-2017&pivots=cs1-bash
*/

CREATE SCHEMA test_schema;
GO

CREATE TABLE test_schema.test_table (
    ID INT PRIMARY KEY IDENTITY,
    Name NVARCHAR(100),
    Email NVARCHAR(100) UNIQUE
);
GO

INSERT INTO test_schema.test_table (Name, Email) VALUES
    ('Alice', 'alice@example.com'),
    ('Bob', 'bob@example.com')
;
GO

CREATE VIEW test_schema.test_view AS (
    SELECT * FROM test_schema.test_table WHERE Name = 'Bob'
);
GO

CREATE DATABASE dev;
GO
