package main

import (
	"fmt"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	"github.com/kndndrj/nvim-dbee/dbee/output/format"
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

		entry.Register(
			"Dbee_register_connection",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID   string `arg:"id"`
				URL  string `arg:"url"`
				Type string `arg:"type"`
			},
			) (any, error) {
				// Get the right client
				client, err := clients.NewFromType(args.URL, args.Type)
				if err != nil {
					return false, err
				}

				r.ConnectionStorage.Register(conn.New(args.ID, client, r.Logger))

				return true, nil
			}))

		entry.Register(
			"Dbee_execute",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID         string `arg:"id"`
				Query      string `arg:"query"`
				CallbackID string `arg:"callback_id"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return "", err
				}

				cb := func(details *call.CallDetails) {
					success := details.State != call.CallStateFailed

					err := r.Callbacker.TriggerCallback(args.CallbackID, details.ID, success, details.Took)
					if err != nil {
						r.Logger.Error(err.Error())
					}
				}

				callID, err := c.Execute(args.Query, cb)
				if err != nil {
					return "", err
				}

				return callID, nil
			}))

		entry.Register(
			"Dbee_list_calls",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return "{}", err
				}

				return c.Calls(), nil
			}))

		entry.Register(
			"Dbee_cancel_call",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID     string `arg:"id"`
				CallID string `arg:"call_id"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return nil, err
				}

				c.CancelCall(args.CallID)

				return nil, nil
			}))

		entry.Register(
			"Dbee_switch_database",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID   string `arg:"id"`
				Name string `arg:"name"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return nil, err
				}

				err = c.SwitchDatabase(args.Name)
				if err != nil {
					return nil, err
				}

				return nil, nil
			}))

		entry.Register(
			"Dbee_get_result",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID     string `arg:"id"`
				CallID string `arg:"call_id"`
				Buffer int    `arg:"buffer"`
				From   int    `arg:"from"`
				To     int    `arg:"to"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return 0, err
				}

				result, err := c.GetResult(args.CallID, args.From, args.To)
				if err != nil {
					return 0, err
				}

				err = r.BufferOutput.Write(result, nvim.Buffer(args.Buffer))
				if err != nil {
					return 0, err
				}

				return result.Meta().TotalLength, nil
			}))

		entry.Register(
			"Dbee_store",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID     string `arg:"id"`
				CallID string `arg:"call_id"`
				Format string `arg:"format"`
				Output string `arg:"output"`
				From   int    `arg:"from"`
				To     int    `arg:"to"`
				// these two are optional (depending on the output used)
				Buffer int    `arg:"buffer,optional"`
				Path   string `arg:"path,optional"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return nil, err
				}

				var formatter output.Formatter
				switch args.Format {
				case "json":
					formatter = format.NewJSON()
				case "csv":
					formatter = format.NewCSV()
				case "table":
					formatter = format.NewTable()
				default:
					return nil, fmt.Errorf("store output: %q is not supported", args.Format)
				}

				result, err := c.GetResult(args.CallID, args.From, args.To)
				if err != nil {
					return nil, err
				}

				switch args.Output {
				case "file":
					if args.Path == "" {
						return nil, fmt.Errorf("invalid output path")
					}
					err = output.NewFile(args.Path, formatter, r.Logger).Write(result)
					if err != nil {
						return nil, err
					}
				case "buffer":
					err = output.NewBuffer(r.Vim, formatter).Write(result, nvim.Buffer(args.Buffer))
					if err != nil {
						return nil, err
					}
				case "yank":
					err = output.NewYankRegister(r.Vim, formatter).Write(result)
					if err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("store output: %q is not supported", args.Output)
				}

				return nil, nil
			}))

		entry.Register(
			"Dbee_layout",
			vim.Wrap(func(r *vim.SharedResource, args *struct {
				ID string `arg:"id"`
			},
			) (any, error) {
				// Get the right connection
				c, err := r.ConnectionStorage.Get(args.ID)
				if err != nil {
					return nil, err
				}

				return c.Layout()
			}))

		return nil
	})
}
