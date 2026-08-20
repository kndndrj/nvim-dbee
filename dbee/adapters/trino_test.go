package adapters

import (
	"testing"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrino_Connect(t *testing.T) {
	tests := []struct {
		name          string
		connectionURL string
		wantErr       bool
		messageErr    string
	}{
		{
			name:          "should fail with invalid url format",
			connectionURL: "://invalid",
			wantErr:       true,
			messageErr:    "failed to parse connection string",
		},
		{
			name:          "should fail with missing catalog",
			connectionURL: "https://user@host:443?accessToken=tok",
			wantErr:       true,
			messageErr:    "'?catalog=<catalog>' is missing",
		},
		{
			name:          "should fail with missing access token",
			connectionURL: "https://user@host:443?catalog=lakehouse",
			wantErr:       true,
			messageErr:    "'?accessToken=<token>' is missing",
		},
		{
			name:          "should fail with an unexpanded token template",
			connectionURL: `https://user@host:443?catalog=lakehouse&accessToken={{ exec "some-token-command" }}`,
			wantErr:       true,
			messageErr:    "was not expanded",
		},
		{
			name:          "should succeed with a valid connection",
			connectionURL: "https://user@host:443?catalog=lakehouse&accessToken=tok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := (&Trino{}).Connect(tt.connectionURL)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.messageErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestTrino_ConnectStripsSchemasParam(t *testing.T) {
	t.Parallel()

	drv, err := (&Trino{}).Connect(
		"https://user@host:443?catalog=lakehouse&accessToken=tok&schemas=marketing,product&source=nvim-dbee")
	require.NoError(t, err)

	tDrv, ok := drv.(*trinoDriver)
	require.True(t, ok, "expected *trinoDriver")

	assert.Equal(t, "lakehouse", tDrv.catalog)
	assert.Equal(t, []string{"marketing", "product"}, tDrv.schemas)
	assert.NotContains(t, tDrv.connectionURL.String(), "schemas=")
	assert.Contains(t, tDrv.connectionURL.String(), "source=nvim-dbee")
	assert.Contains(t, tDrv.connectionURL.String(), "accessToken=tok")
}

func TestParseSchemas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "single", raw: "marketing", want: []string{"marketing"}},
		{name: "trims whitespace", raw: " marketing , product ", want: []string{"marketing", "product"}},
		{name: "drops empty entries", raw: "marketing,,product,", want: []string{"marketing", "product"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSchemas(tt.raw))
		})
	}
}

func TestQuoteTrino(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "two parts", parts: []string{"marketing", "app_installs"}, want: `"marketing"."app_installs"`},
		{name: "three parts", parts: []string{"lakehouse", "marketing", "app_installs"}, want: `"lakehouse"."marketing"."app_installs"`},
		{name: "metadata table", parts: []string{"marketing", "app_installs$partitions"}, want: `"marketing"."app_installs$partitions"`},
		{name: "escapes inner quotes", parts: []string{`we"ird`}, want: `"we""ird"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, quoteTrino(tt.parts...))
		})
	}
}

func TestTrino_GetHelpers(t *testing.T) {
	t.Parallel()

	got := (&Trino{}).GetHelpers(&core.TableOptions{
		Schema:          "marketing",
		Table:           "app_installs",
		Materialization: core.StructureTypeTable,
	})

	want := map[string]string{
		"List":       `SELECT * FROM "marketing"."app_installs" LIMIT 100;`,
		"Columns":    `DESCRIBE "marketing"."app_installs";`,
		"DDL":        `SHOW CREATE TABLE "marketing"."app_installs";`,
		"Statistics": `SHOW STATS FOR "marketing"."app_installs";`,
		"Partitions": `SELECT * FROM "marketing"."app_installs$partitions";`,
		"Snapshots":  `SELECT * FROM "marketing"."app_installs$snapshots";`,
		"History":    `SELECT * FROM "marketing"."app_installs$history";`,
		"Files":      `SELECT * FROM "marketing"."app_installs$files";`,
	}

	assert.Equal(t, want, got)
}

func TestFilterSchemas(t *testing.T) {
	t.Parallel()

	all := []string{"information_schema", "marketing", "product", "risk"}

	tests := []struct {
		name  string
		allow []string
		want  []string
	}{
		{
			name:  "drops information_schema when no allowlist is set",
			allow: nil,
			want:  []string{"marketing", "product", "risk"},
		},
		{
			name:  "applies the allowlist and keeps the source order",
			allow: []string{"risk", "marketing"},
			want:  []string{"marketing", "risk"},
		},
		{
			name:  "ignores allowlist entries that do not exist",
			allow: []string{"marketing", "nope"},
			want:  []string{"marketing"},
		},
		{
			name:  "never returns information_schema even when it is allowlisted",
			allow: []string{"information_schema", "marketing"},
			want:  []string{"marketing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, filterSchemas(all, tt.allow))
		})
	}
}

func TestBuildStructure(t *testing.T) {
	t.Parallel()

	got := buildStructure([]schemaTables{
		{schema: "marketing", names: []string{"app_installs", "ltv_curves"}},
		{schema: "risk", names: []string{"scores"}},
	})

	require.Len(t, got, 2)

	// input order is preserved, so the drawer order stays stable
	marketing := got[0]
	assert.Equal(t, "marketing", marketing.Name)
	assert.Equal(t, core.StructureTypeSchema, marketing.Type)
	assert.Equal(t, "marketing", marketing.Schema)
	require.Len(t, marketing.Children, 2)
	assert.Equal(t, "app_installs", marketing.Children[0].Name)
	assert.Equal(t, "marketing", marketing.Children[0].Schema)
	assert.Equal(t, core.StructureTypeTable, marketing.Children[0].Type)

	risk := got[1]
	assert.Equal(t, "risk", risk.Name)
	assert.Len(t, risk.Children, 1)
}

func TestBuildStructureEmpty(t *testing.T) {
	t.Parallel()

	assert.Empty(t, buildStructure(nil))
}

func TestTrino_SelectDatabaseRewritesCatalog(t *testing.T) {
	t.Parallel()

	got, err := (&Trino{}).Connect(
		"https://user@host:443?catalog=lakehouse&accessToken=tok&source=nvim-dbee")
	require.NoError(t, err)

	d, ok := got.(*trinoDriver)
	require.True(t, ok, "expected *trinoDriver")

	require.NoError(t, d.SelectDatabase("hive"))

	assert.Equal(t, "hive", d.catalog)
	assert.Equal(t, "hive", d.connectionURL.Query().Get("catalog"))
	// the token and any pass-through parameters must survive the switch
	assert.Equal(t, "tok", d.connectionURL.Query().Get("accessToken"))
	assert.Equal(t, "nvim-dbee", d.connectionURL.Query().Get("source"))
}
