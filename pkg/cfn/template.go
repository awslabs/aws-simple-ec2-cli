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

package cfn

import (
	"encoding/base64"
)

// Encoded CloudFormation template strings populated by Makefile
var (
	SimpleEc2CloudformationTemplateEncoded        = "{}"
	E2eCfnTestCloudformationTemplateEncoded       = "{}"
	E2eConnectTestCloudformationTemplateEncoded   = "{}"
	E2eEc2helperTestCloudformationTemplateEncoded = "{}"
)

// Decoded CloudFormation template strings
var (
	SimpleEc2CloudformationTemplate        string
	E2eCfnTestCloudformationTemplate       string
	E2eConnectTestCloudformationTemplate   string
	E2eEc2helperTestCloudformationTemplate string
)

// Decode all encoded CloudFormation template strings into corresponding variables
func DecodeTemplateVariables() (err error) {
	templatePairs := [][]*string{
		{&SimpleEc2CloudformationTemplateEncoded, &SimpleEc2CloudformationTemplate},
		{&E2eCfnTestCloudformationTemplateEncoded, &E2eCfnTestCloudformationTemplate},
		{&E2eConnectTestCloudformationTemplateEncoded, &E2eConnectTestCloudformationTemplate},
		{&E2eEc2helperTestCloudformationTemplateEncoded, &E2eEc2helperTestCloudformationTemplate},
	}

	for _, pair := range templatePairs {
		err = decodeTemplateVariable(*pair[0], pair[1])
		if err != nil {
			return err
		}
	}

	return nil
}

// Decode an encoded template string into a decoded template string
func decodeTemplateVariable(encoded string, template *string) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	*template = string(decoded)

	return nil
}
