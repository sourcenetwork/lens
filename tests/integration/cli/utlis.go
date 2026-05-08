// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestCase[TSource any, TResult any] struct {
	// The LensFile contents to test as a string
	LensFile string

	// The set of input values to feed into the lens pipeline
	Input []TSource

	// The set of values expected as output from the lens pipeline
	ExpectedOutput []TResult

	// The error expected to be returned during the transform
	ExpectedError string
}

var hostExecutablePaths = []string{
	getPathRelativeToProjectRoot(
		"/host-go/build/host-go.exe",
	),
}

func init() {
	if rustHostPath := os.Getenv("DEFRADB_RS_LENS_HOST_BIN"); rustHostPath != "" {
		hostExecutablePaths = append(hostExecutablePaths, rustHostPath)
	}
}

func getPathRelativeToProjectRoot(relativePath string) string {
	_, filename, _, _ := runtime.Caller(0)
	root := path.Dir(path.Dir(path.Dir(path.Dir(filename))))
	return path.Join(root, relativePath)
}

func executeTest[TSource any, TResult any](t *testing.T, testCase TestCase[TSource, TResult]) {
	tempDir := t.TempDir()
	lensFilePath := path.Join(tempDir, "lensFile.json")
	err := os.WriteFile(lensFilePath, []byte(testCase.LensFile), 0700)
	if err != nil {
		t.Fatal(err)
	}

	inputBytes, err := json.Marshal(testCase.Input)
	if err != nil {
		t.Fatal(err)
	}
	inputJson := string(inputBytes)

	for _, hostPath := range hostExecutablePaths {
		pipeLineCommand := exec.Command(hostPath, lensFilePath)

		var stderr bytes.Buffer
		pipeLineCommand.Stderr = &stderr

		stdin, err := pipeLineCommand.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}

		var stdout bytes.Buffer
		pipeLineCommand.Stdout = &stdout

		err = pipeLineCommand.Start()
		if err != nil {
			t.Fatal(err)
		}

		_, err = io.WriteString(stdin, inputJson)
		if err != nil {
			t.Fatal(err)
		}

		err = stdin.Close()
		if err != nil {
			t.Fatal(err)
		}

		err = pipeLineCommand.Wait()
		if err != nil {
			if assertError(t, testCase.ExpectedError, err, stderr) {
				return
			}
		}

		outputBytes := stdout.Bytes()

		var output []TResult
		err = json.Unmarshal(outputBytes, &output)
		if err != nil {
			t.Fatal(err)
		}

		// We could just assert on the string/byte array, but this gives us clearer errors
		assert.Equal(t, testCase.ExpectedOutput, output)
	}
}

func assertError(t *testing.T, expectedError string, err error, stderr bytes.Buffer) bool {
	stderrBytes := stderr.Bytes()
	if _, isExistErr := err.(*exec.ExitError); isExistErr {
		if expectedError != "" && strings.Contains(string(stderrBytes), expectedError) {
			return true
		}
	}

	// If it is unexpected we dont want to lose the stderr output
	_, writeErr := os.Stderr.Write(stderrBytes)
	if writeErr != nil {
		// This should never happen, if it does there is a bug in the test code
		t.Fatal(writeErr)
	}

	t.Fatal(err)
	return false
}
