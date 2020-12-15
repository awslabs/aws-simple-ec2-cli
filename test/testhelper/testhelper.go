// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package testhelper

import (
	"io/ioutil"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// For more consistent error messages
const (
	ExpectErrorMsg               = "Expect an error but none is found"
	UnexpectedErrorFormat        = "No error is expected but an error is thrown: %s"
	IncorrectElementNumberFormat = "Incorrect number of %s, expected: %d, got: %d"
	IncorrectValueFormat         = "Incorrect %s, expected: %s, got: %s"
)

// Compare two string slice and decide if they have the exact same elements without order
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	compareMap := map[string]int{}

	// Add occurrence to the map
	for _, element := range a {
		// If element not present in the map, initialize counter
		_, found := compareMap[element]
		if !found {
			compareMap[element] = 0
		}

		// Add 1 occurrence
		compareMap[element]++
	}

	// Subtract occurrence from the map
	for _, element := range b {
		// If element not present in the map, return false
		_, found := compareMap[element]
		if !found {
			return false
		}

		// Minus 1 occurrence
		compareMap[element]--
	}

	// Check if all elements in the map has value 0
	for _, value := range compareMap {
		if value != 0 {
			return false
		}
	}

	return true
}

// Compare if two objects are equal using package cmp. Ignores unexported fields.
func Equal(x, y, objectType interface{}) bool {
	opts := cmp.Options{
		cmpopts.IgnoreUnexported(objectType),
	}

	return cmp.Equal(x, y, opts)
}

var reader, writer, oldStdout *os.File

// Take over Stdout for reading its value programmatically later
func TakeOverStdout() error {
	var err error
	oldStdout = os.Stdout
	reader, writer, err = os.Pipe()
	if err != nil {
		return err
	}

	os.Stdout = writer

	return nil
}

// Read output from Stdout and release it
func ReadStdout() string {
	writer.Close()
	out, _ := ioutil.ReadAll(reader)
	os.Stdout = oldStdout

	return string(out)
}

var tmpFile, oldStdin *os.File

// Take over Stdin for mocking user input
func TakeOverStdin(input string) error {
	var err error
	content := []byte(input)
	tmpFile, err = ioutil.TempFile("", "mocked_input")

	if err != nil {
		return err
	}

	if _, err := tmpFile.Write(content); err != nil {
		return err
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return err
	}

	oldStdin = os.Stdin
	os.Stdin = tmpFile

	return nil
}

// Restore Stdin from the mocked input
func RestoreStdin() {
	os.Stdin = oldStdin

	tmpFile.Close()
	os.Remove(tmpFile.Name())
}
