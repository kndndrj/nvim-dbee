package main

import (
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	hnd "github.com/kndndrj/nvim-dbee/dbee/handler"
	"github.com/kndndrj/nvim-dbee/dbee/vim"
)

func main() {
	var entry *vim.Entrypoint
	defer func() {
		entry.Close()
		// TODO: I'm sure this can be done prettier
		time.Sleep(10 * time.Second)
	}()

	plugin.Main(func(p *plugin.Plugin) error {
		entry = vim.NewEntrypoint(p)
		handler := hnd.NewHandler(p.Nvim, vim.NewLogger(p.Nvim))

		entry.Register(
			"DbeeCreateConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID   string `arg:"id"`
				URL  string `arg:"url"`
				Type string `arg:"type"`
				Name string `arg:"name"`
			},
			) (any, error) {
				return handler.CreateConnection(&conn.Params{
					ID:   conn.ID(args.ID),
					Name: args.Name,
					Type: args.Type,
					URL:  args.URL,
				})
			}))

		entry.Register(
			"DbeeDeleteConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				handler.DeleteConnection(conn.ID(args.ID))
				return nil, nil
			}))

		entry.Register(
			"DbeeGetConnections",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				IDs []conn.ID `arg:"ids"`
			},
			) (any, error) {
				return handler.GetConnections(args.IDs), nil
			}))

		entry.Register(
			"DbeeSetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.SetCurrentConnection(conn.ID(args.ID)), nil
			}))

		entry.Register(
			"DbeeGetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct{},
			) (any, error) {
				return handler.GetCurrentConnection()
			}))

		entry.Register(
			"DbeeConnExecute",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID    string `arg:"id"`
				Query string `arg:"query"`
			},
			) (any, error) {
				return handler.ConnExecute(conn.ID(args.ID), args.Query)
			}))

		entry.Register(
			"DbeeConnGetCalls",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnGetCalls(conn.ID(args.ID))
			}))

		entry.Register(
			"DbeeConnGetStructure",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnGetStructure(conn.ID(args.ID))
			}))

		entry.Register(
			"DbeeConnListDatabases",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				current, available, err := handler.ConnListDatabases(conn.ID(args.ID))
				if err != nil {
					return nil, err
				}
				return []any{current, available}, nil
			}))

		entry.Register(
			"DbeeConnSelectDatabase",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID       string `arg:"id"`
				Database string `arg:"database"`
			},
			) (any, error) {
				return nil, handler.ConnSelectDatabase(conn.ID(args.ID), args.Database)
			}))

		// func (*Handler).CreateConnection(spec *conn.Params) (conn.ID, error)
		// func (*Handler).DeleteConnection(connID conn.ID)
		// func (*Handler).GetConnections(ids []conn.ID) []*conn.Conn
		// func (*Handler).SetCurrentConnection(connID conn.ID) error
		// func (*Handler).GetCurrentConnection() (*conn.Conn, bool)
		// func (*Handler).ConnExecute(connID conn.ID, query string) (*call.Stat, error)
		// func (*Handler).ConnGetCalls(connID conn.ID) ([]*call.Stat, error)
		// func (*Handler).ConnGetStructure(connID conn.ID) ([]models.Layout, error)
		// func (*Handler).ConnListDatabases(connID conn.ID) (current string, available []string, err error)
		// func (*Handler).ConnSelectDatabase(connID conn.ID, database string) error
		// func (*Handler).CallCancel(callID call.StatID) error
		// func (*Handler).CallDisplayResult(callID call.StatID, buffer nvim.Buffer, from int, to int) (int, error)
		// func (*Handler).CallStoreResult(callID call.StatID, fmat string, out string, from int, to int, arg ...any) error

		entry.Register(
			"DbeeCallCancel",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return nil, handler.CallCancel(call.StatID(args.ID))
			}))

		entry.Register(
			"DbeeCallDisplayResult",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID     string `arg:"id"`
				Buffer int    `arg:"buffer"`
				From   int    `arg:"from"`
				To     int    `arg:"to"`
			},
			) (any, error) {
				return handler.CallDisplayResult(call.StatID(args.ID), nvim.Buffer(args.Buffer), args.From, args.To)
			}))

		entry.Register(
			"DbeeCallStoreResult",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID     string `arg:"id"`
				Format string `arg:"format"`
				Output string `arg:"output"`
				From   int    `arg:"from"`
				To     int    `arg:"to"`
				// these two are optional (depending on the output used)
				Buffer int    `arg:"buffer,optional"`
				Path   string `arg:"path,optional"`
			},
			) (any, error) {
				var extraArg any
				if args.Output == "file" {
					extraArg = args.Path
				} else if args.Output == "buffer" {
					extraArg = args.Buffer
				}

				return nil, handler.CallStoreResult(call.StatID(args.ID), args.Format, args.Output, args.From, args.To, extraArg)
			}))

		return nil
	})
}
