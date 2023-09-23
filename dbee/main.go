package main

import (
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	hnd "github.com/kndndrj/nvim-dbee/dbee/handler"
	"github.com/kndndrj/nvim-dbee/dbee/vim"
)

func main() {
	var handler *hnd.Handler
	defer func() {
		handler.Close()
		// TODO: I'm sure this can be done prettier
		time.Sleep(10 * time.Second)
	}()

	plugin.Main(func(p *plugin.Plugin) error {
		entry := vim.NewEntrypoint(p)
		handler = hnd.NewHandler(p.Nvim, vim.NewLogger(p.Nvim))

		entry.Register(
			"DbeeCreateConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID   string `arg:"id,optional"`
				URL  string `arg:"url"`
				Type string `arg:"type"`
				Name string `arg:"name"`
			},
			) (any, error) {
				return handler.CreateConnection(&core.ConnectionParams{
					ID:   core.ConnectionID(args.ID),
					Name: args.Name,
					Type: args.Type,
					URL:  args.URL,
				})
			}))

		entry.Register(
			"DbeeGetConnections",
			func(r *vim.SharedResource, args map[string]any) (any, error) {
				raw, ok := args["ids"]
				if !ok {
					return nil, nil
				}

				ids, ok := raw.([]any)
				if !ok {
					return nil, nil
				}

				is := make([]core.ConnectionID, len(ids))
				for i := range ids {
					str, ok := ids[i].(string)
					if !ok {
						continue
					}
					is[i] = core.ConnectionID(str)
				}

				return handler.GetConnections(is), nil
			})

		entry.Register(
			"DbeeSetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.SetCurrentConnection(core.ConnectionID(args.ID)), nil
			}))

		entry.Register(
			"DbeeGetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct{},
			) (any, error) {
				return handler.GetCurrentConnection()
			}))

		entry.Register(
			"DbeeConnectionExecute",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID    string `arg:"id"`
				Query string `arg:"query"`
			},
			) (any, error) {
				return handler.ConnectionExecute(core.ConnectionID(args.ID), args.Query)
			}))

		entry.Register(
			"DbeeConnectionGetCalls",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnectionGetCalls(core.ConnectionID(args.ID))
			}))

		entry.Register(
			"DbeeConnectionGetParams",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnectionGetParams(core.ConnectionID(args.ID))
			}))

		entry.Register(
			"DbeeConnectionGetStructure",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnectionGetStructure(core.ConnectionID(args.ID))
			}))

		entry.Register(
			"DbeeConnectionListDatabases",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				current, available, err := handler.ConnectionListDatabases(core.ConnectionID(args.ID))
				if err != nil {
					return nil, err
				}
				return []any{current, available}, nil
			}))

		entry.Register(
			"DbeeConnectionSelectDatabase",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID       string `arg:"id"`
				Database string `arg:"database"`
			},
			) (any, error) {
				return nil, handler.ConnectionSelectDatabase(core.ConnectionID(args.ID), args.Database)
			}))

		entry.Register(
			"DbeeCallCancel",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return nil, handler.CallCancel(core.CallID(args.ID))
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
				return handler.CallDisplayResult(core.CallID(args.ID), nvim.Buffer(args.Buffer), args.From, args.To)
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

				return nil, handler.CallStoreResult(core.CallID(args.ID), args.Format, args.Output, args.From, args.To, extraArg)
			}))

		return nil
	})
}
