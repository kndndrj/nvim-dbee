package main

import (
	"log"
	"os"
	"time"

	"github.com/neovim/go-client/nvim"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	hnd "github.com/kndndrj/nvim-dbee/dbee/handler"
	"github.com/kndndrj/nvim-dbee/dbee/plugin"
	"github.com/kndndrj/nvim-dbee/dbee/vim"
)

func main() {
	stdout := os.Stdout
	os.Stdout = os.Stderr
	log.SetFlags(0)

	v, err := nvim.New(os.Stdin, stdout, stdout, log.Printf)
	if err != nil {
		log.Fatal(err)
	}
	logger := vim.NewLogger(v)

	p := plugin.New(v, logger)
	handler := hnd.NewHandler(v, logger)
	defer func() {
		handler.Close()
		// TODO: I'm sure this can be done prettier
		time.Sleep(10 * time.Second)
	}()

	p.RegisterEndpoint(
		"DbeeCreateConnection",
		func(args *struct {
			Opts *struct {
				ID   string `msgpack:"id"`
				URL  string `msgpack:"url"`
				Type string `msgpack:"type"`
				Name string `msgpack:"name"`
			} `msgpack:",array"`
		},
		) (core.ConnectionID, error) {
			return handler.CreateConnection(&core.ConnectionParams{
				ID:   core.ConnectionID(args.Opts.ID),
				Name: args.Opts.Name,
				Type: args.Opts.Type,
				URL:  args.Opts.URL,
			})
		})

	p.RegisterEndpoint(
		"DbeeGetConnections",
		func(args *struct {
			IDs []core.ConnectionID `msgpack:",array"`
		},
		) (any, error) {
			return hnd.WrapConnections(handler.GetConnections(args.IDs)), nil
		})

	p.RegisterEndpoint(
		"DbeeAddHelpers",
		func(args *struct {
			Type    string `msgpack:",array"`
			Helpers map[string]string
		},
		) (any, error) {
			return nil, handler.AddHelpers(args.Type, args.Helpers)
		})

	p.RegisterEndpoint(
		"DbeeConnectionGetHelpers",
		func(args *struct {
			ID   string `msgpack:",array"`
			Opts *struct {
				Table           string `msgpack:"table"`
				Schema          string `msgpack:"schema"`
				Materialization string `msgpack:"materialization"`
			}
		},
		) (any, error) {
			return handler.ConnectionGetHelpers(core.ConnectionID(args.ID), &core.HelperOptions{
				Table:           args.Opts.Table,
				Schema:          args.Opts.Schema,
				Materialization: core.StructureTypeFromString(args.Opts.Materialization),
			})
		})

	p.RegisterEndpoint(
		"DbeeSetCurrentConnection",
		func(args *struct {
			ID core.ConnectionID `msgpack:",array"`
		},
		) error {
			return handler.SetCurrentConnection(args.ID)
		})

	p.RegisterEndpoint(
		"DbeeGetCurrentConnection",
		func() (any, error) {
			conn, err := handler.GetCurrentConnection()
			return hnd.WrapConnection(conn), err
		})

	p.RegisterEndpoint(
		"DbeeConnectionExecute",
		func(args *struct {
			ID    core.ConnectionID `msgpack:",array"`
			Query string
		},
		) (any, error) {
			call, err := handler.ConnectionExecute(args.ID, args.Query)
			return hnd.WrapCall(call), err
		})

	p.RegisterEndpoint(
		"DbeeConnectionGetCalls",
		func(args *struct {
			ID core.ConnectionID `msgpack:",array"`
		},
		) (any, error) {
			calls, err := handler.ConnectionGetCalls(args.ID)
			return hnd.WrapCalls(calls), err
		})

	p.RegisterEndpoint(
		"DbeeConnectionGetParams",
		func(args *struct {
			ID core.ConnectionID `msgpack:",array"`
		},
		) (any, error) {
			params, err := handler.ConnectionGetParams(args.ID)
			return hnd.WrapConnectionParams(params), err
		})

	p.RegisterEndpoint(
		"DbeeConnectionGetStructure",
		func(args *struct {
			ID core.ConnectionID `msgpack:",array"`
		},
		) (any, error) {
			str, err := handler.ConnectionGetStructure(args.ID)
			return hnd.WrapStructures(str), err
		})

	p.RegisterEndpoint(
		"DbeeConnectionListDatabases",
		func(args *struct {
			ID core.ConnectionID `msgpack:",array"`
		},
		) (any, error) {
			current, available, err := handler.ConnectionListDatabases(args.ID)
			if err != nil {
				return nil, err
			}
			return []any{current, available}, nil
		})

	p.RegisterEndpoint(
		"DbeeConnectionSelectDatabase",
		func(args *struct {
			ID       core.ConnectionID `msgpack:",array"`
			Database string
		},
		) (any, error) {
			return nil, handler.ConnectionSelectDatabase(args.ID, args.Database)
		})

	p.RegisterEndpoint(
		"DbeeCallCancel",
		func(args *struct {
			ID core.CallID `msgpack:",array"`
		},
		) (any, error) {
			return nil, handler.CallCancel(args.ID)
		})

	p.RegisterEndpoint(
		"DbeeCallDisplayResult",
		func(args *struct {
			ID   core.CallID `msgpack:",array"`
			Opts *struct {
				Buffer int `msgpack:"buffer"`
				From   int `msgpack:"from"`
				To     int `msgpack:"to"`
			}
		},
		) (any, error) {
			return handler.CallDisplayResult(args.ID, nvim.Buffer(args.Opts.Buffer), args.Opts.From, args.Opts.To)
		})

	p.RegisterEndpoint(
		"DbeeCallStoreResult",
		func(args *struct {
			ID     core.CallID `msgpack:",array"`
			Format string
			Output string
			Opts   *struct {
				From     int `msgpack:"from"`
				To       int `msgpack:"to"`
				ExtraArg any `msgpack:"buffer"`
			}
		},
		) (any, error) {
			return nil, handler.CallStoreResult(args.ID, args.Format, args.Output, args.Opts.From, args.Opts.To, args.Opts.ExtraArg)
		})

	if err := v.Serve(); err != nil {
		log.Fatal(err)
	}
}
