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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

type MockedIAMSvc struct {
	ListInstanceProfilesError error
	InstanceProfiles          []*iam.InstanceProfile
}

func (i *MockedIAMSvc) ListInstanceProfiles(input *iam.ListInstanceProfilesInput) (*iam.ListInstanceProfilesOutput, error) {
	output := &iam.ListInstanceProfilesOutput{
		InstanceProfiles: i.InstanceProfiles,
		IsTruncated:      aws.Bool(false),
		Marker:           nil,
	}
	return output, i.ListInstanceProfilesError
}
