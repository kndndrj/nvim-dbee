package integration

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	th "github.com/kndndrj/nvim-dbee/dbee/tests/testhelpers"
	"github.com/stretchr/testify/assert"
	tsuite "github.com/stretchr/testify/suite"
)

// DuckDBTestSuite defines the integration test suite for DuckDB.
type DuckDBTestSuite struct {
	tsuite.Suite
	ctr *th.DuckDBContainer
	ctx context.Context
	d   *core.Connection
}

// TestDuckDBTestSuite runs the test suite.
func TestDuckDBTestSuite(t *testing.T) {
	tsuite.Run(t, new(DuckDBTestSuite))
}

// SetupSuite initializes an in-memory DuckDB instance.
func (suite *DuckDBTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewDuckDBContainer(&core.ConnectionParams{
		ID:   "test-duckdb",
		Name: "test-duckdb",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver // easier access to driver

	// Create test table and insert random data
	setupSQL := `
		CREATE SCHEMA test;
		CREATE TABLE test.users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
		INSERT INTO test.users (id, name, created_at) VALUES
		(1, 'john', '2025-01-21 00:00:00'),
		(2, 'bob', '2025-01-21 00:01:00');
	`

	call := suite.d.Execute(setupSQL, nil)
	// TODO: (ph) not sure on thi
	err = call.Err()
	if err != nil {
		log.Fatal(err)
	}
}

// TeardownSuite cleans up after tests.
func (suite *DuckDBTestSuite) TeardownSuite() {
	suite.d.Close()
}

func (suite *DuckDBTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "SELECT COUNT(*) FROM range(5000000000)")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

// TestShouldReturnRows validates data retrieval for one rows.
func (suite *DuckDBTestSuite) TestShouldReturnOneRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{
		"id", "name", "created_at",
	}
	wantRows := []core.Row{
		{
			int32(1),
			"john",
			time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC),
		},
	}

	query := "SELECT id, name, created_at FROM test.users WHERE id = 1"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

// TestShouldReturnRows validates data retrieval for all rows.
func (suite *DuckDBTestSuite) TestShouldReturnManyRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{
		"id", "name", "created_at",
	}
	wantRows := []core.Row{
		{
			int32(1),
			"john",
			time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			int32(2),
			"bob",
			time.Date(2025, 1, 21, 0, 1, 0, 0, time.UTC),
		},
	}

	query := "SELECT id, name, created_at FROM test.users"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

// TestShouldFailInvalidQuery ensures invalid SQL fails.
func (suite *DuckDBTestSuite) TestShouldFailInvalidQuery() {
	t := suite.T()

	want := "syntax error"

	call := suite.d.Execute("INVALID SQL", func(cs core.CallState, c *core.Call) {
		if cs == core.CallStateExecutingFailed {
			assert.ErrorContains(t, c.Err(), want)
		}
	})
	assert.NotNil(t, call)
}

// TestShouldReturnColumns validates column metadata.
func (suite *DuckDBTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "id", Type: "INTEGER"},
		{Name: "name", Type: "VARCHAR"},
		{Name: "created_at", Type: "TIMESTAMP"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "users",
		Schema:          "test",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestShouldReturnStructure validates the schema structure.
func (suite *DuckDBTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	wantSchemas := []string{"test"}
	wantTables := []string{"users"}

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	gotSchemas := th.GetSchemas(t, structure)
	assert.ElementsMatch(t, wantTables, gotTables)
	assert.ElementsMatch(t, wantSchemas, gotSchemas)
}

// TestShouldFailSwitchDatabase validates error connecting to database that
// doesn't exist
func (suite *DuckDBTestSuite) TestShouldFailSwitchDatabase() {
	t := suite.T()

	want := "database switching not supported"
	// create a new connection to avoid changing the default database
	driver, err := suite.ctr.NewDriver(&core.ConnectionParams{
		ID:   "test-duckdb-2",
		Name: "test-duckdb-2",
	})
	assert.NoError(t, err)

	newDatabase := "doesnt exist"
	err = driver.SelectDatabase(newDatabase)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), want)
}
