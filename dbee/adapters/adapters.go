package adapters

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var (
	errNoValidTypeAliases   = errors.New("no valid type aliases provided")
	ErrUnsupportedTypeAlias = errors.New("no driver registered for provided type alias")
)

var _ core.Adapter = (*wrappedAdapter)(nil)

// wrappedAdapter is returned from Mux and adds extra helpers to internal adapter.
type wrappedAdapter struct {
	adapter      core.Adapter
	extraHelpers map[string]*template.Template
}

// registeredAdapters holds implemented adapters - specific adapters register themselves in their init functions.
// The main reason is to be able to compile the binary without unsupported os/arch of specific drivers.
var registeredAdapters = make(map[string]*wrappedAdapter)

// register registers a new adapter for specific database
func register(adapter core.Adapter, aliases ...string) error {
	if len(aliases) < 1 {
		return errNoValidTypeAliases
	}

	value := &wrappedAdapter{
		adapter: adapter,
	}

	invalidCount := 0
	for _, alias := range aliases {
		if alias == "" {
			invalidCount++
			continue
		}
		registeredAdapters[alias] = value
	}

	if invalidCount == len(aliases) {
		return errNoValidTypeAliases
	}

	return nil
}

// Mux is an interface to all internal adapters.
type Mux struct{}

func (*Mux) GetAdapter(typ string) (core.Adapter, error) {
	value, ok := registeredAdapters[typ]
	if !ok {
		return nil, ErrUnsupportedTypeAlias
	}

	return value, nil
}

func (*Mux) AddAdapter(typ string, adapter core.Adapter) error {
	return register(adapter, typ)
}

func (*Mux) AddHelpers(typ string, helpers map[string]string) error {
	value, ok := registeredAdapters[typ]
	if !ok {
		return ErrUnsupportedTypeAlias
	}

	if value.extraHelpers == nil {
		value.extraHelpers = make(map[string]*template.Template)
	}

	// new helpers have priority
	for k, v := range helpers {
		tmpl, err := template.New("helpers").Parse(v)
		if err != nil {
			return fmt.Errorf("template.New.Parse: %w", err)
		}

		value.extraHelpers[k] = tmpl
	}

	return nil
}

func (wa *wrappedAdapter) Connect(url string) (core.Driver, error) {
	return wa.adapter.Connect(url)
}

func (wa *wrappedAdapter) GetHelpers(opts *core.TableOptions) map[string]string {
	helpers := wa.adapter.GetHelpers(opts)
	if helpers == nil {
		helpers = make(map[string]string)
	}

	// extra helpers have priority
	for k, tmpl := range wa.extraHelpers {
		var out bytes.Buffer
		err := tmpl.Execute(&out, opts)
		if err != nil {
			continue
		}

		helpers[k] = out.String()
	}

	return helpers
}

// NewConnection is a wrapper around core.NewConnection that uses the internal mux for
// adapter registration.
func NewConnection(params *core.ConnectionParams) (*core.Connection, error) {
	adapter, err := new(Mux).GetAdapter(params.Expand().Type)
	if err != nil {
		return nil, fmt.Errorf("Mux.GetAdapters: %w", err)
	}

	c, err := core.NewConnection(params, adapter)
	if err != nil {
		return nil, fmt.Errorf("core.NewConnection: %w", err)
	}

	return c, nil
}
