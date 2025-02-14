// Package testhelpers provides helpers for integration tests.
package testhelpers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	// eventBufferTime is a padding to let events come through (e.g. archived)
	eventBufferTime = 100 * time.Millisecond
	// eventTimeout is the maximum time to wait for an event to come through
	eventTimeout = 10 * time.Second
)

// errTimeOut is an error for when an event did not finish within the expected time.
var errTimeOut = fmt.Errorf("event did not finish within %v", eventTimeout)

// GetContainerProvider returns the container provider type to use for the tests.
// If we detect podman is available, we use it, otherwise we use docker.
func GetContainerProvider() testcontainers.ProviderType {
	if _, err := exec.LookPath("podman"); err == nil {
		fmt.Println("Podman detected. Remember to set TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED=true;")
		return testcontainers.ProviderPodman
	}
	return testcontainers.ProviderDocker
}

// GetResult is a helper function for calling the Execute method on a driver
// and waiting for the result to be available.
func GetResult(t *testing.T, d *core.Connection, query string) ([]core.Row, core.Header, []core.CallState, error) {
	t.Helper()

	var result *core.Result
	outStates := make([]core.CallState, 0)
	outRows := make([]core.Row, 0)

	call := d.Execute(query, func(state core.CallState, c *core.Call) {
		outStates = append(outStates, state)

		var err error
		if state == core.CallStateArchived || state == core.CallStateRetrieving {
			result, err = c.GetResult()
			require.NoError(t, err, "failed getting result with %s, err: %s", state, c.Err())
			outRows, err = result.Rows(0, result.Len())
			require.NoError(t, err, "failed getting rows with %s, err: %s", state, c.Err())
		}
	})

	select {
	case <-call.Done():
		time.Sleep(eventBufferTime)
		require.NotNil(t, result, call.Err())
		return outRows, result.Header(), outStates, nil

	case <-time.After(eventTimeout):
		return nil, nil, nil, errTimeOut
	}
}

// GetResultWithCancel is a helper function for calling the Execute method on a driver
// and canceling the call after the first state is received.
func GetResultWithCancel(t *testing.T, d *core.Connection, query string) (*core.Result, []core.CallState, error) {
	t.Helper()

	var (
		outResult *core.Result
		outErr    error
	)
	outStates := make([]core.CallState, 0)

	call := d.Execute(query, func(cs core.CallState, c *core.Call) {
		outStates = append(outStates, cs)
		c.Cancel()
	})

	select {
	case <-call.Done():
		time.Sleep(eventBufferTime)
		return outResult, outStates, outErr
	case <-time.After(eventTimeout):
		return nil, nil, errTimeOut
	}
}

// GetSchemas returns a list of schema names from the given structure.
func GetSchemas(t *testing.T, structure []*core.Structure) []string {
	t.Helper()

	schemas := make([]string, 0)
	for _, s := range structure {
		if s.Name == s.Schema {
			schemas = append(schemas, s.Name)
			continue
		}
	}
	return schemas
}

// GetModels returns a list of model names (views, table, etc) from the given structure.
func GetModels(t *testing.T, structure []*core.Structure, modelType core.StructureType) []string {
	t.Helper()

	out := make([]string, 0)
	for _, s := range structure {
		for _, c := range s.Children {
			if c.Type == modelType {
				out = append(out, c.Name)
				continue
			}
		}
	}
	return out
}

// GetTestDataPath returns the path to the testdata directory.
func GetTestDataPath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	return filepath.Join(filepath.Dir(currentFile), "../testdata"), nil
}

// GetTestDataFile returns a file from the testdata directory.
func GetTestDataFile(filename string) (*os.File, error) {
	testDataPath, err := GetTestDataPath()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(testDataPath, filename)
	return os.Open(path)
}
