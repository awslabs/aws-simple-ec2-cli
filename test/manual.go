package main

import (
	"errors"
	"fmt"

	"ez-ec2/pkg/cli"
	th "ez-ec2/test/testhelper"
)

func main() {
	var err error
	preErrMsg := "Test error shown"
	errMsg := "Test error"
	correctOutput := fmt.Sprintf("%s: %s", preErrMsg, errMsg)

	if cli.ShowError(err, "Test error shown") {

	}

	err = errors.New(errMsg)

	th.TakeOverStdout()
	isError := cli.ShowError(err, preErrMsg)
	output := th.ReadStdout()

	if !isError {

	} else if output != correctOutput {

	}
}
