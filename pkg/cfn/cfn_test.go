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
	"errors"
	"testing"

	"simple-ec2/pkg/cfn"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const testStackName = "TestStack"

var testCfn = &cfn.Cfn{}

func TestNew(t *testing.T) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
	c := cfn.New(sess)
	if c == nil {
		t.Error("Cfn is not created successfully")
	}
}

const testVpcId = "vpc-12345"
const testInstanceId = "i-12345"

var testSubnetIds = []string{
	"subnet-12345",
	"subnet-67890",
}

var mockedResources = []*cloudformation.StackResource{
	{
		ResourceType:       aws.String(cfn.ResourceTypeVpc),
		PhysicalResourceId: aws.String(testVpcId),
	},
	{
		ResourceType:       aws.String(cfn.ResourceTypeSubnet),
		PhysicalResourceId: aws.String(testSubnetIds[0]),
	},
	{
		ResourceType:       aws.String(cfn.ResourceTypeSubnet),
		PhysicalResourceId: aws.String(testSubnetIds[1]),
	},
	{
		ResourceType:       aws.String(cfn.ResourceTypeInstance),
		PhysicalResourceId: aws.String(testInstanceId),
	},
}

var mockedEvents = []*cloudformation.StackEvent{
	{
		LogicalResourceId: aws.String(cfn.DefaultStackName),
		ResourceStatus:    aws.String(cloudformation.ResourceStatusCreateComplete),
	},
	{
		LogicalResourceId: aws.String("Test Resource"),
		ResourceStatus:    aws.String(cloudformation.ResourceStatusCreateComplete),
	},
}

var testAzs = []*ec2.AvailabilityZone{
	{
		ZoneName: aws.String("AZ1"),
	},
	{
		ZoneName: aws.String("AZ2"),
	},
	{
		ZoneName: aws.String("AZ3"),
	},
}

func TestCreateStackAndGetResources_Success(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: mockedResources,
		StackEvents:    mockedEvents,
	}

	vpcId, subnetIds, instanceId, _, err := testCfn.CreateStackAndGetResources(testAzs, nil, "")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else {
		if *vpcId != testVpcId {
			t.Errorf(th.IncorrectValueFormat, "VPC ID", testVpcId, *vpcId)
		}
		if !th.StringSliceEqual(subnetIds, testSubnetIds) {
			t.Errorf(th.IncorrectValueFormat, "subnet IDs", testSubnetIds, subnetIds)
		}
		if *instanceId != testInstanceId {
			t.Errorf(th.IncorrectValueFormat, "instance ID", testInstanceId, *instanceId)
		}
	}
}

func TestCreateStackAndGetResources_DescribeStackEventsPagesError(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources:                mockedResources,
		StackEvents:                   mockedEvents,
		DescribeStackEventsPagesError: errors.New("Test error"),
	}

	_, _, _, _, err := testCfn.CreateStackAndGetResources(testAzs, nil, "")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateStackAndGetResources_DescribeStackResourcesError(t *testing.T) {
	// Test 3: GetStackResources error
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources:              mockedResources,
		StackEvents:                 mockedEvents,
		DescribeStackResourcesError: errors.New("Test error"),
	}

	_, _, _, _, err := testCfn.CreateStackAndGetResources(testAzs, nil, "")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateStackAndGetResources_NoSubnet(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: []*cloudformation.StackResource{
			{
				ResourceType:       aws.String(cfn.ResourceTypeVpc),
				PhysicalResourceId: aws.String("vpc-12345"),
			},
		},
		StackEvents: mockedEvents,
	}

	_, _, _, _, err := testCfn.CreateStackAndGetResources(testAzs, nil, "")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateStackAndGetResources_NoVpc(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: []*cloudformation.StackResource{
			{
				ResourceType:       aws.String(cfn.ResourceTypeSubnet),
				PhysicalResourceId: aws.String("subnet-12345"),
			},
			{
				ResourceType:       aws.String(cfn.ResourceTypeSubnet),
				PhysicalResourceId: aws.String("subnet-67890"),
			},
		},
		StackEvents: mockedEvents,
	}

	_, _, _, _, err := testCfn.CreateStackAndGetResources(testAzs, nil, "")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateStack_Success(t *testing.T) {
	// Update stack name for testing
	mockedEvents[0].SetLogicalResourceId(testStackName)

	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: mockedResources,
		StackEvents:    mockedEvents,
		StackId:        aws.String("stack-12345"),
	}

	_, err := testCfn.CreateStack(testStackName, "", testAzs)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestCreateStack_CreateStackError(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources:   mockedResources,
		StackEvents:      mockedEvents,
		StackId:          aws.String("stack-12345"),
		CreateStackError: errors.New("Test error"),
	}

	_, err := testCfn.CreateStack(testStackName, "", testAzs)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateStack_DescribeStackEventsPagesError(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources:                mockedResources,
		StackEvents:                   mockedEvents,
		StackId:                       aws.String("stack-12345"),
		DescribeStackEventsPagesError: errors.New("Test error"),
	}

	_, err := testCfn.CreateStack(testStackName, "", testAzs)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateStack_EventError(t *testing.T) {
	// Fail a resource creation
	mockedEvents[1].SetResourceStatus(cloudformation.ResourceStatusCreateFailed)
	mockedEvents[1].SetResourceStatusReason("Test failure")
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: mockedResources,
		StackEvents:    mockedEvents,
		StackId:        aws.String("stack-12345"),
	}

	_, err := testCfn.CreateStack(testStackName, "", testAzs)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetStackResources_Success(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: mockedResources,
	}

	resources, err := testCfn.GetStackResources("")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(resources, mockedResources, cloudformation.StackResource{}) {
		t.Error("Incorrect resources")
	}
}

func TestGetStackResources_DescribeStackResourcesError(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources:              mockedResources,
		DescribeStackResourcesError: errors.New("Test error"),
	}

	_, err := testCfn.GetStackResources("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetStackResources_NoResult(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackResources: []*cloudformation.StackResource{},
	}

	_, err := testCfn.GetStackResources("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetStackEventsByName_Success(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackEvents: mockedEvents,
	}

	events, err := testCfn.GetStackEventsByName("")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(events, mockedEvents[1:], cloudformation.StackEvent{}) {
		t.Error("Incorrect stack events")
	}
}

func TestGetStackEventsByName_DescribeStackEventsPagesError(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackEvents:                   mockedEvents,
		DescribeStackEventsPagesError: errors.New("Test error"),
	}

	_, err := testCfn.GetStackEventsByName("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetStackEventsByName_NoResult(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		StackEvents: []*cloudformation.StackEvent{},
	}

	_, err := testCfn.GetStackEventsByName("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestDeleteStack_Success(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{}

	err := testCfn.DeleteStack("")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestDeleteStack_DeleteStackError(t *testing.T) {
	testCfn.Svc = &th.MockedCfnSvc{
		DeleteStackError: errors.New("Test error"),
	}

	err := testCfn.DeleteStack("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}
