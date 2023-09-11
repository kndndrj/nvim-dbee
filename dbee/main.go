package main

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			func(args []int) error {
				method := "Dbee_set_results_buf"
				logger.Debugf("calling %q", method)
				if len(args) < 1 {
					logger.Errorf("not enough arguments passed to %q", method)
					return nil
				}

				bufferOutput.SetBuffer(nvim.Buffer(args[0]))

				logger.Debugf("%q returned successfully", method)
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_register_connection"},
			func(args []string) (bool, error) {
				method := "Dbee_register_connection"
				logger.Debugf("calling %q", method)
				if len(args) < 3 {
					logger.Errorf("not enough arguments passed to %q", method)
					return false, nil
				}

				id, url, typ := args[0], args[1], args[2]

				// Get the right client
				client, err := clients.NewFromType(url, typ)
				if err != nil {
					logger.Error(err.Error())
					return false, nil
				}

				c := conn.New(id, client, logger)

				connections[id] = c

				logger.Debugf("%q returned successfully", method)
				return true, nil
			})

		// execute the query and return the call id
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_execute"},
			func(args []string) (string, error) {
				method := "Dbee_execute"
				logger.Debug("calling " + method)
				if len(args) < 3 {
					logger.Errorf("not enough arguments passed to %q", method)
					return "", nil
				}

				id, query, callbackId := args[0], args[1], args[2]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Errorf("connection with id %q not registered", id)
					return "", nil
				}

				cb := func(details *call.CallDetails) {
					success := details.State != call.CallStateFailed

					err := callbacker.TriggerCallback(callbackId, details.ID, success, details.Took)
					if err != nil {
						logger.Error(err.Error())
					}
				}

				callID, err := c.Execute(query, cb)
				if err != nil {
					logger.Error(err.Error())
				}

				logger.Debugf("%q returned successfully", method)
				return callID, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_list_calls"},
			func(args []string) (string, error) {
				method := "Dbee_list_calls"
				logger.Debug("calling " + method)
				if len(args) < 1 {
					logger.Errorf("not enough arguments passed to %q", method)
					return "{}", nil
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Errorf("connection with id %q not registered", id)
					return "{}", nil
				}

				details := c.Calls()

				parsed, err := json.Marshal(details)
				if err != nil {
					logger.Error(err.Error())
					return "{}", nil
				}

				logger.Debugf("%q returned successfully", method)
				return string(parsed), nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_cancel_call"},
			func(args []string) error {
				method := "Dbee_cancel_call"
				logger.Debugf("calling %q", method)
				if len(args) < 2 {
					logger.Errorf("not enough arguments passed to %q", method)
					return nil
				}

				id := args[0]
				callID := args[1]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Errorf("connection with id %q not registered", id)
					return nil
				}

				c.CancelCall(callID)

				logger.Debugf("%q returned successfully", method)
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

				if err := c.SwitchDatabase(name); err != nil {
					logger.Error(err.Error())
					return nil
				}

				logger.Debug(method + " returned successfully")
				return nil
			})

		// pages result to buffer output, returns total number of rows
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_get_result"},
			func(args []string) (int, error) {
				method := "Dbee_get_result"
				logger.Debugf("calling %q", method)
				if len(args) < 4 {
					logger.Errorf("not enough arguments passed to %q", method)
					return 0, nil
				}

				id := args[0]
				callID := args[1]
				from, err := strconv.Atoi(args[2])
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}
				to, err := strconv.Atoi(args[3])
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Errorf("connection with id %q not registered", id)
					return 0, nil
				}

				length, err := c.GetResult(callID, from, to, bufferOutput)
				if err != nil {
					logger.Error(err.Error())
					return 0, nil
				}
				logger.Debugf("%q returned successfully", method)
				return length, nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_store"},
			func(v *nvim.Nvim, args []string) error {
				method := "Dbee_store"
				logger.Debugf("calling %q", method)
				if len(args) < 6 {
					logger.Errorf("not enough arguments passed to %q", method)
					return nil
				}
				id, callID, fmat, out := args[0], args[1], args[2], args[3]
				// range of rows
				from, err := strconv.Atoi(args[4])
				if err != nil {
					return err
				}
				to, err := strconv.Atoi(args[5])
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
					logger.Errorf("connection with id %q not registered", id)
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
					logger.Errorf("store output: %q is not supported", fmat)
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
					logger.Errorf("store output: %q is not supported", out)
					return nil
				}

				_, err = c.GetResult(callID, from, to, outpt)
				if err != nil {
					logger.Error(err.Error())
					return nil
				}

				logger.Debugf("%q returned successfully", method)
				return nil
			})

		// returns json string (must parse on caller side)
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_layout"},
			func(args []string) (string, error) {
				method := "Dbee_layout"
				layoutErrString := "{}"

				logger.Debugf("calling %q", method)
				if len(args) < 1 {
					logger.Errorf("not enough arguments passed to %q", method)
					return layoutErrString, nil
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					logger.Errorf("connection with id %q not registered", id)
					return layoutErrString, nil
				}

				layout, err := c.Layout()
				if err != nil {
					logger.Error(err.Error())
					return layoutErrString, nil
				}
				parsed, err := json.Marshal(layout)
				if err != nil {
					logger.Error(err.Error())
					return layoutErrString, nil
				}

				logger.Debugf("%q returned successfully", method)
				return string(parsed), nil
			})

		return nil
	})
}
