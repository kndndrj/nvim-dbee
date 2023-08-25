package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	"github.com/kndndrj/nvim-dbee/dbee/output/format"
	"github.com/kndndrj/nvim-dbee/dbee/vim"
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
		logger := vim.NewLogger(p.Nvim)
		callbacker := vim.NewCallbacker(p.Nvim)

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

		bufferOutput := output.NewBuffer(p.Nvim, format.NewTable(), -1)

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
				blockUntil, err := strconv.Atoi(args[3])
				if err != nil {
					logger.Error(err.Error())
					return false, nil
				}

				// Get the right client
				client, err := clients.NewFromType(url, typ)
				if err != nil {
					logger.Error(err.Error())
					return false, nil
				}

				h := conn.NewHistory(id, logger)

				c := conn.New(client, blockUntil, h, logger)

				connections[id] = c

				logger.Debug(method + " returned successfully")
				return true, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_execute"},
			func(args []string) error {
				method := "Dbee_execute"
				logger.Debug("calling " + method)
				if len(args) < 3 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				query := args[1]
				callbackId := args[2]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				// execute and open the first page
				go func() {
					ok := true
					err := c.Execute(query)
					if err != nil {
						ok = false
						logger.Error(err.Error())
					}
					err = callbacker.TriggerCallback(callbackId, ok)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					logger.Debug(method + " executed successfully")
				}()

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_history"},
			func(args []string) error {
				method := "Dbee_history"
				logger.Debug("calling " + method)
				if len(args) < 3 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				historyId := args[1]
				callbackId := args[2]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				go func() {
					ok := true
					err := c.History(historyId)
					if err != nil {
						ok = false
						logger.Error(err.Error())
					}
					err = callbacker.TriggerCallback(callbackId, ok)
					if err != nil {
						logger.Error(err.Error())
						return
					}
					logger.Debug(method + " executed successfully")
				}()

				logger.Debug(method + " returned successfully")
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_switch_database"},
			func(args []string) error {
				method := "Dbee_switch_database"
				logger.Debug("calling " + method)
				if len(args) < 2 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}

				id := args[0]
				name := args[1]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				err := c.SwitchDatabase(name)
				if err != nil {
					logger.Error(err.Error())
					return nil
				}

				logger.Debug(method + " returned successfully")
				return nil
			})

		// pages result to buffer output, returns total number of rows
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_get_current_result"},
			func(args []string) (int, error) {
				method := "Dbee_page"
				logger.Debug("calling " + method)
				if len(args) < 3 {
					logger.Error("not enough arguments passed to " + method)
					return 0, nil
				}

				id := args[0]
				from, err := strconv.Atoi(args[1])
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}
				to, err := strconv.Atoi(args[2])
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

				length, err := c.GetCurrentResult(from, to, bufferOutput)
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}
				logger.Debug(method + " returned successfully")
				return length, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_store"},
			func(v *nvim.Nvim, args []string) error {
				method := "Dbee_store"
				logger.Debug("calling " + method)
				if len(args) < 5 {
					logger.Error("not enough arguments passed to " + method)
					return nil
				}
				id := args[0]
				// format
				fmat := args[1]
				// output
				out := args[2]
				// range of rows
				from, err := strconv.Atoi(args[3])
				if err != nil {
					return err
				}
				to, err := strconv.Atoi(args[4])
				if err != nil {
					return err
				}
				// param is an extra parameter for some outputs/formatters
				param := ""
				if len(args) >= 6 {
					param = args[5]
				}

				getBufnr := func(p string) (nvim.Buffer, error) {
					b, err := strconv.Atoi(p)
					if err != nil {
						return -1, err
					}
					return nvim.Buffer(b), nil
				}

				getFile := func(p string) (string, error) {
					if p == "" {
						return "", fmt.Errorf("invalid file name: \"\"")
					}
					return p, nil
				}

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Error("connection with id " + id + " not registered")
					return nil
				}

				var formatter output.Formatter
				switch fmat {
				case "json":
					formatter = format.NewJSON()
				case "csv":
					formatter = format.NewCSV()
				case "table":
					formatter = format.NewTable()
				default:
					logger.Error("store format: \"" + fmat + "\" is not supported")
					return nil
				}

				var outpt conn.Output
				switch out {
				case "file":
					file, err := getFile(param)
					if err != nil {
						logger.Error(err.Error())
						return nil
					}
					outpt = output.NewFile(file, formatter, logger)
				case "buffer":
					buf, err := getBufnr(param)
					if err != nil {
						logger.Error(err.Error())
						return nil
					}
					outpt = output.NewBuffer(v, formatter, buf)
				case "yank":
					outpt = output.NewYankRegister(v, formatter)
				default:
					logger.Error("store output: \"" + out + "\" is not supported")
					return nil
				}

				_, err = c.GetCurrentResult(from, to, outpt)

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
