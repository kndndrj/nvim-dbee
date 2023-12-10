package adapters

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	nurl "net/url"
	"strings"

	_ "github.com/lib/pq"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&Postgres{}, "postgres", "postgresql", "pg")

	// register special json response with gob
	gob.Register(&postgresJSONResponse{})
}

var _ core.Adapter = (*Postgres)(nil)

type Postgres struct{}

func (p *Postgres) Connect(url string) (core.Driver, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres database: %w", err)
	}

	jsonProcessor := func(a any) any {
		b, ok := a.([]byte)
		if !ok {
			return a
		}

		return newPostgresJSONResponse(b)
	}

	return &postgresDriver{
		c: builders.NewClient(db,
			builders.WithCustomTypeProcessor("json", jsonProcessor),
			builders.WithCustomTypeProcessor("jsonb", jsonProcessor),
		),
		url: u,
	}, nil
}

var (
	_ core.Driver           = (*postgresDriver)(nil)
	_ core.DatabaseSwitcher = (*postgresDriver)(nil)
)

type postgresDriver struct {
	c   *builders.Client
	url *nurl.URL
}

func (c *postgresDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	con, err := c.c.Conn(ctx)
	if err != nil {
		return nil, err
	}
	cb := func() {
		con.Close()
	}
	defer func() {
		if err != nil {
			cb()
		}
	}()

	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")

	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		rows, err := con.Exec(ctx, query)
		if err != nil {
			return nil, err
		}
		rows.SetCallback(cb)
		return rows, nil
	}

	rows, err := con.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(rows.Header()) == 0 {
		rows.SetCustomHeader(core.Header{"No Results"})
	}
	rows.SetCallback(cb)

	return rows, nil
}

func (c *postgresDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT table_schema, table_name, table_type FROM information_schema.tables UNION ALL
		SELECT schemaname, matviewname, 'VIEW' FROM pg_matviews;
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return getPGStructure(rows)
}

func (c *postgresDriver) Close() {
	c.c.Close()
}

func (c *postgresDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT current_database(), datname FROM pg_database
		WHERE datistemplate = false
		AND datname != current_database();
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

func (c *postgresDriver) SelectDatabase(name string) error {
	c.url.Path = fmt.Sprintf("/%s", name)
	db, err := sql.Open("postgres", c.url.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	c.c.Swap(db)

	return nil
}

// getPGStructure fetches the layout from the postgres database.
// rows is at least 3 column wide result
func getPGStructure(rows core.ResultStream) ([]*core.Structure, error) {
	children := make(map[string][]*core.Structure)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		schema, table, tableType := row[0].(string), row[1].(string), row[2].(string)

		children[schema] = append(children[schema], &core.Structure{
			Name:   table,
			Schema: schema,
			Type:   getPGStructureType(tableType),
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

// getPGStructureType returns the structure type based on the provided string.
func getPGStructureType(typ string) core.StructureType {
	switch typ {
	case "TABLE", "BASE TABLE":
		return core.StructureTypeTable
	case "VIEW":
		return core.StructureTypeView
	default:
		return core.StructureTypeNone
	}
}

// postgresJSONResponse serves as a wrapper around the json response
// to pretty-print the return values
type postgresJSONResponse struct {
	value []byte
}

func newPostgresJSONResponse(val []byte) *postgresJSONResponse {
	return &postgresJSONResponse{
		value: val,
	}
}

func (pj *postgresJSONResponse) String() string {
	var parsed bytes.Buffer
	err := json.Indent(&parsed, pj.value, "", "  ")
	if err != nil {
		return string(pj.value)
	}
	return parsed.String()
}

func (pj *postgresJSONResponse) MarshalJSON() ([]byte, error) {
	if json.Valid(pj.value) {
		return pj.value, nil
	}

	return json.Marshal(pj.value)
}

func (pj *postgresJSONResponse) GobEncode() ([]byte, error) {
	var err error
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err = encoder.Encode(pj.value)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), err
}

func (pj *postgresJSONResponse) GobDecode(buf []byte) error {
	var err error
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&pj.value)
	if err != nil {
		return err
	}
	return err
}
