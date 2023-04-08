package main

import (
	"log"
	"os"
	"strconv"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	nvimlog "github.com/kndndrj/nvim-dbee/dbee/nvimlog"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

func main() {

	plugin.Main(func(p *plugin.Plugin) error {

		// TODO: find a better place for logs
		logFile, err := os.OpenFile("/tmp/nvim-dbee.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		dl := log.New(logFile, "", log.Ldate|log.Ltime)
		logger := nvimlog.New(p.Nvim, dl)

		logger.Debug("Starting up...")

		// Call clients from lua via id (string)
		connections := make(map[string]*conn.Conn)
		defer func() {
			for _, c := range connections {
				c.Close()
			}
		}()

		bufferOutput := output.NewBufferOutput(p.Nvim, "")

		// Control the results window
		// This must be called before bufferOutput is used
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_results"},
			func(v *nvim.Nvim, args []string) error {
				method := "Dbee_results"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				action := args[0]

				switch action {
				case "set":
					if len(args) >= 2 {
						bufferOutput.SetWindowCommand(args[1])
					} else {
						logger.Error("not enough arguments to " + method + " - create")
					}
					return nil
				case "open":
					err := bufferOutput.Open()
					if err != nil {
						logger.Error(err.Error())
					}
				case "close":
					err := bufferOutput.Close()
					if err != nil {
						logger.Error(err.Error())
					}
				}

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_register_connection"},
			func(args []string) error {
				method := "Dbee_register_connection"
				logger.Debug("calling " + method)
				if len(args) < 3 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				url := args[1]
				typ := args[2]

				// Get the right client
				var client conn.Client
				var err error
				switch typ {
				case "postgres":
					client, err = clients.NewPostgres(url)
					if err != nil {
						logger.Error(err.Error())
						return nil
					}
				case "mysql":
					client, err = clients.NewMysql(url)
					if err != nil {
						logger.Error(err.Error())
						return nil
					}
				case "sqlite":
					client, err = clients.NewSqlite(url)
					if err != nil {
						logger.Error(err.Error())
						return nil
					}
				case "redis":
					client, err = clients.NewRedis(url)
					if err != nil {
						logger.Error(err.Error())
						return nil
					}
				default:
					logger.Error("database of type \"" + typ + "\" is not supported")
					return nil
				}

				h := conn.NewHistory()

				c := conn.New(client, 100, h, logger)

				connections[id] = c

				logger.Debug(method + " returned successfully")
				return nil
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
					_, err = c.PageCurrent(0, bufferOutput)
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
					_, err = c.PageCurrent(0, bufferOutput)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					logger.Debug(method + " finished successfully")
				}()

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_list_history"},
			func(args []string) ([]string, error) {
				method := "Dbee_list_history"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Error("not enough arguments passed to " + method)
					return nil, nil
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil, nil
				}

				list := c.ListHistory()

				logger.Debug(method + " returned successfully")
				return list, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_page"},
			func(args []string) (int, error) {
				method := "Dbee_page"
				logger.Debug("calling " + method)
				if len(args) < 2 {
					logger.Error("not enough arguments passed to " + method)
					return 0, nil
				}

				id := args[0]
				page, err := strconv.Atoi(args[1])
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return 0, nil
				}

				currentPage, err := c.PageCurrent(page, bufferOutput)
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}
				logger.Debug(method + " returned successfully")
				return currentPage, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_save"},
			func(args []string) error {
				method := "Dbee_save"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}
				err := c.WriteCurrent(bufferOutput)
				if err != nil {
					logger.Error(err.Error())
					return nil
				}

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_schema"},
			func(args []string) (map[string][]string, error) {
				method := "Dbee_schema"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Error("not enough arguments passed to " + method)
					return nil, nil
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil, nil
				}

				schema, err := c.Schema()
				if err != nil {
					logger.Error(err.Error())
					return nil, nil
				}

				logger.Debug(method + " returned successfully")
				return schema, nil
			})

		return nil
	})
}
