package integration

import (
	"context"
	"log"
	"testing"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	th "github.com/kndndrj/nvim-dbee/dbee/tests/testhelpers"
	"github.com/stretchr/testify/assert"
	tsuite "github.com/stretchr/testify/suite"
	tc "github.com/testcontainers/testcontainers-go"
)

// PostgresTestSuite is the test suite for the postgres adapter.
type PostgresTestSuite struct {
	tsuite.Suite // inherit from testify suite
	// ctr is the postgres testcontainer
	ctr *th.PostgresContainer
	ctx context.Context
	// d is the postgres adapter
	d *core.Connection
}

// TestPostgresTestSuite is the entrypoint for go test.
//
// testify/suite can't handle parallel tests, see
// https://github.com/stretchr/testify/issues/934
func TestPostgresTestSuite(t *testing.T) {
	tsuite.Run(t, new(PostgresTestSuite))
}

func (suite *PostgresTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewPostgresContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-postgres",
		Name: "test-postgres",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver // easier access to driver
}

func (suite *PostgresTestSuite) TeardownSuite() {
	tc.CleanupContainer(suite.T(), suite.ctr)
}

func (suite *PostgresTestSuite) TestShouldErrorInvalidQuery() {
	t := suite.T()

	want := "syntax error"

	call := suite.d.Execute("invalid sql", func(cs core.CallState, c *core.Call) {
		if cs == core.CallStateExecutingFailed {
			assert.ErrorContains(t, c.Err(), want)
		}
	})
	assert.NotNil(t, call)
}

func (suite *PostgresTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "SELECT pg_sleep(1)")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func (suite *PostgresTestSuite) TestShouldReturnRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{
		"usename", "usesysid",
		"usecreatedb", "usesuper",
		"userepl", "usebypassrls",
		"passwd", "valuntil", "useconfig",
	}
	wantRows := []core.Row{
		{"postgres", "10", true, true, true, true, "********", nil, nil},
	}

	query := `
	SELECT *
	FROM pg_catalog.pg_user
	WHERE usename = 'postgres'
	LIMIT 1`

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *PostgresTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	// no need to check entire structure, just some key elements
	wantSchemas := []string{"pg_catalog", "information_schema"}
	wantSomeTable, wantSomeView := "pg_statistic", "pg_roles"

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotSchemas := th.GetSchemas(t, structure)
	assert.ElementsMatch(t, wantSchemas, gotSchemas)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	assert.Contains(t, gotTables, wantSomeTable)

	gotViews := th.GetModels(t, structure, core.StructureTypeView)
	assert.Contains(t, gotViews, wantSomeView)
}

func (suite *PostgresTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "usename", Type: "name"},
		{Name: "usesysid", Type: "oid"},
		{Name: "usecreatedb", Type: "boolean"},
		{Name: "usesuper", Type: "boolean"},
		{Name: "userepl", Type: "boolean"},
		{Name: "usebypassrls", Type: "boolean"},
		{Name: "passwd", Type: "text"},
		{Name: "valuntil", Type: "timestamp with time zone"},
		{Name: "useconfig", Type: "ARRAY"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "pg_user",
		Schema:          "pg_catalog",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func (suite *PostgresTestSuite) TestShouldSwitchDatabase() {
	t := suite.T()

	want := "postgres" // default database always present
	wantAllExceptCurrent := []string{"dev"}

	err := suite.d.SelectDatabase(want)
	assert.NoError(t, err)

	got, gotAllExceptCurrent, err := suite.d.ListDatabases()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, wantAllExceptCurrent, gotAllExceptCurrent)
}

func (suite *PostgresTestSuite) TestShouldFailSwitchDatabase() {
	t := suite.T()

	want := "doesnt exist"
	// create a new connection to avoid changing the default database
	driver, err := suite.ctr.NewDriver(&core.ConnectionParams{
		ID:   "test-postgres-2",
		Name: "test-postgres-2",
	})
	assert.NoError(t, err)

	err = driver.SelectDatabase(want)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), want)
}
