// NOTE(ms): required due to https://github.com/microsoft/mssql-docker/issues/895
//go:debug x509negativeserial=1

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

// MSSQLServerTestSuite is the test suite for the MSSQLServer adapter.
type MSSQLServerTestSuite struct {
	tsuite.Suite
	ctr *th.MSSQLServerContainer
	ctx context.Context
	d   *core.Connection
}

// TestMSSQLServerTestSuite is the entrypoint for go test.
func TestMSSQLServerTestSuite(t *testing.T) {
	tsuite.Run(t, new(MSSQLServerTestSuite))
}

func (suite *MSSQLServerTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewSQLServerContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-mssql",
		Name: "test-mssql",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver
}

func (suite *MSSQLServerTestSuite) TeardownSuite() {
	tc.CleanupContainer(suite.T(), suite.ctr)
}

func (suite *MSSQLServerTestSuite) TestShouldErrorInvalidQuery() {
	t := suite.T()

	want := "Invalid column name 'invalid'."

	call := suite.d.Execute("select invalid", func(cs core.CallState, c *core.Call) {
		if cs == core.CallStateExecutingFailed {
			assert.ErrorContains(t, c.Err(), want)
		}
	})
	assert.NotNil(t, call)
}

func (suite *MSSQLServerTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "select 1")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func (suite *MSSQLServerTestSuite) TestShouldReturnSingleRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"ID", "Name", "Email"}
	wantRows := []core.Row{{int64(2), "Bob", "bob@example.com"}}

	query := "SELECT * FROM test_schema.test_view"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *MSSQLServerTestSuite) TestShouldReturnManyRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"ID", "Name", "Email"}
	wantRows := []core.Row{
		{int64(1), "Alice", "alice@example.com"},
		{int64(2), "Bob", "bob@example.com"},
	}

	query := "SELECT * FROM test_schema.test_table"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *MSSQLServerTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	// no need to check entire structure, just some key elements
	wantSomeSchemas, wantSomeTable, wantSomeView := "test_schema", "test_table", "test_view"

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotSchemas := th.GetSchemas(t, structure)
	assert.Contains(t, gotSchemas, wantSomeSchemas)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	assert.Contains(t, gotTables, wantSomeTable)

	gotViews := th.GetModels(t, structure, core.StructureTypeView)
	assert.Contains(t, gotViews, wantSomeView)
}

func (suite *MSSQLServerTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "ID", Type: "int"},
		{Name: "Name", Type: "nvarchar"},
		{Name: "Email", Type: "nvarchar"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "test_table",
		Schema:          "test_schema",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func (suite *MSSQLServerTestSuite) TestShouldSwitchDatabase() {
	t := suite.T()

	want := "master" // default database always present
	wantAllExceptCurrent := []string{"dev", "model", "msdb", "tempdb"}

	err := suite.d.SelectDatabase(want)
	assert.NoError(t, err)

	got, gotAllExceptCurrent, err := suite.d.ListDatabases()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, wantAllExceptCurrent, gotAllExceptCurrent)
}

func (suite *MSSQLServerTestSuite) TestShouldFailSwitchDatabase() {
	t := suite.T()

	want := "doesnt exist"
	driver, err := suite.ctr.NewDriver(&core.ConnectionParams{
		ID:   "test-mssql-2",
		Name: "test-mssql-2",
	})
	assert.NoError(t, err)

	err = driver.SelectDatabase(want)
	assert.Error(t, err)
	assert.ErrorContains(t, err, want)
}
