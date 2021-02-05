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
	actualShowError := cli.ShowError(nil, "Test error shown")
	th.Assert(t, actualShowError == false, "No error from cli but returns true")
}

func TestShowError_Error(t *testing.T) {
	preErrMsg := "Test error shown"
	errMsg := "Test error"
	correctOutput := fmt.Sprintf("%s: %s\n", preErrMsg, errMsg)
	err := th.TakeOverStdout()
	th.Ok(t, err)

	testErr := errors.New(errMsg)
	isError := cli.ShowError(testErr, preErrMsg)
	output := th.ReadStdout()

	th.Equals(t, true, isError)
	th.Equals(t, correctOutput, output)
}
