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
	th "simple-ec2/test/testhelper"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/instancetypes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

var testEC2 = &ec2helper.EC2Helper{
	Sess: &session.Session{
		Config: &aws.Config{
			Region: aws.String("us-east-1"),
		},
	},
}
var defaultArchitecture = aws.StringSlice([]string{"x86_64"})

/*
AskQuestion Tests
*/

const expectedOutput = `
These are the optionsThis is a question [default option]:  `
const invalidInputQuestionPrompt = `
These are the optionsThis is a question [default option]:  Input invalid. Please try again.
This is a question [default option]:  `

var input = &question.AskQuestionInput{
	QuestionString:    "This is a question",
	OptionsString:     aws.String("These are the options"),
	DefaultOptionRepr: aws.String("default option"),
	DefaultOption:     aws.String(cli.ResponseYes),
	IndexedOptions:    []string{"Option 1", "Option 2"},
	StringOptions:     []string{cli.ResponseYes, cli.ResponseNo},
	AcceptAnyInteger:  true,
	AcceptAnyString:   true,
}

func TestAskQuestion_StringOptionAnswer(t *testing.T) {
	const testResponse = cli.ResponseNo
	initQuestionTest(t, testResponse+"\n")

	answer := question.AskQuestion(input)
	output := cleanupQuestionTest()
	th.Equals(t, expectedOutput, output)
	th.Equals(t, testResponse, answer)
}

func TestAskQuestion_InvalidInput(t *testing.T) {
	const expectedInvalidInput = "heap"
	initQuestionTest(t, expectedInvalidInput+"\n")
	input.AcceptAnyString = false

	question.AskQuestion(input)

	output := cleanupQuestionTest()
	th.Equals(t, invalidInputQuestionPrompt, output)
}

func TestAskQuestion_IndexedOptionAnswer(t *testing.T) {
	input.DefaultOptionRepr = nil
	const index = "1"
	initQuestionTest(t, index+"\n")

	answer := question.AskQuestion(input)
	th.Equals(t, input.IndexedOptions[0], answer)

	cleanupQuestionTest()
}

func TestAskQuestion_DefaultAnswer(t *testing.T) {
	initQuestionTest(t, "\n")

	answer := question.AskQuestion(input)
	th.Equals(t, *input.DefaultOption, answer)

	cleanupQuestionTest()
}

func TestAskQuestion_IntegerAnswer(t *testing.T) {
	const expectedInteger = "5"
	initQuestionTest(t, expectedInteger+"\n")

	answer := question.AskQuestion(input)
	th.Equals(t, expectedInteger, answer)

	cleanupQuestionTest()
}

func TestAskQuestion_AnyStringAnswer(t *testing.T) {
	// This test needs its own copy of the AskQuestionInput object to prevent some kind of race condition with the other
	// tests when running the whole test suite. We don't exactly know why, but it works.
	var anyStringQuestion = &question.AskQuestionInput{
		QuestionString:  "This is a question?",
		AcceptAnyString: true,
	}
	const expectedString = "any string"
	initQuestionTest(t, expectedString+"\n")

	answer := question.AskQuestion(anyStringQuestion)
	th.Equals(t, expectedString, answer)

	cleanupQuestionTest()
}

func TestAskQuestion_FunctionCheckedInput(t *testing.T) {
	const expectedImageId = "ami-12345"
	testEC2 := &ec2helper.EC2Helper{
		Svc: &th.MockedEC2Svc{
			Images: []*ec2.Image{
				{
					ImageId: aws.String(expectedImageId),
				},
			},
		},
	}
	input.EC2Helper = testEC2
	input.Fns = []question.CheckInput{
		ec2helper.ValidateImageId,
	}

	initQuestionTest(t, expectedImageId+"\n")

	answer := question.AskQuestion(input)
	th.Equals(t, expectedImageId, answer)

	cleanupQuestionTest()
}

/*
Other Question Asking Tests
*/

func TestAskRegion_Success(t *testing.T) {
	const expectedRegion = "us-east-2"
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskRegion(testEC2, "")
	th.Ok(t, err)
	th.Equals(t, expectedRegion, *answer)

	cleanupQuestionTest()
}

