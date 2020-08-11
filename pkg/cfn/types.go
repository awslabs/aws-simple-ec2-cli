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
	cfn "github.com/aws/aws-sdk-go/service/cloudformation"
)

type CfnSvc interface {
	CreateStack(input *cfn.CreateStackInput) (*cfn.CreateStackOutput, error)
	DescribeStackResources(input *cfn.DescribeStackResourcesInput) (*cfn.DescribeStackResourcesOutput, error)
	DescribeStackEventsPages(input *cfn.DescribeStackEventsInput, fn func(*cfn.DescribeStackEventsOutput, bool) bool) error
	DeleteStack(input *cfn.DeleteStackInput) (*cfn.DeleteStackOutput, error)
}

type Cfn struct {
	Svc CfnSvc
}
