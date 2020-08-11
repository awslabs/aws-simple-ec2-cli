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
	cfn "github.com/aws/aws-sdk-go/service/cloudformation"
)

type MockedCfnSvc struct {
	DescribeStackEventsPagesError error
	DescribeStackResourcesError   error
	DeleteStackError              error
	CreateStackError              error
	StackEvents                   []*cfn.StackEvent
	StackResources                []*cfn.StackResource
	StackId                       *string
	EventCounter                  int
}

func (c *MockedCfnSvc) CreateStack(input *cfn.CreateStackInput) (*cfn.CreateStackOutput, error) {
	output := &cfn.CreateStackOutput{
		StackId: c.StackId,
	}
	return output, c.CreateStackError
}

func (c *MockedCfnSvc) DescribeStackResources(input *cfn.DescribeStackResourcesInput) (*cfn.DescribeStackResourcesOutput, error) {
	output := &cfn.DescribeStackResourcesOutput{
		StackResources: c.StackResources,
	}
	return output, c.DescribeStackResourcesError
}

func (c *MockedCfnSvc) DescribeStackEventsPages(input *cfn.DescribeStackEventsInput, fn func(*cfn.DescribeStackEventsOutput, bool) bool) error {
	leftBound := len(c.StackEvents) - c.EventCounter - 1
	if leftBound < 0 {
		leftBound = 0
	}

	output := &cfn.DescribeStackEventsOutput{
		StackEvents: c.StackEvents[leftBound:],
	}
	c.EventCounter++

	for {
		if !fn(output, true) {
			return c.DescribeStackEventsPagesError
		}
	}
}

func (c *MockedCfnSvc) DeleteStack(input *cfn.DeleteStackInput) (*cfn.DeleteStackOutput, error) {
	return nil, c.DeleteStackError
}