func TestAskRegion_DescribeRegionsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeRegionsError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskRegion(testEC2, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskLaunchTemplate_Success(t *testing.T) {
	const expectedTemplateId = "lt-12345"
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskLaunchTemplate(testEC2, "")
	th.Equals(t, expectedTemplateId, *answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskLaunchTemplate_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	answer, err := question.AskLaunchTemplate(testEC2, "")
	th.Equals(t, cli.ResponseNo, *answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskLaunchTemplateVersion_Success(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskLaunchTemplateVersion(testEC2, testTemplateId, "")
	th.Ok(t, err)
	th.Equals(t, strconv.Itoa(testVersion), *answer)

	cleanupQuestionTest()
}

func TestAskLaunchTemplateVersion_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	const testTemplateId = "lt-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskLaunchTemplateVersion(testEC2, testTemplateId, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskIfEnterInstanceType_Success(t *testing.T) {
	const expectedInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "\n")

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(expectedInstanceType),
				FreeTierEligible: aws.Bool(true),
			},
		},
	}

	answer, err := question.AskIfEnterInstanceType(testEC2, "")
	th.Ok(t, err)
	th.Equals(t, expectedInstanceType, *answer)

	cleanupQuestionTest()
}

func TestAskIfEnterInstanceType_(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskIfEnterInstanceType(testEC2, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceType_Success(t *testing.T) {
	const expectedInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, expectedInstanceType+"\n")

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(expectedInstanceType),
				FreeTierEligible: aws.Bool(true),
			},
		},
	}

	answer, err := question.AskInstanceType(testEC2, "")
	th.Ok(t, err)
	th.Equals(t, expectedInstanceType, *answer)

	cleanupQuestionTest()
}

func TestAskInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskInstanceType(testEC2, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceTypeVCpu(t *testing.T) {
	const expectedVcpus = "2"
	initQuestionTest(t, expectedVcpus+"\n")

	answer, err := question.AskInstanceTypeVCpu(testEC2)
	th.Equals(t, expectedVcpus, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskInstanceTypeMemory(t *testing.T) {
	const expectedMemory = "2"
	initQuestionTest(t, expectedMemory+"\n")

	answer, err := question.AskInstanceTypeMemory(testEC2)
	th.Equals(t, expectedMemory, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskImage_Success(t *testing.T) {
	const expectedImage = "ami-12345"
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskImage(testEC2, testInstanceType, "")
	th.Ok(t, err)
	th.Equals(t, expectedImage, *answer.ImageId)

	cleanupQuestionTest()
}

func TestAskImage_NoImage(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:             aws.String(testInstanceType),
				InstanceStorageSupported: aws.Bool(true),
				ProcessorInfo:            &ec2.ProcessorInfo{SupportedArchitectures: defaultArchitecture},
			},
		},
	}

	_, err := question.AskImage(testEC2, testInstanceType, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskImage_DescribeInstanceTypesPagesError(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := question.AskImage(testEC2, testInstanceType, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskImage_DescribeImagesError(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

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

	_, err := question.AskImage(testEC2, testInstanceType, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskKeepEbsVolume(t *testing.T) {
	const expectedAnswer = cli.ResponseYes
	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskKeepEbsVolume(true)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskAutoTerminationTimerMinutes(t *testing.T) {
	const expectedAnswer = "30"
	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskAutoTerminationTimerMinutes(testEC2, 0)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskVpc_Success(t *testing.T) {
	const expectedVpc = "vpc-12345"
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskVpc(testEC2, "")
	th.Ok(t, err)
	th.Equals(t, expectedVpc, *answer)

	cleanupQuestionTest()
}

func TestAskVpc_DescribeVpcsPagesError(t *testing.T) {
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	_, err := question.AskVpc(testEC2, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskSubnet_Success(t *testing.T) {
	const testVpc = "vpc-12345"
	const expectedSubnet = "subnet-12345"
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskSubnet(testEC2, testVpc, "")
	th.Ok(t, err)
	th.Equals(t, expectedSubnet, *answer)

	cleanupQuestionTest()
}

func TestAskSubnet_DescribeSubnetsPagesError(t *testing.T) {
	const testVpc = "vpc-12345"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := question.AskSubnet(testEC2, testVpc, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskSubnetPlaceholder_Success(t *testing.T) {
	const expectedAz = "us-east-1"
	initQuestionTest(t, "1\n")

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

	answer, err := question.AskSubnetPlaceholder(testEC2, "")
	th.Ok(t, err)
	th.Equals(t, expectedAz, *answer)

	cleanupQuestionTest()
}

func TestAskSubnetPlaceholder_DescribeAvailabilityZonesError(t *testing.T) {
	const testAz = "us-east-1"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeAvailabilityZonesError: errors.New("Test error"),
	}

	_, err := question.AskSubnetPlaceholder(testEC2, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskSecurityGroups_Success(t *testing.T) {
	const expectedGroup = "sg-12345"
	defaultGroups := []*ec2.SecurityGroup{}

	testSecurityGroups := []*ec2.SecurityGroup{
		{
			GroupName: aws.String("Group1"),
			GroupId:   aws.String(expectedGroup),
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
			GroupId:   aws.String("sg-67890"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
	}

	initQuestionTest(t, "1\n")

	answer, err := question.AskSecurityGroups(testSecurityGroups, defaultGroups)
	th.Equals(t, expectedGroup, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskSecurityGroups_NoGroup(t *testing.T) {
	testSecurityGroups := []*ec2.SecurityGroup{}
	defaultGroups := []*ec2.SecurityGroup{}

	initQuestionTest(t, "1\n")

	answer, err := question.AskSecurityGroups(testSecurityGroups, defaultGroups)
	th.Equals(t, cli.ResponseNo, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskSecurityGroupPlaceholder(t *testing.T) {
	initQuestionTest(t, "1\n")

	answer, err := question.AskSecurityGroupPlaceholder()
	th.Equals(t, cli.ResponseAll, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
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

	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskConfirmationWithTemplate(testEC2, testSimpleConfig)
	th.Ok(t, err)
	th.Equals(t, expectedAnswer, *answer)

	cleanupQuestionTest()
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

	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskConfirmationWithTemplate(testEC2, testSimpleConfig)
	th.Ok(t, err)
	th.Equals(t, expectedAnswer, *answer)

	cleanupQuestionTest()
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

	initQuestionTest(t, cli.ResponseYes+"\n")

	_, err := question.AskConfirmationWithTemplate(testEC2, testSimpleConfig)
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskConfirmationWithTemplate_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	testSimpleConfig := config.NewSimpleInfo()

	initQuestionTest(t, cli.ResponseYes+"\n")

	_, err := question.AskConfirmationWithTemplate(testEC2, testSimpleConfig)
	th.Nok(t, err)

	cleanupQuestionTest()
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
	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskConfirmationWithInput(testSimpleConfig, testDetailedConfig, true)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
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

	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskConfirmationWithInput(testSimpleConfig, testDetailedConfig, true)
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskSaveConfig(t *testing.T) {
	const expectedAnswer = cli.ResponseYes
	initQuestionTest(t, expectedAnswer+"\n")

	answer, err := question.AskSaveConfig()
	th.Equals(t, expectedAnswer, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
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

	initQuestionTest(t, "1\n")

	answer, err := question.AskInstanceId(testEC2)
	th.Ok(t, err)
	th.Equals(t, expectedInstance, *answer)

	cleanupQuestionTest()
}

func TestAskInstanceId_NoInstance(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{},
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceId(testEC2)
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceId_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceId(testEC2)
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceIds_Success(t *testing.T) {
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
	addedInstances := []string{"i-67890"}

	initQuestionTest(t, "1\n")

	answer, err := question.AskInstanceIds(testEC2, addedInstances)
	th.Ok(t, err)
	th.Equals(t, expectedInstance, *answer)

	cleanupQuestionTest()
}

func TestAskInstanceIds_NoInstance(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}
	addedInstances := []string{}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceIds(testEC2, addedInstances)
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceIds_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}
	addedInstances := []string{}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceIds(testEC2, addedInstances)
	th.Nok(t, err)

	cleanupQuestionTest()
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
	initQuestionTest(t, "1\n")

	answer, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "4")
	th.Ok(t, err)
	th.Equals(t, testInstanceType, *answer)

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_BadVcpus(t *testing.T) {
	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "a", "4")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_BadMemory(t *testing.T) {
	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "a")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_NoResult(t *testing.T) {
	testSelector = &th.MockedSelector{
		InstanceTypes: []*instancetypes.Details{},
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "4")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_SelectorError(t *testing.T) {
	testSelector = &th.MockedSelector{
		InstanceTypes: testInstanceTypeInfos,
		SelectorError: errors.New("Test error"),
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "4")
	th.Nok(t, err)

	cleanupQuestionTest()
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
	initQuestionTest(t, "2\n")

	answer, err := question.AskIamProfile(iam, "")
	th.Ok(t, err)
	th.Equals(t, expectedProfileName, answer)

	cleanupQuestionTest()
}

func TestAskIamProfile_Error(t *testing.T) {
	mockedIam := &th.MockedIAMSvc{
		ListInstanceProfilesError: errors.New("Test error"),
	}
	iam := &iamhelper.IAMHelper{Client: mockedIam}
	initQuestionTest(t, "1\n")

	_, err := question.AskIamProfile(iam, "")
	th.Nok(t, err)

	cleanupQuestionTest()
}

func TestAskCapacityType(t *testing.T) {
	testRegion := "us-east-1"
	expectedCapacity := question.DefaultCapacityTypeText.Spot
	initQuestionTest(t, "2\n")

	answer, err := question.AskCapacityType(testInstanceType, testRegion, "")
	th.Equals(t, expectedCapacity, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskBootScriptConfirmation(t *testing.T) {
	expectedConfirmation := cli.ResponseYes
	initQuestionTest(t, cli.ResponseYes+"\n")

	answer, err := question.AskBootScriptConfirmation(testEC2, "")
	th.Equals(t, expectedConfirmation, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskBootScript(t *testing.T) {
	expectedBootScript, err := ioutil.TempFile("", "mocked_filepath")
	defer os.Remove(expectedBootScript.Name())
	if err != nil {
		t.Errorf("There was an error creating tempfile: %v", err)
	}
	initQuestionTest(t, expectedBootScript.Name()+"\n")

	answer, err := question.AskBootScript(testEC2, "")
	th.Equals(t, expectedBootScript.Name(), answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskUserTagsConfirmation(t *testing.T) {
	expectedConfirmation := cli.ResponseNo
	initQuestionTest(t, cli.ResponseNo+"\n")

	userTags := make(map[string]string)

	answer, err := question.AskUserTagsConfirmation(testEC2, userTags)
	th.Equals(t, expectedConfirmation, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskUserTags(t *testing.T) {
	expectedTags := "Key1|Value1,Key2|Value2,Key3|Value3,Key4|Value4"
	initQuestionTest(t, expectedTags+"\n")

	userTags := make(map[string]string)

	answer, err := question.AskUserTags(testEC2, userTags)
	th.Equals(t, expectedTags, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
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
	initQuestionTest(t, "\n")

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

	answer, err := question.AskRegion(testEC2, defaultRegion)
	th.Ok(t, err)

	th.Equals(t, defaultRegion, *answer)

	cleanupQuestionTest()
}

func TestAskLaunchTemplate_WithDefault(t *testing.T) {
	const defaultTemplateId = "lt-67890"
	initQuestionTest(t, "\n")

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

	answer, err := question.AskLaunchTemplate(testEC2, defaultTemplateId)
	th.Equals(t, defaultTemplateId, *answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskLaunchTemplateVersion_WithDefault(t *testing.T) {
	const testTemplateId = "lt-12345"
	const defaultVersion = 2
	initQuestionTest(t, "\n")

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

	answer, err := question.AskLaunchTemplateVersion(testEC2, testTemplateId, strconv.Itoa(defaultVersion))
	th.Ok(t, err)
	th.Equals(t, strconv.Itoa(defaultVersion), *answer)

	cleanupQuestionTest()
}

func TestAskIfEnterInstanceType_WithDefault(t *testing.T) {
	const defaultInstanceType = "t3.medium"
	initQuestionTest(t, "\n")

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

	answer, err := question.AskIfEnterInstanceType(testEC2, defaultInstanceType)
	th.Ok(t, err)
	th.Equals(t, defaultInstanceType, *answer)

	cleanupQuestionTest()
}

func TestAskInstanceType_WithDefault(t *testing.T) {
	const defaultInstanceType = "t1.micro"
	initQuestionTest(t, "\n")

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

	answer, err := question.AskInstanceType(testEC2, defaultInstanceType)
	th.Ok(t, err)
	th.Equals(t, defaultInstanceType, *answer)

	cleanupQuestionTest()
}

func TestAskImage_WithDefault(t *testing.T) {
	const defaultImage = "ami-12345"
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "\n")

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

	answer, err := question.AskImage(testEC2, testInstanceType, defaultImage)
	th.Ok(t, err)
	th.Equals(t, defaultImage, *answer.ImageId)

	cleanupQuestionTest()
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
	initQuestionTest(t, "\n")

	answer, err := question.AskIamProfile(iam, defaultProfileName)
	th.Ok(t, err)
	th.Equals(t, defaultProfileName, answer)

	cleanupQuestionTest()
}

func TestAskVpc_WithDefault(t *testing.T) {
	const defaultVpc = "vpc-91378"
	initQuestionTest(t, "\n")

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

	answer, err := question.AskVpc(testEC2, defaultVpc)
	th.Ok(t, err)
	th.Equals(t, defaultVpc, *answer)

	cleanupQuestionTest()
}

func TestAskSubnet_WithDefault(t *testing.T) {
	const testVpc = "vpc-12345"
	const defaultSubnet = "subnet-12345"
	initQuestionTest(t, "\n")

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

	answer, err := question.AskSubnet(testEC2, testVpc, defaultSubnet)
	th.Ok(t, err)
	th.Equals(t, defaultSubnet, *answer)

	cleanupQuestionTest()
}

func TestAskSubnetPlaceholder_WithDefault(t *testing.T) {
	const defaultAz = "us-east-2"
	initQuestionTest(t, "\n")

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

	answer, err := question.AskSubnetPlaceholder(testEC2, defaultAz)
	th.Ok(t, err)
	th.Equals(t, defaultAz, *answer)

	cleanupQuestionTest()
}

func TestAskBootScriptConfirmation_WithDefault(t *testing.T) {
	defaultBootScript := "BootScript/FilePath"
	defaultConfirmation := cli.ResponseYes
	initQuestionTest(t, "\n")

	confirmation, err := question.AskBootScriptConfirmation(testEC2, defaultBootScript)
	th.Equals(t, defaultConfirmation, confirmation)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskBootScript_WithDefault(t *testing.T) {
	defaultBootScript, err := ioutil.TempFile("", "mocked_filepath")
	defer os.Remove(defaultBootScript.Name())
	if err != nil {
		t.Errorf("There was an error creating tempfile: %v", err)
	}
	initQuestionTest(t, "\n")

	answer, err := question.AskBootScript(testEC2, defaultBootScript.Name())
	th.Equals(t, defaultBootScript.Name(), answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskUserTagsConfirmation_WithDefault(t *testing.T) {
	defaultConfirmation := cli.ResponseYes
	initQuestionTest(t, "\n")

	userTags := make(map[string]string)
	userTags["key"] = "value"

	confirmation, err := question.AskUserTagsConfirmation(testEC2, userTags)
	th.Equals(t, defaultConfirmation, confirmation)

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskUserTags_WithDefault(t *testing.T) {
	expectedTagString := "Key1|Value1,Key2|Value2,Key3|Value3,Key4|Value4"
	expectedTags := strings.Split(expectedTagString, ",")
	initQuestionTest(t, "\n")

	userTags := make(map[string]string)
	userTags["Key1"] = "Value1"
	userTags["Key2"] = "Value2"
	userTags["Key3"] = "Value3"
	userTags["Key4"] = "Value4"

	answer, err := question.AskUserTags(testEC2, userTags)
	actualTags := strings.Split(answer, ",")

	th.Assert(t, len(actualTags) == 4, "ActualTags length should be 4")
	for _, expectedTag := range expectedTags {
		thisTagMatches := false
		for _, actualTag := range actualTags {
			if expectedTag == actualTag {
				th.Equals(t, expectedTag, actualTag)
				thisTagMatches = true
				break
			}
		}
		th.Assert(t, thisTagMatches, fmt.Sprintf("Unable to find matching actual tag for expected tag %s", expectedTag))
	}

	th.Ok(t, err)
	cleanupQuestionTest()
}

func TestAskCapacityType_WithDefault(t *testing.T) {
	testRegion := "us-east-1"
	defaultCapacity := question.DefaultCapacityTypeText.Spot
	initQuestionTest(t, "\n")

	answer, err := question.AskCapacityType(testInstanceType, testRegion, defaultCapacity)
	th.Equals(t, defaultCapacity, answer)

	th.Ok(t, err)
	cleanupQuestionTest()
}
