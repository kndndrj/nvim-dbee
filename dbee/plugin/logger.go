package plugin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/neovim/go-client/nvim"
)

type Logger struct {
	vim          *nvim.Nvim
	logger       *log.Logger
	file         *os.File
	triedFileSet bool
}

func NewLogger(vim *nvim.Nvim) *Logger {
	return &Logger{
		vim:          vim,
		logger:       log.New(os.Stdout, "", log.Ldate|log.Ltime),
		triedFileSet: false,
	}
}

func (l *Logger) setupFile() error {
	var fileName string
	err := l.vim.Call("stdpath", &fileName, "cache")
	if err != nil {
		return err
	}
	fileName = filepath.Join(fileName, "dbee", "dbee.log")

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}

	l.logger.SetOutput(file)
	return nil
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) log(level, message string) {
	if l.file == nil && !l.triedFileSet {
		err := l.setupFile()
		if err != nil {
			l.logger.Print(err)
		}
		l.triedFileSet = true
	}

	l.logger.Printf("[%s]: %s", level, message)
}

func (l *Logger) Infof(format string, args ...any) {
	l.log("info", fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.log("error", fmt.Sprintf(format, args...))
}
