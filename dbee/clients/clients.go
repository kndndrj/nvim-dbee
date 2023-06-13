package clients

import (
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
)

type creator func(url string) (conn.Client, error)

// storage holds implmented client types - specific clients register themselves in their init functions.
// The main idea behind this is to compile the binary without unsupported clients for os or arch
type storage struct {
	creators map[string]creator
}

// Store is an instance of the storage, available for public use
var Store = storage{creators: make(map[string]creator)}

// registers a new client by submitting a creator ("new") function
func (s *storage) Register(alias string, creator creator) error {
	if alias == "" {
		return fmt.Errorf("registering a client requires a valid type alias")
	}

	s.creators[alias] = creator
	return nil
}

func (s *storage) Get(alias string) (creator, error) {
	c, ok := s.creators[alias]
	if !ok {
		return nil, fmt.Errorf("no client registered with type: %s", alias)
	}
	return c, nil
}

func NewFromType(url string, typ string) (conn.Client, error) {
	new, err := Store.Get(typ)
	if err != nil {
		return nil, err
	}

	return new(url)
}
