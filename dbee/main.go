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
				return handler.CreateConnection(&core.Params{
					ID:   core.ID(args.ID),
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
				handler.DeleteConnection(core.ID(args.ID))
				return nil, nil
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

				is := make([]core.ID, len(ids))
				for i := range ids {
					str, ok := ids[i].(string)
					if !ok {
						continue
					}
					is[i] = core.ID(str)
				}

				return handler.GetConnections(is), nil
			})

		entry.Register(
			"DbeeSetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.SetCurrentConnection(core.ID(args.ID)), nil
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
				return handler.ConnExecute(core.ID(args.ID), args.Query)
			}))

		entry.Register(
			"DbeeConnGetCalls",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnGetCalls(core.ID(args.ID))
			}))

		entry.Register(
			"DbeeConnGetParams",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnGetParams(core.ID(args.ID))
			}))

		entry.Register(
			"DbeeConnGetStructure",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return handler.ConnGetStructure(core.ID(args.ID))
			}))

		entry.Register(
			"DbeeConnListDatabases",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				current, available, err := handler.ConnListDatabases(core.ID(args.ID))
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
				return nil, handler.ConnSelectDatabase(core.ID(args.ID), args.Database)
			}))

		entry.Register(
			"DbeeCallCancel",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return nil, handler.CallCancel(core.StatID(args.ID))
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
				return handler.CallDisplayResult(core.StatID(args.ID), nvim.Buffer(args.Buffer), args.From, args.To)
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

				return nil, handler.CallStoreResult(core.StatID(args.ID), args.Format, args.Output, args.From, args.To, extraArg)
			}))

		return nil
	})
}
