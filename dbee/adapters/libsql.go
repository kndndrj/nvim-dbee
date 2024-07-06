package adapters

import (
	"database/sql"
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// Register client
func init() {
	_ = register(&LibSQL{}, "libsql", "libSQL")
}

var _ core.Adapter = (*LibSQL)(nil)

type LibSQL struct{}

func (s *LibSQL) expandPath(path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("user.Current: %w", err)
	}

	if path == "~" {
		return usr.HomeDir, nil
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(usr.HomeDir, path[2:]), nil
	}

	return path, nil
}

func (s *LibSQL) Connect(url string) (core.Driver, error) {
	path, err := s.expandPath(url)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("libsql", path)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to libsql database: %v", err)
	}

	return &libSQLDriver{
		c: builders.NewClient(db),
	}, nil
}

func (*LibSQL) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List":         fmt.Sprintf("SELECT * FROM %q LIMIT 500", opts.Table),
		"Columns":      fmt.Sprintf("PRAGMA table_info('%s')", opts.Table),
		"Indexes":      fmt.Sprintf("SELECT * FROM pragma_index_list('%s')", opts.Table),
		"Foreign Keys": fmt.Sprintf("SELECT * FROM pragma_foreign_key_list('%s')", opts.Table),
		"Primary Keys": fmt.Sprintf("SELECT * FROM pragma_index_list('%s') WHERE origin = 'pk'", opts.Table),
	}
}
