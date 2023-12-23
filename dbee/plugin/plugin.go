package plugin

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

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

func (spec *pluginSpec) path() string {
	if i := strings.Index(spec.sm, ":"); i > 0 {
		return spec.sm[:i]
	}

	return ""
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

type byServiceMethod []*pluginSpec

func (a byServiceMethod) Len() int           { return len(a) }
func (a byServiceMethod) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byServiceMethod) Less(i, j int) bool { return a[i].sm < a[j].sm }

func (p *Plugin) Manifest(host string) []byte {
	var buf bytes.Buffer

	// Sort for consistent order on output.
	sort.Sort(byServiceMethod(p.pluginSpecs))
	escape := strings.NewReplacer(`'`, `''`).Replace

	prevPath := ""
	for _, spec := range p.pluginSpecs {
		path := spec.path()
		if path != prevPath {
			if prevPath != "" {
				fmt.Fprintf(&buf, "\\ )")
			}
			fmt.Fprintf(&buf, "call remote#host#RegisterPlugin('%s', '%s', [\n", host, path)
			prevPath = path
		}

		sync := "0"
		if spec.Sync {
			sync = "1"
		}

		fmt.Fprintf(&buf, "\\ {'type': '%s', 'name': '%s', 'sync': %s, 'opts': {", spec.Type, spec.Name, sync)

		var keys []string
		for k := range spec.Opts {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		optDelim := ""
		for _, k := range keys {
			fmt.Fprintf(&buf, "%s'%s': '%s'", optDelim, k, escape(spec.Opts[k]))
			optDelim = ", "
		}

		fmt.Fprintf(&buf, "}},\n")
	}
	if prevPath != "" {
		fmt.Fprintf(&buf, "\\ ])\n")
	}

	return buf.Bytes()
}
