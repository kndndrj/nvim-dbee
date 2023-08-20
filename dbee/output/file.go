package output

import (
	"fmt"
	"os"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type File struct {
	fileName  string
	log       models.Logger
	formatter Formatter
}

func NewFile(fileName string, formatter Formatter, logger models.Logger) *File {
	return &File{
		fileName:  fileName,
		log:       logger,
		formatter: formatter,
	}
}

func (co *File) Write(result models.Result) error {
	file, err := os.Create(co.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	err = co.formatter.Format(result, file)
	if err != nil {
		return fmt.Errorf("failed to format results as %s: %s", co.formatter.Name(), err)
	}

	co.log.Info("successfully saved " + co.formatter.Name() + " to " + co.fileName)
	return nil
}
