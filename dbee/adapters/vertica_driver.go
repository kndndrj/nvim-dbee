package adapters

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	nurl "net/url"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*verticaDriver)(nil)
	_ core.DatabaseSwitcher = (*verticaDriver)(nil)
)

type verticaDriver struct {
	c   *builders.Client
	url *nurl.URL
}

func (c *verticaDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")

	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		return c.c.Exec(ctx, query)
	}

	return c.c.QueryUntilNotEmpty(ctx, query)
}

func (c *verticaDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery(`
		SELECT column_name, data_type
		FROM v_catalog.columns
		WHERE
			table_schema='%s' AND
			table_name='%s'
		`, opts.Schema, opts.Table)
}

func (c *verticaDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT table_schema, table_name, 'TABLE' as table_type FROM v_catalog.tables UNION ALL
		SELECT table_schema, table_name, 'VIEW' as table_type FROM v_catalog.views;
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return getVTStructure(rows)
}

func (c *verticaDriver) Close() {
	c.c.Close()
}

func (c *verticaDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT current_database(), database_name as datname FROM v_catalog.databases
		WHERE database_name != current_database();
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return "", nil, err
	}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		current = row[0].(string)
		available = append(available, row[1].(string))
	}

	return current, available, nil
}

func (c *verticaDriver) SelectDatabase(name string) error {
	c.url.Path = fmt.Sprintf("/%s", name)
	db, err := sql.Open("vertica", c.url.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	c.c.Swap(db)

	return nil
}

// getVTStructure fetches the layout from the vertica database.
// rows is at least 3 column wide result
func getVTStructure(rows core.ResultStream) ([]*core.Structure, error) {
	children := make(map[string][]*core.Structure)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}
		if len(row) < 3 {
			return nil, errors.New("could not retrieve structure: insufficient info")
		}

		schema, table, tableType := row[0].(string), row[1].(string), row[2].(string)

		children[schema] = append(children[schema], &core.Structure{
			Name:   table,
			Schema: schema,
			Type:   getVTStructureType(tableType),
		})
	}

	var structure []*core.Structure

	for k, v := range children {
		structure = append(structure, &core.Structure{
			Name:     k,
			Schema:   k,
			Type:     core.StructureTypeNone,
			Children: v,
		})
	}

	return structure, nil
}

// getVTStructureType returns the structure type based on the provided string.
func getVTStructureType(typ string) core.StructureType {
	switch typ {
	case "TABLE", "BASE TABLE", "FOREIGN", "FOREIGN TABLE":
		return core.StructureTypeTable
	case "VIEW", "SYSTEM VIEW":
		return core.StructureTypeView
	default:
		return core.StructureTypeNone
	}
}

// verticaJSONResponse serves as a wrapper around the json response
// to pretty-print the return values
type verticaJSONResponse struct {
	value []byte
}

func newVerticaJSONResponse(val []byte) *verticaJSONResponse {
	return &verticaJSONResponse{
		value: val,
	}
}

func (pj *verticaJSONResponse) String() string {
	var parsed bytes.Buffer
	err := json.Indent(&parsed, pj.value, "", "  ")
	if err != nil {
		return string(pj.value)
	}
	return parsed.String()
}

func (pj *verticaJSONResponse) MarshalJSON() ([]byte, error) {
	if json.Valid(pj.value) {
		return pj.value, nil
	}

	return json.Marshal(pj.value)
}

func (pj *verticaJSONResponse) GobEncode() ([]byte, error) {
	var err error
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err = encoder.Encode(pj.value)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), err
}

func (pj *verticaJSONResponse) GobDecode(buf []byte) error {
	var err error
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&pj.value)
	if err != nil {
		return err
	}
	return err
}
