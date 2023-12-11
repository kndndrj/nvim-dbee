package main

import (
	"fmt"
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

				return hnd.WrapConnections(handler.GetConnections(is)), nil
			})

		entry.Register(
			"DbeeAddHelpers",
			func(r *vim.SharedResource, args map[string]any) (any, error) {
				t, ok := args["type"]
				if !ok {
					return nil, nil
				}
				typ, ok := t.(string)
				if !ok {
					return nil, fmt.Errorf("type not a string: %v", t)
				}

				raw, ok := args["helpers"]
				if !ok {
					return nil, nil
				}

				rawHelpers, ok := raw.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("helpers are not a string-any map: %#v", raw)
				}

				helpers := make(map[string]string)

				for k, v := range rawHelpers {

					stringV, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("value not a string: %v", v)
					}

					helpers[k] = stringV
				}

				return nil, handler.AddHelpers(typ, helpers)
			})

		entry.Register(
			"DbeeConnectionGetHelpers",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID              string `arg:"id"`
				Table           string `arg:"table,optional"`
				Schema          string `arg:"schema,optional"`
				Materialization string `arg:"materialization,optional"`
			},
			) (any, error) {
				return handler.ConnectionGetHelpers(core.ConnectionID(args.ID), &core.HelperOptions{
					Table:           args.Table,
					Schema:          args.Schema,
					Materialization: core.StructureTypeFromString(args.Materialization),
				})
			}))

		entry.Register(
			"DbeeSetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				return nil, handler.SetCurrentConnection(core.ConnectionID(args.ID))
			}))

		entry.Register(
			"DbeeGetCurrentConnection",
			vim.Wrap(func(r *vim.SharedResource, args *struct{},
			) (any, error) {
				conn, err := handler.GetCurrentConnection()
				return hnd.WrapConnection(conn), err
			}))

		entry.Register(
			"DbeeConnectionExecute",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID    string `arg:"id"`
				Query string `arg:"query"`
			},
			) (any, error) {
				call, err := handler.ConnectionExecute(core.ConnectionID(args.ID), args.Query)
				return hnd.WrapCall(call), err
			}))

		entry.Register(
			"DbeeConnectionGetCalls",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				calls, err := handler.ConnectionGetCalls(core.ConnectionID(args.ID))
				return hnd.WrapCalls(calls), err
			}))

		entry.Register(
			"DbeeConnectionGetParams",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				params, err := handler.ConnectionGetParams(core.ConnectionID(args.ID))
				return hnd.WrapConnectionParams(params), err
			}))

		entry.Register(
			"DbeeConnectionGetStructure",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				str, err := handler.ConnectionGetStructure(core.ConnectionID(args.ID))
				return hnd.WrapStructures(str), err
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
