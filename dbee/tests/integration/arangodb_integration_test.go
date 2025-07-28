package integration

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	tsuite "github.com/stretchr/testify/suite"
	tc "github.com/testcontainers/testcontainers-go"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	th "github.com/kndndrj/nvim-dbee/dbee/tests/testhelpers"
)

// ArangoDBTestSuite is the test suite for the ArangoDB adapter.
type BaseArangoDBTestSuite struct {
	tsuite.Suite
	ctr *th.ArangoDBContainer
	ctx context.Context
	d   *core.Connection
}

type ArangoDBTestSuite struct {
	BaseArangoDBTestSuite
}

type NonSystemArangoDBTestSuite struct {
	BaseArangoDBTestSuite
}

type PasswordlessArangoDBTestSuite struct {
	BaseArangoDBTestSuite
}

// TestArangoDBTestSuite is the entrypoint for go test.
func TestArangoDBTestSuite(t *testing.T) {
	tsuite.Run(t, new(ArangoDBTestSuite))
}

func TestNonSystemArangoDBTestSuite(t *testing.T) {
	tsuite.Run(t, new(NonSystemArangoDBTestSuite))
}

func TestPasswordlessArangoDBTestSuite(t *testing.T) {
	tsuite.Run(t, new(PasswordlessArangoDBTestSuite))
}

func (suite *BaseArangoDBTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewArangoDBContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-arangodb",
		Name: "test-arangodb",
	}, &th.ArangoDBContainerParams{
		Passwordless: false,
		DatabaseName: "_system",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver
}

func (suite *NonSystemArangoDBTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewArangoDBContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-arangodb",
		Name: "test-arangodb",
	}, &th.ArangoDBContainerParams{
		Passwordless: false,
		DatabaseName: "non-system",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver
}

func (suite *PasswordlessArangoDBTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewArangoDBContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-arangodb",
		Name: "test-arangodb",
	}, &th.ArangoDBContainerParams{
		Passwordless: true,
		DatabaseName: "_system",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver
}

func (suite *ArangoDBTestSuite) TeardownSuite() {
	tc.CleanupContainer(suite.T(), suite.ctr)
}

func (suite *ArangoDBTestSuite) TestShouldReturnOneRow() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"Results"}

	data := map[string]any{
		"_id":  "testcollection/4",
		"_key": "4",
	}

	rowData := adapters.NewArangoResponse(data)

	wantRows := []core.Row{
		{rowData},
	}

	query := `
		for n in testcollection 
			sort n._key desc 
			limit 1 
			return { _id: n._id, _key: n._key }
	`

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)
	assert.ElementsMatch(t, wantStates, gotStates)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *ArangoDBTestSuite) TestShouldReturnManyRows() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"Results"}

	first := adapters.NewArangoResponse(map[string]any{
		"name": "Jane",
	})

	second := adapters.NewArangoResponse(map[string]any{
		"name": "Joe",
	})

	third := adapters.NewArangoResponse(map[string]any{
		"name": "John",
	})

	wantRows := []core.Row{
		{first},
		{second},
		{third},
	}

	query := `
		for n in testcollection 
			sort n.name
			filter n.name != null
			return { name: n.name }
	`

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)
	assert.ElementsMatch(t, wantStates, gotStates)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *ArangoDBTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "for n in testcollection return n")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func (suite *ArangoDBTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	wantSchemas := []string{"_system"}
	wantSomeTable := "testcollection"

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotSchemas := th.GetSchemas(t, structure)
	assert.ElementsMatch(t, wantSchemas, gotSchemas)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	assert.Contains(t, gotTables, wantSomeTable)
}

func (suite *ArangoDBTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "_id", Type: "collection"},
		{Name: "_key", Type: "collection"},
		{Name: "_rev", Type: "collection"},
		{Name: "age", Type: "collection"},
		{Name: "name", Type: "collection"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "testcollection",
		Schema:          "_system",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
