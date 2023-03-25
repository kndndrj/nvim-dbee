package clients

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

type PostgresClient struct {
	db *pgx.Conn
}

func NewPostgres(url string) (*PostgresClient, error) {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	return &PostgresClient{
		db: conn,
	}, nil
}

func (c *PostgresClient) Execute(query string) (Rows, error) {

	dbRows, err := c.db.Query(context.Background(), query) // Note: Ignoring errors for brevity
	if err != nil {
		return nil, err
	}

	pgRows := NewPGRows(dbRows)

	return pgRows, nil
}

func (c *PostgresClient) Schema() (Schema, error) {
	query := `
    SELECT table_schema, table_name FROM information_schema.tables UNION ALL
    SELECT schemaname, matviewname FROM pg_matviews;
	`

	rows, err := c.Execute(query)
	if err != nil {
		return nil, err
	}

	var schema = make(Schema)

	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		key := row[0].(string)
		val := row[1].(string)
		schema[key] = append(schema[key], val)
	}

	return schema, nil
}

func (c *PostgresClient) Close() {
	c.db.Close(context.Background())
}

type PGRows struct {
	dbRows pgx.Rows
}

func NewPGRows(pgRows pgx.Rows) *PGRows {
	return &PGRows{
		dbRows: pgRows,
	}
}

func (r *PGRows) Header() (Header, error) {
	dbCols := r.dbRows.FieldDescriptions()

	var header Header
	for _, col := range dbCols {
		header = append(header, col.Name)
	}
	return header, nil
}

func (r *PGRows) Next() (Row, error) {
	dbCols := r.dbRows.FieldDescriptions()

	if !r.dbRows.Next() {
		return nil, nil
	}

	// Create a slice of any's to represent each column,
	// and a second slice to contain pointers to each item in the columns slice.
	columns := make([]any, len(dbCols))
	columnPointers := make([]any, len(dbCols))
	for i := range columns {
		columnPointers[i] = &columns[i]
	}

	// Scan the result into the column pointers...
	if err := r.dbRows.Scan(columnPointers...); err != nil {
		return nil, err
	}

	// Create our map, and retrieve the value for each column from the pointers slice,
	// storing it in the map with the name of the column as the key.
	var row = make(Row, len(dbCols))
	for i := range dbCols {
		val := columnPointers[i].(*any)
		row[i] = *val
	}

	return row, nil
}
