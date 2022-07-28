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
	"errors"
	"fmt"
	"time"

	"simple-ec2/pkg/tag"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
)

const DefaultStackName = "simple-ec2"
const creationCheckInterval = time.Second
const RequiredAvailabilityZones = 3
const PostCreationWait = time.Second * 60

// Enum values for CloudFormation resource types
const (
	ResourceTypeSubnet         = "AWS::EC2::Subnet"
	ResourceTypeVpc            = "AWS::EC2::VPC"
	ResourceTypeInstance       = "AWS::EC2::Instance"
	ResourceTypeLaunchTemplate = "AWS::EC2::LaunchTemplate"
	ResourceTypeSecurityGroup  = "AWS::EC2::SecurityGroup"
)

func New(sess *session.Session) *Cfn {
	return &Cfn{
		Svc: cloudformation.New(sess),
	}
}

// Create a stack and ger resources in it, including VPC ID, subnet ID and instance ID
func (c Cfn) CreateStackAndGetResources(availabilityZones []*ec2.AvailabilityZone,
	stackName *string, template string) (vpcId *string, subnetIds []string, instanceId *string,
	stackResources []*cloudformation.StackResource, err error) {
	if stackName == nil {
		stackIdentifier := uuid.New()
		stackName = aws.String(fmt.Sprintf("%s%s", DefaultStackName, stackIdentifier))
	}

	zonesToUse := []*ec2.AvailabilityZone{}
	if availabilityZones != nil {
		for i := 0; i < RequiredAvailabilityZones; i++ {
			// Wrap around in case of the number of azs is smaller than the required number
			zonesToUse = append(zonesToUse, availabilityZones[i%len(availabilityZones)])
		}
	}

	// Create a new stack
	_, err = c.CreateStack(*stackName, template, zonesToUse)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Get all resources in the stack and select the first subnet id
	resources, err := c.GetStackResources(*stackName)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	subnetIds = []string{}
	for _, resource := range resources {
		if *resource.ResourceType == ResourceTypeVpc {
			vpcId = resource.PhysicalResourceId
		} else if *resource.ResourceType == ResourceTypeSubnet {
			subnetIds = append(subnetIds, *resource.PhysicalResourceId)
		} else if *resource.ResourceType == ResourceTypeInstance {
			instanceId = resource.PhysicalResourceId
		}
	}

	// If no VPC or subnet is available, return an error. Note that instanceId is allowed to be nil
	if vpcId == nil || len(subnetIds) <= 0 {
		return nil, nil, nil, nil, errors.New("No enough resources available in the created stack")
	}

	return vpcId, subnetIds, instanceId, resources, nil
}

// Create a stack from a cloudformation template
func (c Cfn) CreateStack(stackName, template string, zones []*ec2.AvailabilityZone) (*string, error) {
	fmt.Println("Creating CloudFormation stack...")

	input := &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(template),
		Tags:         getSimpleEc2Tags(),
	}

	if zones != nil && len(zones) > 0 {
		input.Parameters = []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("AZ0"),
				ParameterValue: zones[0].ZoneName,
			},
			{
				ParameterKey:   aws.String("AZ1"),
				ParameterValue: zones[1].ZoneName,
			},
			{
				ParameterKey:   aws.String("AZ2"),
				ParameterValue: zones[2].ZoneName,
			},
		}
	}

	output, err := c.Svc.CreateStack(input)
	if err != nil {
		return nil, err
	}

	// Keep pinging the stack on creationCheckInterval periodic, until its creation finishes or fails
	for {
		events, err := c.GetStackEventsByName(stackName)
		if err != nil {
			return nil, err
		}

		// Loop over the events and decide further actions
		ifEnd := false
		for _, event := range events {
			/*
				If the resource is the stack and status is create complete, move on.
				if the status is create failed, no matter what the resource is, throw an error with reason
			*/
			if *event.LogicalResourceId == stackName &&
				*event.ResourceStatus == cloudformation.ResourceStatusCreateComplete {
				ifEnd = true
			} else if *event.ResourceStatus == cloudformation.ResourceStatusCreateFailed {
				return nil, errors.New("Stack creation failed: " + *event.LogicalResourceId + *event.ResourceStatusReason)
			}
		}

		if ifEnd {
			break
		}

		// Sleep to prevent rate exceeded error
		time.Sleep(creationCheckInterval)
	}

	fmt.Println("CloudFormation stack", stackName, "created successfully")

	return output.StackId, nil
}

// Get the resources of a stack
func (c Cfn) GetStackResources(name string) ([]*cloudformation.StackResource, error) {
	input := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(name),
	}

	output, err := c.Svc.DescribeStackResources(input)
	if err != nil {
		return nil, err
	}
	if output.StackResources == nil || len(output.StackResources) <= 0 {
		return nil, errors.New("No resources available in stack")
	}

	return output.StackResources, nil
}

// Get the stack events by name
func (c Cfn) GetStackEventsByName(stackName string) ([]*cloudformation.StackEvent, error) {
	input := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	}

	var allEvents []*cloudformation.StackEvent

	err := c.Svc.DescribeStackEventsPages(input, func(page *cloudformation.DescribeStackEventsOutput,
		lastPage bool) bool {
		allEvents = append(allEvents, page.StackEvents...)
		return !lastPage
	})
	if err != nil {
		return nil, err
	} else if allEvents == nil || len(allEvents) <= 0 {
		return nil, errors.New("No stack events available")
	}

	return allEvents, nil
}

// Delete a stack by name
func (c Cfn) DeleteStack(stackName string) error {
	input := &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	}

	_, err := c.Svc.DeleteStack(input)
	if err != nil {
		return err
	}

	return nil
}

// Get the tags for resources created by simple-ec2
func getSimpleEc2Tags() []*cloudformation.Tag {
	simpleEc2Tags := []*cloudformation.Tag{}

	tags := tag.GetSimpleEc2Tags()
	for key, value := range *tags {
		simpleEc2Tags = append(simpleEc2Tags, &cloudformation.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	return simpleEc2Tags
}
