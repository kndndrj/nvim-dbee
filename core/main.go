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

				h := conn.NewHistory()

				c := conn.New(client, 100, h)

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

				// Get the right connection
				c, ok := conns[id]
				if !ok {
					return fmt.Errorf("connection with id %s not registered", id)
				}

				return c.Execute(query)
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_history"},
			func(v *nvim.Nvim, args []string) error {
				log.Print("calling Dbee_history")
				if len(args) < 2 {
					return errors.New("not enough arguments passed to Dbee_history")
				}

				id := args[0]
				historyId := args[1]

				// Get the right connection
				c, ok := conns[id]
				if !ok {
					return fmt.Errorf("connection with id %s not registered", id)
				}

				return c.History(historyId)
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_list_history"},
			func(v *nvim.Nvim, args []string) ([]string, error) {
				log.Print("calling Dbee_list_history")
				if len(args) < 1 {
					return nil, errors.New("not enough arguments passed to Dbee_list_history")
				}

				id := args[0]

				// Get the right connection
				c, ok := conns[id]
				if !ok {
					return nil, fmt.Errorf("connection with id %s not registered", id)
				}

				return c.ListHistory(), nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_display"},
			func(v *nvim.Nvim, args []string) (int, error) {
				log.Print("calling Dbee_display")
				if len(args) < 3 {
					return 0, errors.New("not enough arguments passed to Dbee_display")
				}

				id := args[0]
				page, err := strconv.Atoi(args[1])
				if err != nil {
					return 0, err
				}
				b, err := strconv.Atoi(args[2])
				if err != nil {
					return 0, err
				}
				bufnr := nvim.Buffer(b)

				// Get the right connection
				c, ok := conns[id]
				if !ok {
					return 0, fmt.Errorf("connection with id %s not registered", id)
				}

				out := output.NewBufferOutput(v, bufnr)

				return c.Display(page, out)
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
