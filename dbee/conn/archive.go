package conn

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

const callArchiveBasePath = "/tmp/dbee-call-archive/"

// ScanOldCalls scans the directory to find old calls
//
// dir is a directory of a connection gob files
func ArchiveCall(connID string, ca *call.Call) error {
	dir := filepath.Join(callArchiveBasePath, connID)

	// create the directory
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	fileName := filepath.Join(dir, fmt.Sprintf("%s.gob", ca.GetDetails().ID))
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(*ca.GetDetails())
	if err != nil {
		return err
	}

	return nil
}

// ScanOldCalls scans the directory to find old calls
//
// dir is a directory of a connection gob files
func ScanOldCalls(connID string, logger models.Logger) []*call.Call {
	dir := filepath.Join(callArchiveBasePath, connID)

	// check if dir exists and is a directory
	dirInfo, err := os.Stat(dir)
	if os.IsNotExist(err) || !dirInfo.IsDir() {
		return nil
	}

	contents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	calls := []*call.Call{}

	for _, c := range contents {
		if c.IsDir() {
			continue
		}

		fileName := filepath.Join(dir, c.Name())
		file, err := os.Open(fileName)
		if err != nil {
			continue
		}
		defer file.Close()

		var callDetails call.CallDetails
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&callDetails)
		if err != nil {
			continue
		}

		ca := call.NewCaller(logger).FromDetails(&callDetails)

		calls = append(calls, ca)

	}

	return calls
}
