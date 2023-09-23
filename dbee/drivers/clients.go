package drivers

import (
	"errors"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var (
	errNoValidTypeAliases   = errors.New("no valid type aliases provided")
	ErrUnsupportedTypeAlias = errors.New("no driver registered for provided type alias")
)

// creator creates a new driver instance
type creator func(url string) (core.Driver, error)

// registeredCreators holds implemented driver types - specific drivers register themselves in their init functions.
// The main reason is to be able to compile the binary without unsupported os/arch of specific drivers
var registeredCreators = make(map[string]creator)

// register registers a new client by submitting a creator ("new") function
func register(creator creator, aliases ...string) error {
	if len(aliases) < 1 {
		return errNoValidTypeAliases
	}

	invalidCount := 0
	for _, al := range aliases {
		if al == "" {
			invalidCount++
			continue
		}
		registeredCreators[al] = creator
	}

	if invalidCount == len(aliases) {
		return errNoValidTypeAliases
	}

	return nil
}

var _ core.Adapter = (*DefaultAdapter)(nil)

type DefaultAdapter struct{}

func Adapter() *DefaultAdapter {
	return &DefaultAdapter{}
}

func (*DefaultAdapter) Connect(typ string, url string) (core.Driver, error) {
	creator, ok := registeredCreators[typ]
	if !ok {
		return nil, ErrUnsupportedTypeAlias
	}

	driver, err := creator(url)
	if err != nil {
		return nil, fmt.Errorf("creator: %w", err)
	}

	return driver, nil
}
