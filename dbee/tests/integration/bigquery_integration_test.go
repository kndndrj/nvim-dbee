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
	tc "github.com/testcontainers/testcontainers-go"
)

// BigQueryTestSuite is the test suite for the bigquery adapter.
type BigQueryTestSuite struct {
	tsuite.Suite
	ctr *th.BigQueryContainer
	ctx context.Context
	d   *core.Connection
}

// TestBigQueryTestSuite is the entrypoint for go test.
func TestBigQueryTestSuite(t *testing.T) {
	tsuite.Run(t, new(BigQueryTestSuite))
}

func (suite *BigQueryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	ctr, err := th.NewBigQueryContainer(suite.ctx, &core.ConnectionParams{
		ID:   "test-bigquery",
		Name: "test-bigquery",
	})
	if err != nil {
		log.Fatal(err)
	}

	suite.ctr = ctr
	suite.d = ctr.Driver
}

func (suite *BigQueryTestSuite) TeardownSuite() {
	tc.CleanupContainer(suite.T(), suite.ctr)
}

func (suite *BigQueryTestSuite) TestShouldErrorInvalidQuery() {
	t := suite.T()

	want := "Syntax error"

	call := suite.d.Execute("invalid sql", func(cs core.CallState, c *core.Call) {
		if cs == core.CallStateExecutingFailed {
			assert.ErrorContains(t, c.Err(), want)
		}
	})
	assert.NotNil(t, call)
}

func (suite *BigQueryTestSuite) TestShouldCancelQuery() {
	t := suite.T()
	want := []core.CallState{core.CallStateExecuting, core.CallStateCanceled}

	_, got, err := th.GetResultWithCancel(t, suite.d, "SELECT 1")
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func (suite *BigQueryTestSuite) TestShouldReturnOneRow() {
	t := suite.T()

	wantStates := []core.CallState{
		core.CallStateExecuting, core.CallStateRetrieving, core.CallStateArchived,
	}
	wantCols := []string{"id", "createdAt", "name"}
	wantRows := []core.Row{
		{
			int64(1),
			"john",
			time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC),
		},
	}

	query := "SELECT id, name, createdAt FROM `dataset_test.table_test` WHERE id = 1"

	gotRows, gotCols, gotStates, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)

	assert.ElementsMatch(t, wantCols, gotCols)
	assert.ElementsMatch(t, wantStates, gotStates)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *BigQueryTestSuite) TestShouldReturnManyRows() {
	t := suite.T()

	wantRows := []core.Row{
		{
			int64(1),
			"john",
			time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			int64(2),
			"bob",
			time.Date(2025, 1, 21, 0, 1, 0, 0, time.UTC),
		},
	}
	query := "SELECT id, name, createdAt FROM `dataset_test.table_test` WHERE id IN (1, 2)"

	gotRows, _, _, err := th.GetResult(t, suite.d, query)
	assert.NoError(t, err)
	assert.Equal(t, wantRows, gotRows)
}

func (suite *BigQueryTestSuite) TestShouldReturnStructure() {
	t := suite.T()

	wantSomeSchema, wantSomeTable := "dataset_test", "table_test"

	structure, err := suite.d.GetStructure()
	assert.NoError(t, err)

	gotSchemas := th.GetSchemas(t, structure)
	assert.Contains(t, gotSchemas, wantSomeSchema)

	gotTables := th.GetModels(t, structure, core.StructureTypeTable)
	assert.Contains(t, gotTables, wantSomeTable)
}

func (suite *BigQueryTestSuite) TestShouldReturnColumns() {
	t := suite.T()

	want := []*core.Column{
		{Name: "id", Type: "INTEGER"},
		{Name: "name", Type: "STRING"},
		{Name: "createdAt", Type: "TIMESTAMP"},
	}

	got, err := suite.d.GetColumns(&core.TableOptions{
		Table:           "table_test",
		Schema:          "dataset_test",
		Materialization: core.StructureTypeTable,
	})

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
