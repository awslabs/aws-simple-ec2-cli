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

package question_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/iamhelper"
	"simple-ec2/pkg/question"
	"simple-ec2/pkg/questionModel"
	th "simple-ec2/test/testhelper"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/instancetypes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	tea "github.com/charmbracelet/bubbletea"
)

var testEC2 = &ec2helper.EC2Helper{
	Sess: &session.Session{
		Config: &aws.Config{
			Region: aws.String("us-east-1"),
		},
	},
}
var testQMHelper = &questionModel.QuestionModelHelper{}
var defaultArchitecture = aws.StringSlice([]string{"x86_64"})

/*
Other Question Asking Tests
*/

func TestAskRegion_Success(t *testing.T) {
	const expectedRegion = "us-east-2"

	testEC2.Svc = &th.MockedEC2Svc{
		Regions: []*ec2.Region{
			{
				RegionName: aws.String(expectedRegion),
			},
			{
				RegionName: aws.String("us-west-1"),
			},
			{
				RegionName: aws.String("us-west-2"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskRegion(testEC2, testQMHelper, "")
	th.Ok(t, err)
	th.Equals(t, expectedRegion, *answer)
}

func TestAskRegion_DescribeRegionsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeRegionsError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskRegion(testEC2, testQMHelper, "")
	th.Nok(t, err)
}

func TestAskLaunchTemplate_Success(t *testing.T) {
	const expectedTemplateId = "lt-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{
			{
				LaunchTemplateId:    aws.String(expectedTemplateId),
				LaunchTemplateName:  aws.String(expectedTemplateId),
				LatestVersionNumber: aws.Int64(1),
			},
			{
				LaunchTemplateId:    aws.String("lt-67890"),
				LaunchTemplateName:  aws.String("lt-67890"),
				LatestVersionNumber: aws.Int64(1),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskLaunchTemplate(testEC2, testQMHelper, "")
	th.Equals(t, expectedTemplateId, *answer)

	th.Ok(t, err)
}

func TestAskLaunchTemplate_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskLaunchTemplate(testEC2, testQMHelper, "")
	th.Equals(t, cli.ResponseNo, *answer)

	th.Ok(t, err)
}

func TestAskLaunchTemplateVersion_Success(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{
			{
				LaunchTemplateId:   aws.String(testTemplateId),
				VersionDescription: aws.String("description"),
				VersionNumber:      aws.Int64(testVersion),
				DefaultVersion:     aws.Bool(true),
			},
			{
				LaunchTemplateId: aws.String(testTemplateId),
				VersionNumber:    aws.Int64(2),
				DefaultVersion:   aws.Bool(false),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskLaunchTemplateVersion(testEC2, testQMHelper, testTemplateId, "")
	th.Ok(t, err)
	th.Equals(t, strconv.Itoa(testVersion), *answer)

}

func TestAskLaunchTemplateVersion_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	const testTemplateId = "lt-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskLaunchTemplateVersion(testEC2, testQMHelper, testTemplateId, "")
	th.Nok(t, err)
}

func TestAskIfEnterInstanceType_Success(t *testing.T) {
	const expectedInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(expectedInstanceType),
				FreeTierEligible: aws.Bool(true),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskIfEnterInstanceType(testEC2, testQMHelper, "")
	th.Ok(t, err)
	th.Equals(t, expectedInstanceType, *answer)
}

func TestAskIfEnterInstanceType_(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskIfEnterInstanceType(testEC2, testQMHelper, "")
	th.Nok(t, err)
}

func TestAskInstanceType_Success(t *testing.T) {
	const expectedInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(expectedInstanceType),
				FreeTierEligible: aws.Bool(true),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune(expectedInstanceType),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskInstanceType(testEC2, testQMHelper, "")
	th.Ok(t, err)
	th.Equals(t, expectedInstanceType, *answer)
}

func TestAskInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceType(testEC2, testQMHelper, "")
	th.Nok(t, err)
}

func TestAskInstanceTypeVCpu(t *testing.T) {
	const expectedVcpus = "4"

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune(expectedVcpus),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskInstanceTypeVCpu(testEC2, testQMHelper)
	th.Equals(t, expectedVcpus, answer)

	th.Ok(t, err)
}

func TestAskInstanceTypeMemory(t *testing.T) {
	const expectedMemory = "3"

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune(expectedMemory),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskInstanceTypeMemory(testEC2, testQMHelper)
	th.Equals(t, expectedMemory, answer)

	th.Ok(t, err)
}

func TestAskImage_Success(t *testing.T) {
	const expectedImage = "ami-12345"
	const testInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:             aws.String(testInstanceType),
				InstanceStorageSupported: aws.Bool(true),
				ProcessorInfo:            &ec2.ProcessorInfo{SupportedArchitectures: defaultArchitecture},
			},
		},
		Images: []*ec2.Image{
			{
				ImageId:      aws.String(expectedImage),
				CreationDate: aws.String("some time"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskImage(testEC2, testQMHelper, testInstanceType, "")
	th.Ok(t, err)
	th.Equals(t, expectedImage, *answer.ImageId)
}

func TestAskImage_NoImage(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:             aws.String(testInstanceType),
				InstanceStorageSupported: aws.Bool(true),
				ProcessorInfo:            &ec2.ProcessorInfo{SupportedArchitectures: defaultArchitecture},
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskImage(testEC2, testQMHelper, testInstanceType, "")
	th.Nok(t, err)
}

func TestAskImage_DescribeInstanceTypesPagesError(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskImage(testEC2, testQMHelper, testInstanceType, "")
	th.Nok(t, err)
}

func TestAskImage_DescribeImagesError(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:             aws.String(testInstanceType),
				InstanceStorageSupported: aws.Bool(true),
				ProcessorInfo:            &ec2.ProcessorInfo{SupportedArchitectures: defaultArchitecture},
			},
		},
		DescribeImagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskImage(testEC2, testQMHelper, testInstanceType, "")
	th.Nok(t, err)
}

func TestAskKeepEbsVolume(t *testing.T) {
	const expectedAnswer = cli.ResponseYes

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune(expectedAnswer),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskKeepEbsVolume(testQMHelper, true)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
}

func TestAskAutoTerminationTimerMinutes(t *testing.T) {
	const expectedAnswer = "30"

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune(expectedAnswer),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskAutoTerminationTimerMinutes(testEC2, testQMHelper, 0)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
}

func TestAskVpc_Success(t *testing.T) {
	const expectedVpc = "vpc-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{
			{
				VpcId:     aws.String(expectedVpc),
				CidrBlock: aws.String("some block"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("test vpc"),
					},
				},
				IsDefault: aws.Bool(true),
			},
			{
				VpcId:     aws.String("vpc-67890"),
				CidrBlock: aws.String("some block"),
				IsDefault: aws.Bool(false),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskVpc(testEC2, testQMHelper, "")
	th.Ok(t, err)
	th.Equals(t, expectedVpc, *answer)
}

func TestAskVpc_DescribeVpcsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskVpc(testEC2, testQMHelper, "")
	th.Nok(t, err)
}

func TestAskSubnet_Success(t *testing.T) {
	const testVpc = "vpc-12345"
	const expectedSubnet = "subnet-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{
			{
				SubnetId:         aws.String(expectedSubnet),
				VpcId:            aws.String(testVpc),
				CidrBlock:        aws.String("some block"),
				AvailabilityZone: aws.String("some az"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("test subnet"),
					},
				},
			},
			{
				SubnetId:         aws.String("subnet-67890"),
				VpcId:            aws.String(testVpc),
				CidrBlock:        aws.String("some block"),
				AvailabilityZone: aws.String("some az"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSubnet(testEC2, testQMHelper, testVpc, "")
	th.Ok(t, err)
	th.Equals(t, expectedSubnet, *answer)
}

func TestAskSubnet_DescribeSubnetsPagesError(t *testing.T) {
	const testVpc = "vpc-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskSubnet(testEC2, testQMHelper, testVpc, "")
	th.Nok(t, err)
}

func TestAskSubnetPlaceholder_Success(t *testing.T) {
	const expectedAz = "us-east-1"

	testEC2.Svc = &th.MockedEC2Svc{
		AvailabilityZones: []*ec2.AvailabilityZone{
			{
				ZoneName: aws.String(expectedAz),
				ZoneId:   aws.String("some id"),
			},
			{
				ZoneName: aws.String("us-east-2"),
				ZoneId:   aws.String("some id"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSubnetPlaceholder(testEC2, testQMHelper, "")
	th.Ok(t, err)
	th.Equals(t, expectedAz, *answer)
}

func TestAskSubnetPlaceholder_DescribeAvailabilityZonesError(t *testing.T) {
	const testAz = "us-east-1"

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeAvailabilityZonesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskSubnetPlaceholder(testEC2, testQMHelper, "")
	th.Nok(t, err)

}

func TestAskSecurityGroups_Success(t *testing.T) {
	var expectedGroups = []string{"sg-67890", "sg-12345"}
	defaultGroups := []*ec2.SecurityGroup{}

	testSecurityGroups := []*ec2.SecurityGroup{
		{
			GroupName: aws.String("Group1"),
			GroupId:   aws.String(expectedGroups[0]),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
		{
			GroupName: aws.String("Group2"),
			GroupId:   aws.String("sg-67890"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
		{
			GroupName: aws.String("Group3"),
			GroupId:   aws.String(expectedGroups[1]),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSecurityGroups(testQMHelper, testSecurityGroups, defaultGroups)
	th.Equals(t, expectedGroups, answer)

	th.Ok(t, err)
}

func TestAskSecurityGroups_NoGroup(t *testing.T) {
	testSecurityGroups := []*ec2.SecurityGroup{}
	defaultGroups := []*ec2.SecurityGroup{}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSecurityGroups(testQMHelper, testSecurityGroups, defaultGroups)
	th.Equals(t, cli.ResponseNew, answer[0])

	th.Ok(t, err)
}

func TestAskSecurityGroupPlaceholder(t *testing.T) {
	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSecurityGroupPlaceholder(testQMHelper)
	th.Equals(t, cli.ResponseAll, answer)

	th.Ok(t, err)
}

func TestAskConfirmationWithTemplate_Success_NoOverriding(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1
	const expectedAnswer = cli.ResponseYes

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{
			{
				LaunchTemplateId: aws.String(testTemplateId),
				VersionNumber:    aws.Int64(testVersion),
				LaunchTemplateData: &ec2.ResponseLaunchTemplateData{
					ImageId:      aws.String("ami-12345"),
					InstanceType: aws.String(ec2.InstanceTypeT2Micro),
				},
			},
		},
	}

	testSimpleConfig := &config.SimpleInfo{
		LaunchTemplateId:      testTemplateId,
		LaunchTemplateVersion: strconv.Itoa(testVersion),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskConfirmationWithTemplate(testEC2, testQMHelper, testSimpleConfig)
	th.Ok(t, err)
	th.Equals(t, expectedAnswer, *answer)
}

func TestAskConfirmationWithTemplate_Success_Overriding(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1
	const expectedAnswer = cli.ResponseYes

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{
			{
				LaunchTemplateId: aws.String(testTemplateId),
				VersionNumber:    aws.Int64(testVersion),
				LaunchTemplateData: &ec2.ResponseLaunchTemplateData{
					ImageId:      aws.String("ami-12345"),
					InstanceType: aws.String(ec2.InstanceTypeT2Micro),
				},
			},
		},
	}

	testSimpleConfig := &config.SimpleInfo{
		LaunchTemplateId:      testTemplateId,
		LaunchTemplateVersion: strconv.Itoa(testVersion),
		SubnetId:              "subnet-12345",
		InstanceType:          ec2.InstanceTypeT2Micro,
		ImageId:               "ami-12345",
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskConfirmationWithTemplate(testEC2, testQMHelper, testSimpleConfig)
	th.Ok(t, err)
	th.Equals(t, expectedAnswer, *answer)
}

func TestAskConfirmationWithTemplate_DescribeSubnetsPagesError(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{
			{
				LaunchTemplateId: aws.String(testTemplateId),
				VersionNumber:    aws.Int64(testVersion),
				LaunchTemplateData: &ec2.ResponseLaunchTemplateData{
					ImageId:      aws.String("ami-12345"),
					InstanceType: aws.String(ec2.InstanceTypeT2Micro),
					NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecification{
						{
							SubnetId: aws.String("subnet-12345"),
						},
					},
				},
			},
		},
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	testSimpleConfig := &config.SimpleInfo{
		LaunchTemplateId:      testTemplateId,
		LaunchTemplateVersion: strconv.Itoa(testVersion),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskConfirmationWithTemplate(testEC2, testQMHelper, testSimpleConfig)
	th.Nok(t, err)
}

func TestAskConfirmationWithTemplate_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	testSimpleConfig := config.NewSimpleInfo()

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskConfirmationWithTemplate(testEC2, testQMHelper, testSimpleConfig)
	th.Nok(t, err)
}

/*
AskConfirmationWithInput Tests
*/

var testSimpleConfig = &config.SimpleInfo{
	Region:                        "us-east-1",
	ImageId:                       "ami-12345",
	InstanceType:                  ec2.InstanceTypeT2Micro,
	SubnetId:                      "subnet-12345",
	AutoTerminationTimerMinutes:   30,
	KeepEbsVolumeAfterTermination: true,
	SecurityGroupIds:              []string{"sg-12345"},
}

var testDetailedConfig = &config.DetailedInfo{
	Image: &ec2.Image{
		ImageId: aws.String(testSimpleConfig.ImageId),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("device 1"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeType: aws.String("gp2"),
					VolumeSize: aws.Int64(10),
				},
			},
		},
		PlatformDetails: aws.String(ec2.CapacityReservationInstancePlatformLinuxUnix),
	},
	InstanceTypeInfo: &ec2.InstanceTypeInfo{
		InstanceType: aws.String(testSimpleConfig.InstanceType),
		InstanceStorageInfo: &ec2.InstanceStorageInfo{
			TotalSizeInGB: aws.Int64(40),
		},
	},
	Subnet: &ec2.Subnet{
		SubnetId: aws.String(testSimpleConfig.SubnetId),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("test subnet"),
			},
		},
	},
	Vpc: &ec2.Vpc{
		VpcId: aws.String("vpc-12345"),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("test vpc"),
			},
		},
	},
	SecurityGroups: []*ec2.SecurityGroup{
		{
			GroupId: aws.String(testSimpleConfig.SecurityGroupIds[0]),
		},
	},
}

func TestAskConfirmationWithInput_Success_NoNewInfrastructure(t *testing.T) {
	const expectedAnswer = cli.ResponseYes

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskConfirmationWithInput(testQMHelper, testSimpleConfig, testDetailedConfig, true)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
}

func TestAskConfirmationWithInput_Success_NewInfrastructure(t *testing.T) {
	const expectedAnswer = cli.ResponseYes
	// Modify the configs for creating new infrastructure
	testSimpleConfig.NewVPC = true
	testSimpleConfig.SecurityGroupIds = []string{cli.ResponseNew}
	testSimpleConfig.AutoTerminationTimerMinutes = 0
	testSimpleConfig.SubnetId = "us-east-2"
	testSimpleConfig.CapacityType = "Spot"
	testDetailedConfig.SecurityGroups = nil

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskConfirmationWithInput(testQMHelper, testSimpleConfig, testDetailedConfig, true)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
}

func TestAskSaveConfig(t *testing.T) {
	const expectedAnswer = cli.ResponseYes

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSaveConfig(testQMHelper)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
}

func TestAskInstanceId_Success(t *testing.T) {
	const expectedInstance = "i-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{
			{
				InstanceId: aws.String(expectedInstance),
			},
			{
				InstanceId: aws.String("i-67890"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskInstanceId(testEC2, testQMHelper)
	th.Ok(t, err)
	th.Equals(t, expectedInstance, *answer)
}

func TestAskInstanceId_NoInstance(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceId(testEC2, testQMHelper)
	th.Nok(t, err)
}

func TestAskInstanceId_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceId(testEC2, testQMHelper)
	th.Nok(t, err)
}

func TestAskInstanceIds_Success(t *testing.T) {
	expectedInstances := []string{"i-12345"}

	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{
			{
				InstanceId: aws.String(expectedInstances[0]),
			},
			{
				InstanceId: aws.String("i-67890"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	addedInstances := []string{"i-67890"}

	answer, err := question.AskInstanceIds(testEC2, testQMHelper, addedInstances)
	th.Ok(t, err)
	th.Equals(t, expectedInstances, answer)
}

func TestAskInstanceIds_NoInstance(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}
	addedInstances := []string{}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceIds(testEC2, testQMHelper, addedInstances)
	th.Nok(t, err)
}

func TestAskInstanceIds_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}
	addedInstances := []string{}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceIds(testEC2, testQMHelper, addedInstances)
	th.Nok(t, err)
}

/*
Instance Selector Question Tests
*/

const testInstanceType = ec2.InstanceTypeT2Micro

var testInstanceTypeInfos = []*instancetypes.Details{
	{
		InstanceTypeInfo: ec2.InstanceTypeInfo{
			InstanceType: aws.String(testInstanceType),
			VCpuInfo: &ec2.VCpuInfo{
				DefaultVCpus: aws.Int64(2),
			},
			MemoryInfo: &ec2.MemoryInfo{
				SizeInMiB: aws.Int64(4096),
			},
			InstanceStorageSupported: aws.Bool(false),
		},
	},
	{
		InstanceTypeInfo: ec2.InstanceTypeInfo{
			InstanceType: aws.String("t2.nano"),
			VCpuInfo: &ec2.VCpuInfo{
				DefaultVCpus: aws.Int64(1),
			},
			MemoryInfo: &ec2.MemoryInfo{
				SizeInMiB: aws.Int64(2048),
			},
			InstanceStorageSupported: aws.Bool(false),
		},
	},
}
var testSelector = &th.MockedSelector{
	InstanceTypes: testInstanceTypeInfos,
}

func TestAskInstanceTypeInstanceSelector_Success(t *testing.T) {
	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskInstanceTypeInstanceSelector(testEC2, testQMHelper, testSelector, "2", "4")
	th.Ok(t, err)
	th.Equals(t, testInstanceType, *answer)
}

func TestAskInstanceTypeInstanceSelector_BadVcpus(t *testing.T) {
	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testQMHelper, testSelector, "a", "4")
	th.Nok(t, err)
}

func TestAskInstanceTypeInstanceSelector_BadMemory(t *testing.T) {
	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testQMHelper, testSelector, "2", "a")
	th.Nok(t, err)
}

func TestAskInstanceTypeInstanceSelector_NoResult(t *testing.T) {
	testSelector = &th.MockedSelector{
		InstanceTypes: []*instancetypes.Details{},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testQMHelper, testSelector, "2", "4")
	th.Nok(t, err)
}

func TestAskInstanceTypeInstanceSelector_SelectorError(t *testing.T) {
	testSelector = &th.MockedSelector{
		InstanceTypes: testInstanceTypeInfos,
		SelectorError: errors.New("Test error"),
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testQMHelper, testSelector, "2", "4")
	th.Nok(t, err)
}

func TestAskIamProfile_Success(t *testing.T) {
	expectedProfileName := "profile2"
	testProfiles := []*iam.InstanceProfile{
		{
			InstanceProfileName: aws.String("profile1"),
			InstanceProfileId:   aws.String("id1"),
			CreateDate:          aws.Time(time.Now()),
		},
		{
			InstanceProfileName: aws.String("profile2"),
			InstanceProfileId:   aws.String("id2"),
			CreateDate:          aws.Time(time.Now()),
		},
		{
			InstanceProfileName: aws.String("profile3"),
			InstanceProfileId:   aws.String("id3"),
			CreateDate:          aws.Time(time.Now()),
		},
	}
	mockedIam := &th.MockedIAMSvc{
		InstanceProfiles: testProfiles,
	}
	iam := &iamhelper.IAMHelper{Client: mockedIam}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskIamProfile(testQMHelper, iam, "")
	th.Ok(t, err)
	th.Equals(t, expectedProfileName, answer)
}

func TestAskIamProfile_Error(t *testing.T) {
	mockedIam := &th.MockedIAMSvc{
		ListInstanceProfilesError: errors.New("Test error"),
	}
	iam := &iamhelper.IAMHelper{Client: mockedIam}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	_, err := question.AskIamProfile(testQMHelper, iam, "")
	th.Nok(t, err)
}

func TestAskCapacityType(t *testing.T) {
	testRegion := "us-east-1"
	expectedCapacity := question.DefaultCapacityTypeText.Spot

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskCapacityType(testQMHelper, testInstanceType, testRegion, "")
	th.Equals(t, expectedCapacity, answer)

	th.Ok(t, err)
}

func TestAskBootScriptConfirmation(t *testing.T) {
	expectedConfirmation := cli.ResponseYes
	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyUp,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskBootScriptConfirmation(testEC2, testQMHelper, "")
	th.Equals(t, expectedConfirmation, answer)

	th.Ok(t, err)
}

func TestAskBootScript(t *testing.T) {
	expectedBootScript, err := ioutil.TempFile("", "mocked_filepath")
	defer os.Remove(expectedBootScript.Name())
	if err != nil {
		t.Errorf("There was an error creating tempfile: %v", err)
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune(expectedBootScript.Name()),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskBootScript(testEC2, testQMHelper, "")
	th.Equals(t, expectedBootScript.Name(), answer)

	th.Ok(t, err)
}

func TestAskUserTagsConfirmation(t *testing.T) {
	expectedConfirmation := cli.ResponseNo

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	userTags := make(map[string]string)

	answer, err := question.AskUserTagsConfirmation(testEC2, testQMHelper, userTags)
	th.Equals(t, expectedConfirmation, answer)

	th.Ok(t, err)
}

func TestAskUserTags(t *testing.T) {
	expectedTags := "Key1|Value1, Key2|Value2"
	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Runes: []rune("Key1"),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Runes: []rune("Value1"),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Runes: []rune("Key2"),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Runes: []rune("Value2"),
				Type:  tea.KeyRunes,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyRight,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	userTags := make(map[string]string)

	answer, err := question.AskUserTags(testEC2, testQMHelper, userTags)
	th.Equals(t, expectedTags, answer)

	th.Ok(t, err)
}

func initQuestionTest(t *testing.T, input string) {
	err := th.TakeOverStdin(input)
	th.Ok(t, err)

	err = th.TakeOverStdout()
	th.Ok(t, err)
}

func cleanupQuestionTest() string {
	th.RestoreStdin()
	return th.ReadStdout()
}

/*
Question Default Testing
*/

func TestAskRegion_WithDefault(t *testing.T) {
	const defaultRegion = "us-west-1"

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	testEC2.Svc = &th.MockedEC2Svc{
		Regions: []*ec2.Region{
			{
				RegionName: aws.String("us-east-1"),
			},
			{
				RegionName: aws.String("us-east-2"),
			},
			{
				RegionName: aws.String(defaultRegion),
			},
			{
				RegionName: aws.String("us-west-2"),
			},
		},
	}

	answer, err := question.AskRegion(testEC2, testQMHelper, defaultRegion)
	th.Ok(t, err)

	th.Equals(t, defaultRegion, *answer)
}

func TestAskLaunchTemplate_WithDefault(t *testing.T) {
	const defaultTemplateId = "lt-67890"

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{
			{
				LaunchTemplateId:    aws.String("lt-12345"),
				LaunchTemplateName:  aws.String("lt-12345"),
				LatestVersionNumber: aws.Int64(1),
			},
			{
				LaunchTemplateId:    aws.String(defaultTemplateId),
				LaunchTemplateName:  aws.String(defaultTemplateId),
				LatestVersionNumber: aws.Int64(1),
			},
		},
	}

	answer, err := question.AskLaunchTemplate(testEC2, testQMHelper, defaultTemplateId)
	th.Equals(t, defaultTemplateId, *answer)

	th.Ok(t, err)
}

func TestAskLaunchTemplateVersion_WithDefault(t *testing.T) {
	const testTemplateId = "lt-12345"
	const defaultVersion = 2

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{
			{
				LaunchTemplateId:   aws.String(testTemplateId),
				VersionDescription: aws.String("description"),
				VersionNumber:      aws.Int64(1),
				DefaultVersion:     aws.Bool(true),
			},
			{
				LaunchTemplateId: aws.String(testTemplateId),
				VersionNumber:    aws.Int64(defaultVersion),
				DefaultVersion:   aws.Bool(false),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskLaunchTemplateVersion(testEC2, testQMHelper, testTemplateId, strconv.Itoa(defaultVersion))
	th.Ok(t, err)
	th.Equals(t, strconv.Itoa(defaultVersion), *answer)
}

func TestAskIfEnterInstanceType_WithDefault(t *testing.T) {
	const defaultInstanceType = "t3.medium"

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(ec2.InstanceTypeT2Micro),
				FreeTierEligible: aws.Bool(true),
			},
			{
				InstanceType:     aws.String(defaultInstanceType),
				FreeTierEligible: aws.Bool(false),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskIfEnterInstanceType(testEC2, testQMHelper, defaultInstanceType)
	th.Ok(t, err)
	th.Equals(t, defaultInstanceType, *answer)
}

func TestAskInstanceType_WithDefault(t *testing.T) {
	const defaultInstanceType = "t1.micro"

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(ec2.InstanceTypeT2Micro),
				FreeTierEligible: aws.Bool(true),
			},
			{
				InstanceType:     aws.String(defaultInstanceType),
				FreeTierEligible: aws.Bool(false),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskInstanceType(testEC2, testQMHelper, defaultInstanceType)
	th.Ok(t, err)
	th.Equals(t, defaultInstanceType, *answer)
}

func TestAskImage_WithDefault(t *testing.T) {
	const defaultImage = "ami-12345"
	const testInstanceType = ec2.InstanceTypeT2Micro

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:             aws.String(testInstanceType),
				InstanceStorageSupported: aws.Bool(true),
				ProcessorInfo:            &ec2.ProcessorInfo{SupportedArchitectures: defaultArchitecture},
			},
		},
		Images: []*ec2.Image{
			{
				ImageId:      aws.String("ami-92307"),
				CreationDate: aws.String("some time"),
			},
			{
				ImageId:      aws.String(defaultImage),
				CreationDate: aws.String("some time"),
			},
			{
				ImageId:      aws.String("ami-13458"),
				CreationDate: aws.String("some other time"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskImage(testEC2, testQMHelper, testInstanceType, defaultImage)
	th.Ok(t, err)
	th.Equals(t, defaultImage, *answer.ImageId)
}

func TestAskIamProfile_WithDefault(t *testing.T) {
	defaultProfileName := "profile2"
	testProfiles := []*iam.InstanceProfile{
		{
			InstanceProfileName: aws.String("profile1"),
			InstanceProfileId:   aws.String("id1"),
			CreateDate:          aws.Time(time.Now()),
		},
		{
			InstanceProfileName: aws.String(defaultProfileName),
			InstanceProfileId:   aws.String("id2"),
			CreateDate:          aws.Time(time.Now()),
		},
		{
			InstanceProfileName: aws.String("profile3"),
			InstanceProfileId:   aws.String("id3"),
			CreateDate:          aws.Time(time.Now()),
		},
	}
	mockedIam := &th.MockedIAMSvc{
		InstanceProfiles: testProfiles,
	}
	iam := &iamhelper.IAMHelper{Client: mockedIam}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskIamProfile(testQMHelper, iam, defaultProfileName)
	th.Ok(t, err)
	th.Equals(t, defaultProfileName, answer)
}

func TestAskVpc_WithDefault(t *testing.T) {
	const defaultVpc = "vpc-91378"

	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{
			{
				VpcId:     aws.String("vpc-12345"),
				CidrBlock: aws.String("some block"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("test vpc"),
					},
				},
				IsDefault: aws.Bool(true),
			},
			{
				VpcId:     aws.String("vpc-67890"),
				CidrBlock: aws.String("some block"),
				IsDefault: aws.Bool(false),
			},
			{
				VpcId:     aws.String(defaultVpc),
				CidrBlock: aws.String("some block"),
				IsDefault: aws.Bool(false),
			},
			{
				VpcId:     aws.String("vpc-41239"),
				CidrBlock: aws.String("some block"),
				IsDefault: aws.Bool(false),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskVpc(testEC2, testQMHelper, defaultVpc)
	th.Ok(t, err)
	th.Equals(t, defaultVpc, *answer)
}

func TestAskSubnet_WithDefault(t *testing.T) {
	const testVpc = "vpc-12345"
	const defaultSubnet = "subnet-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{
			{
				SubnetId:         aws.String("subnet-01894"),
				VpcId:            aws.String(testVpc),
				CidrBlock:        aws.String("some block"),
				AvailabilityZone: aws.String("some az"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("test subnet"),
					},
				},
			},
			{
				SubnetId:         aws.String("subnet-67890"),
				VpcId:            aws.String(testVpc),
				CidrBlock:        aws.String("some block"),
				AvailabilityZone: aws.String("some az"),
			},
			{
				SubnetId:         aws.String("subnet-77245"),
				VpcId:            aws.String(testVpc),
				CidrBlock:        aws.String("some block"),
				AvailabilityZone: aws.String("some other az"),
			},
			{
				SubnetId:         aws.String(defaultSubnet),
				VpcId:            aws.String(testVpc),
				CidrBlock:        aws.String("some block"),
				AvailabilityZone: aws.String("some az"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSubnet(testEC2, testQMHelper, testVpc, defaultSubnet)
	th.Ok(t, err)
	th.Equals(t, defaultSubnet, *answer)
}

func TestAskSubnetPlaceholder_WithDefault(t *testing.T) {
	const defaultAz = "us-east-2"

	testEC2.Svc = &th.MockedEC2Svc{
		AvailabilityZones: []*ec2.AvailabilityZone{
			{
				ZoneName: aws.String("us-east-1"),
				ZoneId:   aws.String("some id"),
			},
			{
				ZoneName: aws.String(defaultAz),
				ZoneId:   aws.String("some id"),
			},
			{
				ZoneName: aws.String("us-west-1"),
				ZoneId:   aws.String("some id"),
			},
			{
				ZoneName: aws.String("us-west-2"),
				ZoneId:   aws.String("some id"),
			},
		},
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskSubnetPlaceholder(testEC2, testQMHelper, defaultAz)
	th.Ok(t, err)
	th.Equals(t, defaultAz, *answer)
}

func TestAskBootScriptConfirmation_WithDefault(t *testing.T) {
	defaultBootScript := "BootScript/FilePath"
	defaultConfirmation := cli.ResponseYes

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	confirmation, err := question.AskBootScriptConfirmation(testEC2, testQMHelper, defaultBootScript)
	th.Equals(t, defaultConfirmation, confirmation)

	th.Ok(t, err)
}

func TestAskBootScript_WithDefault(t *testing.T) {
	defaultBootScript, err := ioutil.TempFile("", "mocked_filepath")
	defer os.Remove(defaultBootScript.Name())
	if err != nil {
		t.Errorf("There was an error creating tempfile: %v", err)
	}

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskBootScript(testEC2, testQMHelper, defaultBootScript.Name())
	th.Equals(t, defaultBootScript.Name(), answer)

	th.Ok(t, err)
}

func TestAskUserTagsConfirmation_WithDefault(t *testing.T) {
	defaultConfirmation := cli.ResponseYes

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	userTags := make(map[string]string)
	userTags["key"] = "value"

	confirmation, err := question.AskUserTagsConfirmation(testEC2, testQMHelper, userTags)
	th.Equals(t, defaultConfirmation, confirmation)

	th.Ok(t, err)
}

func TestAskUserTags_WithDefault(t *testing.T) {
	expectedTagString := "Key1|Value1, Key2|Value2, Key3|Value3, Key4|Value4"
	expectedTags := strings.Split(expectedTagString, ",")

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyDown,
			},
			tea.KeyMsg{
				Type: tea.KeyRight,
			},
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	userTags := make(map[string]string)
	userTags["Key1"] = "Value1"
	userTags["Key2"] = "Value2"
	userTags["Key3"] = "Value3"
	userTags["Key4"] = "Value4"

	answer, err := question.AskUserTags(testEC2, testQMHelper, userTags)
	log.Println(answer)
	actualTags := strings.Split(answer, ",")

	th.Assert(t, len(actualTags) == 4, "ActualTags length should be 4")
	for _, expectedTag := range expectedTags {
		thisTagMatches := false
		for _, actualTag := range actualTags {
			if expectedTag == actualTag {
				th.Equals(t, strings.TrimSpace(expectedTag), strings.TrimSpace(actualTag))
				thisTagMatches = true
				break
			}
		}
		th.Assert(t, thisTagMatches, fmt.Sprintf("Unable to find matching actual tag for expected tag %s", expectedTag))
	}

	th.Ok(t, err)
}

func TestAskCapacityType_WithDefault(t *testing.T) {
	testRegion := "us-east-1"
	defaultCapacity := question.DefaultCapacityTypeText.Spot

	testQMHelper.Svc = &th.MockedQMHelperSvc{
		UserInputs: []tea.Msg{
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
		},
	}

	answer, err := question.AskCapacityType(testQMHelper, testInstanceType, testRegion, defaultCapacity)
	th.Equals(t, defaultCapacity, answer)

	th.Ok(t, err)
}
