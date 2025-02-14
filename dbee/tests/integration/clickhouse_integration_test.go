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

// ClickHouseTestSuite is the test suite for the clickhouse adapter.
type ClickHouseTestSuite struct {
	tsuite.Suite
	ctr *th.ClickHouseContainer
	ctx context.Context
	d   *core.Connection
}

func TestClickHouseTestSuite(t *testing.T) {
	tsuite.Run(t, new(ClickHouseTestSuite))
}

func (suite *ClickHouseTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewClickHouseContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-clickhouse",
		Name: "test-clickhouse",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver
}

func (suite *ClickHouseTestSuite) TeardownSuite() {
	tc.CleanupContainer(suite.T(), suite.ctr)
}

func (suite *ClickHouseTestSuite) TestShouldErrorInvalidQuery() {
	t := suite.T()

	want := "Syntax error"

	call := suite.d.Execute("invalid sql", func(cs core.CallState, c *core.Call) {
		if cs == core.CallStateExecutingFailed {
			assert.ErrorContains(t, c.Err(), want)
		}
	})
	assert.NotNil(t, call)
}

func (suite *ClickHouseTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "SELECT 1")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func (suite *ClickHouseTestSuite) TestShouldReturnManyRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"id", "username"}
	wantRows := []core.Row{
		{uint32(1), "john_doe"},
		{uint32(2), "jane_smith"},
	}

	query := "SELECT id, username FROM test.test_view"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *ClickHouseTestSuite) TestShouldReturnOneRow() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"id", "username"}
	wantRows := []core.Row{{uint32(1), "john_doe"}}

	query := "SELECT id, username FROM test.test_table WHERE id = 1"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *ClickHouseTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	var (
		wantSomeSchema = "test"
		wantSomeTable  = "test_table"
		wantSomeView   = "test_view"
	)

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotSchemas := th.GetSchemas(t, structure)
	assert.Contains(t, gotSchemas, wantSomeSchema)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	assert.Contains(t, gotTables, wantSomeTable)

	gotViews := th.GetModels(t, structure, core.StructureTypeView)
	assert.Contains(t, gotViews, wantSomeView)
}

func (suite *ClickHouseTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "id", Type: "UInt32"},
		{Name: "username", Type: "String"},
		{Name: "email", Type: "String"},
		{Name: "created_at", Type: "DateTime"},
		{Name: "is_active", Type: "UInt8"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "test_table",
		Schema:          "test",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func (suite *ClickHouseTestSuite) TestShouldSwitchDatabase() {
	t := suite.T()

	want := "dev"
	wantAllExceptCurrent := []string{"default", "information_schema", "system", "test"}

	err := suite.d.SelectDatabase(want)
	assert.NoError(t, err)

	got, gotAllExceptCurrent, err := suite.d.ListDatabases()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, wantAllExceptCurrent, gotAllExceptCurrent)
}

func (suite *ClickHouseTestSuite) TestShouldFailSwitchDatabase() {
	t := suite.T()

	want := "doesnt exist"
	// create a new connection to avoid changing the default database
	driver, err := suite.ctr.NewDriver(&core.ConnectionParams{
		ID:   "test-clickhouse-2",
		Name: "test-clickhouse-2",
	})
	assert.NoError(t, err)

	err = driver.SelectDatabase(want)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), want)
}
