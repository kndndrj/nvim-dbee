package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/kndndrj/nvim-dbee/clients"
	"github.com/kndndrj/nvim-dbee/conn"
	"github.com/kndndrj/nvim-dbee/output"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

func main() {

	// TODO: find a better place for logs
	// create a log to log to right away. It will help with debugging
	l, _ := os.Create("/tmp/nvim-dbee.log")
	log.SetOutput(l)

	// Call clients from lua via randomly generated id (string)
	conns := make(map[string]*conn.Conn)

	plugin.Main(func(p *plugin.Plugin) error {

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_register_client"},
			func(v *nvim.Nvim, args []string) error {
				log.Print("calling Dbee_register_client")
				if len(args) < 3 {
					return errors.New("not enough arguments passed to Dbee_register_client")
				}

				id := args[0]
				url := args[1]
				typ := args[2]

				// Get the right client
				var client clients.Client
				var err error
				switch typ {
				case "postgres":
					client, err = clients.NewPostgres(url)
					if err != nil {
						return err
					}
				case "redis":
					client, err = clients.NewRedis(url)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("database of type \"%s\" is not supported", typ)
				}

				c := conn.New(client)

				conns[id] = c

				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_execute"},
			func(v *nvim.Nvim, args []string) error {
				log.Print("calling Dbee_execute")
				if len(args) < 2 {
					return errors.New("not enough arguments passed to Dbee_execute")
				}

				id := args[0]
				query := args[1]
				b, err := strconv.Atoi(args[2])
				bufnr := nvim.Buffer(b)
				if err != nil {
					return err
				}

				// Get the right connection
				c, ok := conns[id]
				if !ok {
					return fmt.Errorf("connection with id %s not registered", id)
				}

				out := output.NewBufferOutput(v, bufnr)

				err = c.Execute(query, out)
				return err
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_get_schema"},
			func(v *nvim.Nvim, args []string) (map[string][]string, error) {
				log.Print("calling Dbee_get_schema")
				if len(args) < 1 {
					return nil, errors.New("not enough arguments passed to Dbee_get_schema")
				}

				id := args[0]

				// Get the right connection
				c, ok := conns[id]
				if !ok {
					return nil, fmt.Errorf("connection with id %s not registered", id)
				}

				schema, err := c.Schema()
				if err != nil {
					return nil, err
				}

				return schema, err
			})

		return nil
	})

	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()
}
