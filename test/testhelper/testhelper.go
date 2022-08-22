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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

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

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// AssertStringEquals fails the test if the two strings don't match (case-insensitive)
func AssertStringsEqual(tb testing.TB, expected string, actual string, msg string) {
	Assert(tb, strings.EqualFold(expected, actual), fmt.Sprintf("%s (expected: %s, actual: %s)", msg, expected, actual))
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// Nok fails the test if an err is nil.
func Nok(tb testing.TB, err error) {
	if err == nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected success \033[39m\n\n", filepath.Base(file), line)
		tb.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// StringArrayEquivalent compares two string arrays and asserts that the arrays contain exactly the same items,
// ignoring order, trimming space, and comparing case-insensitive.
func StringArrayEquivalent(tb testing.TB, exp []string, act []string) {
	Assert(tb, len(exp) == len(act), fmt.Sprintf("Expected %d items, but only found %d", len(exp), len(act)))

	for _, expectedItem := range exp {
		thisItemMatches := false
		for _, actualItem := range act {
			if strings.EqualFold(strings.TrimSpace(expectedItem), strings.TrimSpace(actualItem)) {
				thisItemMatches = true
				break
			}
		}
		Assert(tb, thisItemMatches, fmt.Sprintf("Unable to find matching actual item for expected item %s", expectedItem))
	}
}
