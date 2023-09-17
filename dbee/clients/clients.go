package clients

import (
	"errors"
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

var ErrNoValidTypeAliases = errors.New("no valid type aliases provided")

// registers a new client by submitting a creator ("new") function
func (s *storage) Register(creator creator, aliases ...string) error {
	if len(aliases) < 1 {
		return ErrNoValidTypeAliases
	}

	invalidCount := 0
	for _, al := range aliases {
		if al == "" {
			invalidCount++
			continue
		}
		s.creators[al] = creator
	}

	if invalidCount == len(aliases) {
		return ErrNoValidTypeAliases
	}

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
