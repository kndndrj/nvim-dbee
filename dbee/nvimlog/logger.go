package nvimlog

import (
	"log"

	"github.com/neovim/go-client/nvim"
)

type Logger struct {
	vim    *nvim.Nvim
	logger *log.Logger
}

func New(vim *nvim.Nvim, defaultLogger *log.Logger) *Logger {
	return &Logger{
		vim:    vim,
		logger: defaultLogger,
	}
}

func (l *Logger) log(level nvim.LogLevel, message string) {

	luaLevel := "vim.log.levels.INFO"
	notify := true
	switch level {
	case nvim.LogTraceLevel:
		luaLevel = "vim.log.levels.TRACE"
		notify = false
	case nvim.LogDebugLevel:
		luaLevel = "vim.log.levels.DEBUG"
		notify = false
	case nvim.LogInfoLevel:
		luaLevel = "vim.log.levels.INFO"
	case nvim.LogWarnLevel:
		luaLevel = "vim.log.levels.WARN"
	case nvim.LogErrorLevel:
		luaLevel = "vim.log.levels.ERROR"
	}

	if notify {
		// use lua so the plugins can prettify the message
		err := l.vim.ExecLua("vim.notify([["+message+"]], "+luaLevel+", { title = 'nvim-dbee' })", nil)
		if err != nil {
			l.logger.Printf("[lua log failure]: %s", err.Error())
			// fallback to go method
			err = l.vim.Notify(message, level, map[string]any{})
			if err != nil {
				l.logger.Printf("[log failure]: %s", err.Error())
			}
		}
	}

	l.logger.Printf("[%s]: %s", level.String(), message)
}

func (l *Logger) Debug(msg string) {
	l.log(nvim.LogDebugLevel, msg)
}
func (l *Logger) Info(msg string) {
	l.log(nvim.LogInfoLevel, msg)
}
func (l *Logger) Warn(msg string) {
	l.log(nvim.LogWarnLevel, msg)
}
func (l *Logger) Error(msg string) {
	l.log(nvim.LogErrorLevel, msg)
}
