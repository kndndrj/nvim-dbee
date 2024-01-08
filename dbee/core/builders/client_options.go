package builders

import "strings"

type clientConfig struct {
	typeProcessors map[string]func(any) any
}

type ClientOption func(*clientConfig)

func WithCustomTypeProcessor(typ string, fn func(any) any) ClientOption {
	return func(cc *clientConfig) {
		t := strings.ToLower(typ)
		_, ok := cc.typeProcessors[t]
		if ok {
			// processor already registered for this type
			return
		}

		cc.typeProcessors[t] = fn
	}
}
