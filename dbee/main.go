package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	nvimlog "github.com/kndndrj/nvim-dbee/dbee/nvimlog"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

var deferFns []func()

// use deferer to defer in main
func deferer(fn func()) {
	deferFns = append(deferFns, fn)
}

func main() {
	defer func() {
		for _, fn := range deferFns {
			fn()
		}
		// TODO: I'm sure this can be done prettier
		time.Sleep(10 * time.Second)
	}()

	plugin.Main(func(p *plugin.Plugin) error {
		logger := nvimlog.New(p.Nvim)

		deferer(func() {
			logger.Close()
		})

		// Call clients from lua via id (string)
		connections := make(map[string]*conn.Conn)

		deferer(func() {
			for _, c := range connections {
				c.Close()
			}
		})

		bufferOutput := output.NewBufferOutput(p.Nvim)

		// Control the results window
		// This must be called before bufferOutput is used
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_set_results_buf"},
			func(v *nvim.Nvim, args []int) error {
				method := "Dbee_set_results_buf"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				bufferOutput.SetBuffer(nvim.Buffer(args[0]))

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_register_connection"},
			func(args []string) (bool, error) {
				method := "Dbee_register_connection"
				logger.Debug("calling " + method)
				if len(args) < 4 {
					logger.Error("not enough arguments passed to " + method)
					return false, nil
				}

				id := args[0]
				url := args[1]
				typ := args[2]
				pageSize, err := strconv.Atoi(args[3])
				if err != nil {
					logger.Error(err.Error())
					return false, nil
				}

				// Get the right client
				var client conn.Client
				switch typ {
				case "postgres":
					client, err = clients.NewPostgres(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				case "mysql":
					client, err = clients.NewMysql(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				case "sqlite":
					client, err = clients.NewSqlite(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				case "redis":
					client, err = clients.NewRedis(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				case "mongo":
					client, err = clients.NewMongo(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				case "bigquery":
					client, err = clients.NewBigQuery(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				case "sqlserver":
					client, err = clients.NewSQLServer(url)
					if err != nil {
						logger.Error(err.Error())
						return false, nil
					}
				default:
					logger.Error("database of type \"" + typ + "\" is not supported")
					return false, nil
				}

				h := conn.NewHistory(id, logger)

				c := conn.New(client, pageSize, h, logger)

				connections[id] = c

				logger.Debug(method + " returned successfully")
				return true, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_execute"},
			func(v *nvim.Nvim, args []string) error {
				method := "Dbee_execute"
				logger.Debug("calling " + method)
				if len(args) < 2 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				query := args[1]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				// execute and open the first page
				go func() {
					err := c.Execute(query)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					_, _, err = c.PageCurrent(0, bufferOutput)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					logger.Debug(method + " finished successfully")
				}()

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_history"},
			func(args []string) error {
				method := "Dbee_history"
				logger.Debug("calling " + method)
				if len(args) < 2 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				historyId := args[1]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				go func() {
					err := c.History(historyId)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					_, _, err = c.PageCurrent(0, bufferOutput)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					logger.Debug(method + " finished successfully")
				}()

				logger.Debug(method + " returned successfully")
				return nil
			})

		// pages result to buffer output, returns current page and total pages
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_page"},
			func(args []string) ([]int, error) {
				method := "Dbee_page"
				logger.Debug("calling " + method)
				if len(args) < 2 {
					logger.Error("not enough arguments passed to " + method)
					return nil, nil
				}

				id := args[0]
				page, err := strconv.Atoi(args[1])
				if err != nil {
					logger.Error(err.Error())
					return nil, nil
				}

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil, nil
				}

				currentPage, totalPages, err := c.PageCurrent(page, bufferOutput)
				if err != nil {
					logger.Error(err.Error())
					return nil, nil
				}
				logger.Debug(method + " returned successfully")
				return []int{currentPage, totalPages}, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_save"},
			func(args []string) error {
				method := "Dbee_save"
				logger.Debug("calling " + method)
				if len(args) < 3 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				format := args[1]
				file := args[2]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				var out conn.Output
				switch format {
				case "json":
					out = output.NewJSONOutput(file, logger)
				case "csv":
					out = output.NewCSVOutput(file, logger)
				default:
					logger.Error("save format: \"" + format + "\" is not supported")
					return nil
				}
				err := c.WriteCurrent(out)
				if err != nil {
					logger.Error(err.Error())
					return nil
				}

				logger.Debug(method + " returned successfully")
				return nil
			})

		// returns json string (must parse on caller side)
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_layout"},
			func(args []string) (string, error) {
				method := "Dbee_layout"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Error("not enough arguments passed to " + method)
					return "{}", nil
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return "{}", nil
				}

				layout, err := c.Layout()
				if err != nil {
					logger.Error(err.Error())
					return "{}", nil
				}

				parsed, err := json.Marshal(layout)
				if err != nil {
					logger.Error(err.Error())
					return "{}", nil
				}

				logger.Debug(method + " returned successfully")
				return string(parsed), nil
			})

		return nil
	})
}
