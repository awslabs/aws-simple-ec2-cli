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

package cli_test

import (
	"errors"
	"fmt"
	"testing"

	"simple-ec2/pkg/cli"
	th "simple-ec2/test/testhelper"
)

func TestShowError_NoError(t *testing.T) {
	if cli.ShowError(nil, "Test error shown") {
		t.Error("No error but returns true")
	}
}

func TestShowError_Error(t *testing.T) {
	preErrMsg := "Test error shown"
	errMsg := "Test error"
	correctOutput := fmt.Sprintf("%s: %s\n", preErrMsg, errMsg)
	err := th.TakeOverStdout()
	if err != nil {
		t.Error(err)
	}

	testErr := errors.New(errMsg)
	isError := cli.ShowError(testErr, preErrMsg)
	output := th.ReadStdout()

	if !isError {
		t.Error("There is an error but returns false")
	} else if output != correctOutput {
		t.Errorf("Output incorrect, expect: %s, got: %s", correctOutput, output)
	}
}
