package plugin

import (
	"fmt"
	"text/template"
	"os"
	"reflect"
	"sort"

	"github.com/neovim/go-client/nvim"

	"github.com/kndndrj/nvim-dbee/dbee/vim"
)

// Plugin represents a remote plugin.
type Plugin struct {
	vim         *nvim.Nvim
	pluginSpecs []*pluginSpec
	log         *vim.Logger
}

// New returns an intialized plugin.
func New(v *nvim.Nvim, l *vim.Logger) *Plugin {
	return &Plugin{
		vim: v,
		log: l,
	}
}

type pluginSpec struct {
	sm   string
	Type string            `msgpack:"type"`
	Name string            `msgpack:"name"`
	Sync bool              `msgpack:"sync"`
	Opts map[string]string `msgpack:"opts"`
}

func isSync(f interface{}) bool {
	t := reflect.TypeOf(f)

	return t.Kind() == reflect.Func && t.NumOut() > 0
}

func (p *Plugin) handle(fn interface{}, spec *pluginSpec) {
	p.pluginSpecs = append(p.pluginSpecs, spec)
	if p.vim == nil {
		return
	}

	if err := p.vim.RegisterHandler(spec.sm, fn); err != nil {
		panic(err)
	}
}

// RegisterEndpoint registers fn as a handler for a vim function. The function
// signature for fn is one of
//
//	func([v *nvim.Nvim,] args {arrayType}) ({resultType}, error)
//	func([v *nvim.Nvim,] args {arrayType}) error
//
// where {arrayType} is a type that can be unmarshaled from a MessagePack
// array and {resultType} is the type of function result.
func (p *Plugin) RegisterEndpoint(name string, fn any) {
	v := reflect.ValueOf(fn)
	newFn := reflect.MakeFunc(v.Type(), func(args []reflect.Value) (results []reflect.Value) {
		p.log.Debugf("calling method %q", name)
		ret := v.Call(args)
		p.log.Debugf("method %q returned successfully", name)
		return ret
	})

	p.handle(newFn.Interface(), &pluginSpec{
		sm:   `0:function:` + name,
		Type: `function`,
		Name: name,
		Sync: isSync(fn),
		Opts: make(map[string]string),
	})
}

func (p *Plugin) Manifest(host, executable, writeTo string) error {
	// Sort for consistent order on output.
	sort.Slice(p.pluginSpecs, func(i, j int) bool {
		return p.pluginSpecs[i].sm < p.pluginSpecs[j].sm
	})

	tmpl, err := template.New("manifest.lua.tmpl").Parse(manifestLuaFile)
	if err != nil {
		return fmt.Errorf("template.New.Parse: %w", err)
	}

	outputFile, err := os.Create(writeTo)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}

	err = tmpl.Execute(outputFile, struct {
		Host       string
		Executable string
		Specs      []*pluginSpec
	}{
		Host:       host,
		Executable: executable,
		Specs:      p.pluginSpecs,
	})
	if err != nil {
		return fmt.Errorf("tmpl.Execute: %w", err)
	}

	return nil
}
