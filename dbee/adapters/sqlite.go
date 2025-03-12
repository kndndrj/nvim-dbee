//go:build (darwin && (amd64 || arm64)) || (freebsd && (386 || amd64 || arm || arm64)) || (linux && (386 || amd64 || arm || arm64 || ppc64le || riscv64 || s390x)) || (netbsd && amd64) || (openbsd && (amd64 || arm64)) || (windows && (amd64 || arm64))

package adapters

import (
	"database/sql"
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&SQLite{}, "sqlite", "sqlite3")
}

var _ core.Adapter = (*SQLite)(nil)

type SQLite struct{}

func (s *SQLite) expandPath(path string) (string, error) {
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

func (s *SQLite) Connect(url string) (core.Driver, error) {
	path, err := s.expandPath(url)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to sqlite database: %v", err)
	}

	return &sqliteDriver{
		c:               builders.NewClient(db),
		currentDatabase: path,
	}, nil
}

func (*SQLite) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List":         fmt.Sprintf("SELECT * FROM %q LIMIT 500", opts.Table),
		"Columns":      fmt.Sprintf("PRAGMA table_info('%s')", opts.Table),
		"Indexes":      fmt.Sprintf("SELECT * FROM pragma_index_list('%s')", opts.Table),
		"Foreign Keys": fmt.Sprintf("SELECT * FROM pragma_foreign_key_list('%s')", opts.Table),
		"Primary Keys": fmt.Sprintf("SELECT * FROM pragma_index_list('%s') WHERE origin = 'pk'", opts.Table),
	}
}
