package adapters

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTrinoTestDriver builds a trinoDriver over a sqlmock DB, bypassing
// Connect (which needs a real DSN).
func setupTrinoTestDriver(t *testing.T, catalog string) (*trinoDriver, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Structure fans out SHOW TABLES concurrently, so the mock must not
	// require expectations in the order they were queued.
	mock.MatchExpectationsInOrder(false)

	driver := &trinoDriver{
		c:              builders.NewClient(db),
		catalog:        catalog,
		schemasCatalog: catalog,
	}

	return driver, mock
}

// structureByName looks up a schema node by name.
func structureByName(structure []*core.Structure, name string) *core.Structure {
	for _, s := range structure {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func TestTrinoDriver_StructureDegradesOnSchemaFailure(t *testing.T) {
	driver, mock := setupTrinoTestDriver(t, "lakehouse")

	mock.ExpectQuery(`SHOW SCHEMAS FROM "lakehouse"`).
		WillReturnRows(sqlmock.NewRows([]string{"Schema"}).
			AddRow("good").
			AddRow("bad"))

	mock.ExpectQuery(`SHOW TABLES FROM "lakehouse"."good"`).
		WillReturnRows(sqlmock.NewRows([]string{"Table"}).
			AddRow("t1").
			AddRow("t2"))

	mock.ExpectQuery(`SHOW TABLES FROM "lakehouse"."bad"`).
		WillReturnError(fmt.Errorf("metastore timeout"))

	got, err := driver.Structure()
	require.NoError(t, err)

	good := structureByName(got, "good")
	require.NotNil(t, good)
	assert.Len(t, good.Children, 2)

	bad := structureByName(got, "bad")
	require.NotNil(t, bad)
	assert.Len(t, bad.Children, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTrinoDriver_StructureErrorsWhenEverySchemaFails(t *testing.T) {
	driver, mock := setupTrinoTestDriver(t, "lakehouse")

	mock.ExpectQuery(`SHOW SCHEMAS FROM "lakehouse"`).
		WillReturnRows(sqlmock.NewRows([]string{"Schema"}).
			AddRow("good").
			AddRow("bad"))

	mock.ExpectQuery(`SHOW TABLES FROM "lakehouse"."good"`).
		WillReturnError(fmt.Errorf("metastore timeout"))

	mock.ExpectQuery(`SHOW TABLES FROM "lakehouse"."bad"`).
		WillReturnError(fmt.Errorf("metastore timeout"))

	got, err := driver.Structure()
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "2")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTrinoDriver_StructureDropsAllowlistOnOtherCatalog(t *testing.T) {
	driver, mock := setupTrinoTestDriver(t, "system")
	driver.schemas = []string{"marketing"}
	driver.schemasCatalog = "lakehouse"

	mock.ExpectQuery(`SHOW SCHEMAS FROM "system"`).
		WillReturnRows(sqlmock.NewRows([]string{"Schema"}).
			AddRow("information_schema").
			AddRow("runtime"))

	mock.ExpectQuery(`SHOW TABLES FROM "system"."runtime"`).
		WillReturnRows(sqlmock.NewRows([]string{"Table"}).
			AddRow("nodes"))

	got, err := driver.Structure()
	require.NoError(t, err)

	assert.Nil(t, structureByName(got, "information_schema"))

	runtime := structureByName(got, "runtime")
	require.NotNil(t, runtime)
	assert.Len(t, runtime.Children, 1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTrinoDriver_ColumnsQuotesThreeParts(t *testing.T) {
	driver, mock := setupTrinoTestDriver(t, "lakehouse")

	mock.ExpectQuery(`DESCRIBE "lakehouse"."marketing"."sends"`).
		WillReturnRows(sqlmock.NewRows([]string{"Column", "Type"}).
			AddRow("id", "bigint"))

	got, err := driver.Columns(&core.TableOptions{Schema: "marketing", Table: "sends"})
	require.NoError(t, err)
	assert.Equal(t, []*core.Column{{Name: "id", Type: "bigint"}}, got)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTrinoDriver_StructureKeepsSchemaOrder(t *testing.T) {
	driver, mock := setupTrinoTestDriver(t, "lakehouse")

	// SHOW SCHEMAS order is not alphabetical, so a sort would not produce it
	mock.ExpectQuery(`SHOW SCHEMAS FROM "lakehouse"`).
		WillReturnRows(sqlmock.NewRows([]string{"Schema"}).
			AddRow("zulu").
			AddRow("alpha").
			AddRow("mike"))

	for _, schema := range []string{"zulu", "alpha", "mike"} {
		mock.ExpectQuery(fmt.Sprintf(`SHOW TABLES FROM "lakehouse".%q`, schema)).
			WillReturnRows(sqlmock.NewRows([]string{"Table"}).AddRow(schema + "_t"))
	}

	got, err := driver.Structure()
	require.NoError(t, err)

	names := make([]string, 0, len(got))
	for _, s := range got {
		names = append(names, s.Name)
	}
	assert.Equal(t, []string{"zulu", "alpha", "mike"}, names)

	assert.NoError(t, mock.ExpectationsWereMet())
}
