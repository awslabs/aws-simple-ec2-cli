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

package cfn_test

import (
	"testing"

	"simple-ec2/pkg/cfn"
	th "simple-ec2/test/testhelper"
)

// Backup encoded strings
var backupSimpleEc2String = cfn.SimpleEc2CloudformationTemplateEncoded

func TestDecodeTemplateVariables_Success(t *testing.T) {
	// Note that this test will fail when run through VSCode because it relies on some variable substitution
	// done by the way the makefile invokes `go test`. It should pass via `make unit-test`.
	err := cfn.DecodeTemplateVariables()
	th.Ok(t, err)
}

func TestDecodeTemplateVariables_Error(t *testing.T) {
	// Test 2: Decode error
	cfn.E2eConnectTestCloudformationTemplateEncoded = "{}"
	err := cfn.DecodeTemplateVariables()

	// Restore encoded strings
	cfn.SimpleEc2CloudformationTemplateEncoded = backupSimpleEc2String

	th.Nok(t, err)
}
