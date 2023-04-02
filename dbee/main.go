package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/kndndrj/nvim-dbee/dbee/clients"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

func main() {

	// TODO: find a better place for logs
	// create a log to log to right away. It will help with debugging
	l, _ := os.Create("/tmp/nvim-dbee.log")
	log.SetOutput(l)

	// Call clients from lua via id (string)
	connections := make(map[string]*conn.Conn)
	defer func() {
		for _, c := range connections {
			c.Close()
		}
	}()

	// TODO: do some sort of startup routine
	var bufferOutput *output.BufferOutput

	plugin.Main(func(p *plugin.Plugin) error {

		// Control the results window
		// This must be called before bufferOutput is used
		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_results"},
			func(v *nvim.Nvim, args []string) error {
				log.Print("calling Dbee_results")
				if len(args) < 1 {
					return errors.New("not enough arguments passed to Dbee_results")
				}

				action := args[0]

				if bufferOutput == nil && len(args) >= 2 {
					bufferOutput = output.NewBufferOutput(v, args[1])
				}

				switch action {
				case "create":
					return nil
				case "open":
					return bufferOutput.Open()
				case "close":
					return bufferOutput.Close()
				}

				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_register_connection"},
			func(args []string) error {
				log.Print("calling Dbee_register_connection")
				if len(args) < 3 {
					return errors.New("not enough arguments passed to Dbee_register_connection")
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

				connections[id] = c

				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_execute"},
			func(args []string) error {
				log.Print("calling Dbee_execute")
				if len(args) < 2 {
					return errors.New("not enough arguments passed to Dbee_execute")
				}

				id := args[0]
				query := args[1]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					return fmt.Errorf("connection with id %s not registered", id)
				}

				// execute and open the first page
				go func() {
					err := c.Execute(query)
					if err != nil {
						log.Print(err)
					}
					_, err = c.PageCurrent(0, bufferOutput)
					if err != nil {
						log.Print(err)
					}
				}()
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_history"},
			func(args []string) error {
				log.Print("calling Dbee_history")
				if len(args) < 2 {
					return errors.New("not enough arguments passed to Dbee_history")
				}

				id := args[0]
				historyId := args[1]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					return fmt.Errorf("connection with id %s not registered", id)
				}

				go func() {
					err := c.History(historyId)
					if err != nil {
						log.Print(err)
					}
					_, err = c.PageCurrent(0, bufferOutput)
					if err != nil {
						log.Print(err)
					}
				}()
				return nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_list_history"},
			func(args []string) ([]string, error) {
				log.Print("calling Dbee_list_history")
				if len(args) < 1 {
					return nil, errors.New("not enough arguments passed to Dbee_list_history")
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					return nil, fmt.Errorf("connection with id %s not registered", id)
				}

				return c.ListHistory(), nil
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_page"},
			func(args []string) (int, error) {
				log.Print("calling Dbee_page")
				if len(args) < 2 {
					return 0, errors.New("not enough arguments passed to Dbee_page")
				}

				id := args[0]
				page, err := strconv.Atoi(args[1])
				if err != nil {
					return 0, err
				}

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					return 0, fmt.Errorf("connection with id %s not registered", id)
				}

				return c.PageCurrent(page, bufferOutput)
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_save"},
			func(args []string) error {
				log.Print("calling Dbee_save")
				if len(args) < 1 {
					return errors.New("not enough arguments passed to Dbee_save")
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					return fmt.Errorf("connection with id %s not registered", id)
				}

				return c.WriteCurrent(bufferOutput)
			})

		p.HandleFunction(&plugin.FunctionOptions{Name: "Dbee_schema"},
			func(args []string) (map[string][]string, error) {
				log.Print("calling Dbee_schema")
				if len(args) < 1 {
					return nil, errors.New("not enough arguments passed to Dbee_schema")
				}

				id := args[0]

				// Get the right connection
				c, ok := connections[id]
				if !ok {
					return nil, fmt.Errorf("connection with id %s not registered", id)
				}

				return c.Schema()
			})

		return nil
	})
}
