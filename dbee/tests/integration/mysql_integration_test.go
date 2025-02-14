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

// MySQLTestSuite is the test suite for the mysql adapter.
type MySQLTestSuite struct {
	tsuite.Suite
	ctr *th.MySQLContainer
	ctx context.Context
	d   *core.Connection
}

func TestMySQLTestSuite(t *testing.T) {
	tsuite.Run(t, new(MySQLTestSuite))
}

func (suite *MySQLTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewMySQLContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-mysql",
		Name: "test-mysql",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver // easier access to driver
}

func (suite *MySQLTestSuite) TeardownSuite() {
	tc.CleanupContainer(suite.T(), suite.ctr)
}

func (suite *MySQLTestSuite) TestShouldErrorInvalidQuery() {
	t := suite.T()

	want := "You have an error in your SQL syntax"

	call := suite.d.Execute("invalid sql", func(cs core.CallState, c *core.Call) {
		if cs == core.CallStateExecutingFailed {
			assert.ErrorContains(t, c.Err(), want)
		}
	})
	assert.NotNil(t, call)
}

func (suite *MySQLTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "SELECT 1")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func (suite *MySQLTestSuite) TestShouldReturnManyRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"id", "username", "email"}
	wantRows := []core.Row{
		{"1", "john_doe", "john@example.com"},
		{"2", "jane_smith", "jane@example.com"},
		{"3", "bob_wilson", "bob@example.com"},
	}

	query := "SELECT * FROM test.test_table"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *MySQLTestSuite) TestShouldReturnSingleRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"id", "username", "email"}
	wantRows := []core.Row{
		{"2", "jane_smith", "jane@example.com"},
	}

	query := "SELECT * FROM test.test_view"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *MySQLTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	wantSchemas := []string{"information_schema", "mysql", "performance_schema", "sys", "test"}
	wantSomeTable := "test_table"

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotSchemas := th.GetSchemas(t, structure)
	assert.ElementsMatch(t, wantSchemas, gotSchemas)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	assert.Contains(t, gotTables, wantSomeTable)
}

func (suite *MySQLTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "id", Type: "int unsigned"},
		{Name: "username", Type: "varchar(255)"},
		{Name: "email", Type: "varchar(255)"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "test_table",
		Schema:          "test",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
