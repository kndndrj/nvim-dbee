package vim

import (
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/kndndrj/nvim-dbee/dbee/output"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

type SharedResource struct {
	Logger       models.Logger
	BufferOutput *output.Buffer
	Vim          *nvim.Nvim
}

type Entrypoint struct {
	plugin *plugin.Plugin
	shared *SharedResource
}

func NewEntrypoint(p *plugin.Plugin) *Entrypoint {
	return &Entrypoint{
		plugin: p,
		shared: &SharedResource{
			Vim:    p.Nvim,
			Logger: NewLogger(p.Nvim),
		},
	}
}

func Wrap[A any](fn func(r *SharedResource, args *A) (any, error)) func(*SharedResource, map[string]any) (any, error) {
	return func(r *SharedResource, rawArgs map[string]any) (any, error) {
		functionArgs := &FuncArgs[A]{}
		functionArgs.Set(rawArgs)

		parsed, err := functionArgs.Parse()
		if err != nil {
			return nil, err
		}

		return fn(r, parsed)
	}
}

func (e *Entrypoint) Register(name string, fn func(r *SharedResource, args map[string]any) (any, error)) {
	// intermediate type to get data from HandleFunction
	type rawArgs struct {
		Args map[string]any `msgpack:",array"`
	}

	// this function is registered to the nvim plugin handler
	f := func(args *rawArgs) (any, error) {
		e.shared.Logger.Debugf("calling method %q", name)

		// this is the registered function
		res, err := fn(e.shared, args.Args)
		// TODO: propagate errors to lua?
		if err != nil {
			e.shared.Logger.Errorf("%q: %s", name, err)
			return nil, nil
		}
		e.shared.Logger.Debugf("method %q returned successfully", name)
		return res, nil
	}

	e.plugin.HandleFunction(&plugin.FunctionOptions{Name: name}, f)
}
