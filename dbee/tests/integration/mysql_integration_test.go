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

func (suite *MySQLTestSuite) TestShouldReturnRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"Host", "User"}
	wantRows := []core.Row{
		{"%", "root"},
		{"localhost", "root"},
	}

	query := "SELECT Host, User FROM mysql.user WHERE User = 'root'"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *MySQLTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	// no need to check entire structure, just some key elements
	wantSchemas := []string{"information_schema", "mysql", "performance_schema", "sys"}
	wantSomeTable := "sys_config"

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
		{Name: "TABLE_CATALOG", Type: "varchar(64)"},
		{Name: "TABLE_SCHEMA", Type: "varchar(64)"},
		{Name: "TABLE_NAME", Type: "varchar(64)"},
		{Name: "VIEW_DEFINITION", Type: "longtext"},
		{Name: "CHECK_OPTION", Type: "enum('NONE','LOCAL','CASCADED')"},
		{Name: "IS_UPDATABLE", Type: "enum('NO','YES')"},
		{Name: "DEFINER", Type: "varchar(288)"},
		{Name: "SECURITY_TYPE", Type: "varchar(7)"},
		{Name: "CHARACTER_SET_CLIENT", Type: "varchar(64)"},
		{Name: "COLLATION_CONNECTION", Type: "varchar(64)"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "VIEWS",
		Schema:          "information_schema",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
