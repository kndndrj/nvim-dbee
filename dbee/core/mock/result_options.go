package mock

import (
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

type resultStreamConfig struct {
	nextSleep time.Duration
	meta      *core.Meta
	header    core.Header
}

type ResultStreamOption func(*resultStreamConfig)

func ResultStreamWithNextSleep(s time.Duration) ResultStreamOption {
	return func(c *resultStreamConfig) {
		c.nextSleep = s
	}
}

func ResultStreamWithMeta(meta *core.Meta) ResultStreamOption {
	return func(c *resultStreamConfig) {
		c.meta = meta
	}
}

func ResultStreamWithHeader(header core.Header) ResultStreamOption {
	return func(c *resultStreamConfig) {
		c.header = header
	}
}
