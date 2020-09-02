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
	"strconv"
	"testing"

	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/question"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var testEC2 = &ec2helper.EC2Helper{
	Sess: &session.Session{
		Config: &aws.Config{
			Region: aws.String("us-east-1"),
		},
	},
}

/*
AskQuestion Tests
*/

const correctOutput = `
This is a question[default: default option]
These are the options`

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
	if output != correctOutput {
		t.Errorf(th.IncorrectValueFormat, "question output", correctOutput+"\n", output)
	}

	if answer != testResponse {
		t.Errorf(th.IncorrectValueFormat, "response", testResponse, answer)
	}
}

func TestAskQuestion_IndexedOptionAnswer(t *testing.T) {
	input.DefaultOptionRepr = nil
	const index = "1"
	initQuestionTest(t, index+"\n")

	answer := question.AskQuestion(input)
	if answer != input.IndexedOptions[0] {
		t.Errorf(th.IncorrectValueFormat, "response", input.IndexedOptions[0], answer)
	}

	cleanupQuestionTest()
}

func TestAskQuestion_DefaultAnswer(t *testing.T) {
	initQuestionTest(t, "\n")

	answer := question.AskQuestion(input)
	if answer != *input.DefaultOption {
		t.Errorf(th.IncorrectValueFormat, "response", *input.DefaultOption, answer)
	}

	cleanupQuestionTest()
}

func TestAskQuestion_IntegerAnswer(t *testing.T) {
	const testInteger = "5"
	initQuestionTest(t, testInteger+"\n")

	answer := question.AskQuestion(input)
	if answer != testInteger {
		t.Errorf(th.IncorrectValueFormat, "response", testInteger, answer)
	}

	cleanupQuestionTest()
}

func TestAskQuestion_AnyStringAnswer(t *testing.T) {
	const testString = "any string"
	initQuestionTest(t, testString+"\n")

	answer := question.AskQuestion(input)
	if answer != testString {
		t.Errorf(th.IncorrectValueFormat, "response", testString, answer)
	}

	cleanupQuestionTest()
}

func TestAskQuestion_FunctionCheckedInput(t *testing.T) {
	const testImageId = "ami-12345"
	testEC2 := &ec2helper.EC2Helper{
		Svc: &th.MockedEC2Svc{
			Images: []*ec2.Image{
				{
					ImageId: aws.String(testImageId),
				},
			},
		},
	}
	input.EC2Helper = testEC2
	input.Fns = []question.CheckInput{
		ec2helper.ValidateImageId,
	}

	initQuestionTest(t, testImageId+"\n")

	answer := question.AskQuestion(input)
	if answer != testImageId {
		t.Errorf(th.IncorrectValueFormat, "response", testImageId, answer)
	}

	cleanupQuestionTest()
}

/*
Other Question Asking Tests
*/

func TestAskRegion_Success(t *testing.T) {
	const testRegion = "us-east-2"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		Regions: []*ec2.Region{
			{
				RegionName: aws.String(testRegion),
			},
			{
				RegionName: aws.String("us-west-1"),
			},
			{
				RegionName: aws.String("us-west-2"),
			},
		},
	}

	answer, err := question.AskRegion(testEC2)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testRegion {
		t.Errorf(th.IncorrectValueFormat, "answer", testRegion, *answer)
	}

	cleanupQuestionTest()
}

func TestAskRegion_DescribeRegionsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeRegionsError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskRegion(testEC2)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskLaunchTemplate_Success(t *testing.T) {
	const testTemplateId = "lt-12345"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{
			{
				LaunchTemplateId:    aws.String(testTemplateId),
				LaunchTemplateName:  aws.String(testTemplateId),
				LatestVersionNumber: aws.Int64(1),
			},
			{
				LaunchTemplateId:    aws.String("lt-67890"),
				LaunchTemplateName:  aws.String("lt-67890"),
				LatestVersionNumber: aws.Int64(1),
			},
		},
	}

	answer := question.AskLaunchTemplate(testEC2)
	if *answer != testTemplateId {
		t.Errorf(th.IncorrectValueFormat, "answer", testTemplateId, *answer)
	}

	cleanupQuestionTest()
}

func TestAskLaunchTemplate_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	answer := question.AskLaunchTemplate(testEC2)
	if *answer != cli.ResponseNo {
		t.Errorf(th.IncorrectValueFormat, "answer", cli.ResponseNo, *answer)
	}

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

	answer, err := question.AskLaunchTemplateVersion(testEC2, testTemplateId)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != strconv.Itoa(testVersion) {
		t.Errorf(th.IncorrectValueFormat, "answer", strconv.Itoa(testVersion), *answer)
	}

	cleanupQuestionTest()
}

func TestAskLaunchTemplateVersion_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	const testTemplateId = "lt-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskLaunchTemplateVersion(testEC2, testTemplateId)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskIfEnterInstanceType_Success(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "\n")

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(testInstanceType),
				FreeTierEligible: aws.Bool(true),
			},
		},
	}

	answer, err := question.AskIfEnterInstanceType(testEC2)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testInstanceType {
		t.Errorf(th.IncorrectValueFormat, "answer", testInstanceType, *answer)
	}

	cleanupQuestionTest()
}

func TestAskIfEnterInstanceType_(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskIfEnterInstanceType(testEC2)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceType_Success(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, testInstanceType+"\n")

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:     aws.String(testInstanceType),
				FreeTierEligible: aws.Bool(true),
			},
		},
	}

	answer, err := question.AskInstanceType(testEC2)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testInstanceType {
		t.Errorf(th.IncorrectValueFormat, "answer", testInstanceType, *answer)
	}

	cleanupQuestionTest()
}

func TestAskInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "\n")

	_, err := question.AskInstanceType(testEC2)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceTypeVCpu(t *testing.T) {
	const testVcpus = "2"
	initQuestionTest(t, testVcpus+"\n")

	answer := question.AskInstanceTypeVCpu()
	if answer != testVcpus {
		t.Errorf(th.IncorrectValueFormat, "answer", testVcpus, answer)
	}

	cleanupQuestionTest()
}

func TestAskInstanceTypeMemory(t *testing.T) {
	const testMemory = "2"
	initQuestionTest(t, testMemory+"\n")

	answer := question.AskInstanceTypeMemory()
	if answer != testMemory {
		t.Errorf(th.IncorrectValueFormat, "answer", testMemory, answer)
	}

	cleanupQuestionTest()
}

func TestAskImage_Success(t *testing.T) {
	const testImage = "ami-12345"
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{
			{
				InstanceType:             aws.String(testInstanceType),
				InstanceStorageSupported: aws.Bool(true),
			},
		},
		Images: []*ec2.Image{
			{
				ImageId:      aws.String(testImage),
				CreationDate: aws.String("some time"),
			},
		},
	}

	answer, err := question.AskImage(testEC2, testInstanceType)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer.ImageId != testImage {
		t.Errorf(th.IncorrectValueFormat, "answer", testInstanceType, *answer.ImageId)
	}

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
			},
		},
	}

	_, err := question.AskImage(testEC2, testInstanceType)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskImage_DescribeInstanceTypesPagesError(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := question.AskImage(testEC2, testInstanceType)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

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
			},
		},
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := question.AskImage(testEC2, testInstanceType)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskKeepEbsVolume(t *testing.T) {
	const testAnswer = cli.ResponseYes
	initQuestionTest(t, testAnswer+"\n")

	answer := question.AskKeepEbsVolume()
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskAutoTerminationTimerMinutes(t *testing.T) {
	const testAnswer = "30"
	initQuestionTest(t, testAnswer+"\n")

	answer := question.AskAutoTerminationTimerMinutes()
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskVpc_Success(t *testing.T) {
	const testVpc = "vpc-12345"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{
			{
				VpcId:     aws.String(testVpc),
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

	answer, err := question.AskVpc(testEC2)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testVpc {
		t.Errorf(th.IncorrectValueFormat, "answer", testVpc, *answer)
	}

	cleanupQuestionTest()
}

func TestAskVpc_DescribeVpcsPagesError(t *testing.T) {
	const testInstanceType = ec2.InstanceTypeT2Micro
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	_, err := question.AskVpc(testEC2)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskSubnet_Success(t *testing.T) {
	const testVpc = "vpc-12345"
	const testSubnet = "subnet-12345"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{
			{
				SubnetId:         aws.String(testSubnet),
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

	answer, err := question.AskSubnet(testEC2, testVpc)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testSubnet {
		t.Errorf(th.IncorrectValueFormat, "answer", testSubnet, *answer)
	}

	cleanupQuestionTest()
}

func TestAskSubnet_DescribeSubnetsPagesError(t *testing.T) {
	const testVpc = "vpc-12345"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := question.AskSubnet(testEC2, testVpc)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskSubnetPlaceholder_Success(t *testing.T) {
	const testAz = "us-east-1"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		AvailabilityZones: []*ec2.AvailabilityZone{
			{
				ZoneName: aws.String(testAz),
				ZoneId:   aws.String("some id"),
			},
			{
				ZoneName: aws.String("us-east-2"),
				ZoneId:   aws.String("some id"),
			},
		},
	}

	answer, err := question.AskSubnetPlaceholder(testEC2)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testAz {
		t.Errorf(th.IncorrectValueFormat, "answer", testAz, *answer)
	}

	cleanupQuestionTest()
}

func TestAskSubnetPlaceholder_DescribeAvailabilityZonesError(t *testing.T) {
	const testAz = "us-east-1"
	initQuestionTest(t, "1\n")

	testEC2.Svc = &th.MockedEC2Svc{
		DescribeAvailabilityZonesError: errors.New("Test error"),
	}

	_, err := question.AskSubnetPlaceholder(testEC2)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskSecurityGroups_Success(t *testing.T) {
	const testGroup = "sg-12345"

	testSecurityGroups := []*ec2.SecurityGroup{
		{
			GroupId: aws.String(testGroup),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
		{
			GroupId: aws.String("sg-67890"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
		{
			GroupId: aws.String("sg-67890"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("test group"),
				},
			},
			Description: aws.String("some description"),
		},
	}
	addedGroups := []string{*testSecurityGroups[1].GroupId}

	initQuestionTest(t, "1\n")

	answer := question.AskSecurityGroups(testSecurityGroups, addedGroups)
	if answer != testGroup {
		t.Errorf(th.IncorrectValueFormat, "answer", testGroup, answer)
	}

	cleanupQuestionTest()
}

func TestAskSecurityGroups_NoGroup(t *testing.T) {
	const testAnswer = cli.ResponseNo

	testSecurityGroups := []*ec2.SecurityGroup{}
	addedGroups := []string{}

	initQuestionTest(t, "1\n")

	answer := question.AskSecurityGroups(testSecurityGroups, addedGroups)
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskSecurityGroupPlaceholder(t *testing.T) {
	const testAnswer = cli.ResponseAll
	initQuestionTest(t, "1\n")

	answer := question.AskSecurityGroupPlaceholder()
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskComfirmationWithTemplate_Success_NoOverriding(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1
	const testAnswer = cli.ResponseYes

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

	initQuestionTest(t, testAnswer+"\n")

	answer, err := question.AskComfirmationWithTemplate(testEC2, testSimpleConfig)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, *answer)
	}

	cleanupQuestionTest()
}

func TestAskComfirmationWithTemplate_Success_Overriding(t *testing.T) {
	const testTemplateId = "lt-12345"
	const testVersion = 1
	const testAnswer = cli.ResponseYes

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

	initQuestionTest(t, testAnswer+"\n")

	answer, err := question.AskComfirmationWithTemplate(testEC2, testSimpleConfig)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, *answer)
	}

	cleanupQuestionTest()
}

func TestAskComfirmationWithTemplate_DescribeSubnetsPagesError(t *testing.T) {
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

	_, err := question.AskComfirmationWithTemplate(testEC2, testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskComfirmationWithTemplate_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	testSimpleConfig := &config.SimpleInfo{}

	initQuestionTest(t, cli.ResponseYes+"\n")

	_, err := question.AskComfirmationWithTemplate(testEC2, testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

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
	const testAnswer = cli.ResponseYes

	initQuestionTest(t, testAnswer+"\n")

	answer := question.AskConfirmationWithInput(testSimpleConfig, testDetailedConfig, true)
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskConfirmationWithInput_Success_NewInfrastructure(t *testing.T) {
	// Modify the configs for creating new infrastructure
	testSimpleConfig.NewVPC = true
	testSimpleConfig.SecurityGroupIds = []string{cli.ResponseNew}
	testSimpleConfig.AutoTerminationTimerMinutes = 0
	testSimpleConfig.SubnetId = "us-east-2"
	testDetailedConfig.SecurityGroups = nil

	const testAnswer = cli.ResponseYes

	initQuestionTest(t, testAnswer+"\n")

	answer := question.AskConfirmationWithInput(testSimpleConfig, testDetailedConfig, true)
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskSaveConfig(t *testing.T) {
	const testAnswer = cli.ResponseYes
	initQuestionTest(t, testAnswer+"\n")

	answer := question.AskSaveConfig()
	if answer != testAnswer {
		t.Errorf(th.IncorrectValueFormat, "answer", testAnswer, answer)
	}

	cleanupQuestionTest()
}

func TestAskInstanceId_Success(t *testing.T) {
	const testInstance = "i-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{
			{
				InstanceId: aws.String(testInstance),
			},
			{
				InstanceId: aws.String("i-67890"),
			},
		},
	}

	initQuestionTest(t, "1\n")

	answer, err := question.AskInstanceId(testEC2)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testInstance {
		t.Errorf(th.IncorrectValueFormat, "answer", testInstance, *answer)
	}

	cleanupQuestionTest()
}

func TestAskInstanceId_NoInstance(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{},
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceId(testEC2)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceId_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceId(testEC2)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceIds_Success(t *testing.T) {
	const testInstance = "i-12345"

	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{
			{
				InstanceId: aws.String(testInstance),
			},
			{
				InstanceId: aws.String("i-67890"),
			},
		},
	}
	addedInstances := []string{"i-67890"}

	initQuestionTest(t, "1\n")

	answer, err := question.AskInstanceIds(testEC2, addedInstances)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testInstance {
		t.Errorf(th.IncorrectValueFormat, "answer", testInstance, *answer)
	}

	cleanupQuestionTest()
}

func TestAskInstanceIds_NoInstance(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}
	addedInstances := []string{}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceIds(testEC2, addedInstances)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceIds_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}
	addedInstances := []string{}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceIds(testEC2, addedInstances)
	if err == nil {
		t.Errorf(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

/*
Instance Selector Question Tests
*/

const testInstanceType = ec2.InstanceTypeT2Micro

var testInstanceTypeInfos = []*ec2.InstanceTypeInfo{
	{
		InstanceType: aws.String(testInstanceType),
		VCpuInfo: &ec2.VCpuInfo{
			DefaultVCpus: aws.Int64(2),
		},
		MemoryInfo: &ec2.MemoryInfo{
			SizeInMiB: aws.Int64(4096),
		},
		InstanceStorageSupported: aws.Bool(false),
	},
	{
		InstanceType: aws.String("t2.nano"),
		VCpuInfo: &ec2.VCpuInfo{
			DefaultVCpus: aws.Int64(1),
		},
		MemoryInfo: &ec2.MemoryInfo{
			SizeInMiB: aws.Int64(2048),
		},
		InstanceStorageSupported: aws.Bool(false),
	},
}
var testSelector = &th.MockedSelector{
	InstanceTypes: testInstanceTypeInfos,
}

func TestAskInstanceTypeInstanceSelector_Success(t *testing.T) {
	initQuestionTest(t, "1\n")

	answer, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "4")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *answer != testInstanceType {
		t.Errorf(th.IncorrectValueFormat, "answer", testInstanceType, *answer)
	}

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_BadVcpus(t *testing.T) {
	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "a", "4")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_BadMemory(t *testing.T) {
	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "a")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_NoResult(t *testing.T) {
	testSelector = &th.MockedSelector{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "4")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func TestAskInstanceTypeInstanceSelector_SelectorError(t *testing.T) {
	testSelector = &th.MockedSelector{
		InstanceTypes: testInstanceTypeInfos,
		SelectorError: errors.New("Test error"),
	}

	initQuestionTest(t, "1\n")

	_, err := question.AskInstanceTypeInstanceSelector(testEC2, testSelector, "2", "4")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}

	cleanupQuestionTest()
}

func initQuestionTest(t *testing.T, input string) {
	err := th.TakeOverStdin(input)
	if err != nil {
		t.Fatal(err)
	}

	err = th.TakeOverStdout()
	if err != nil {
		t.Fatal(err)
	}
}

func cleanupQuestionTest() string {
	th.RestoreStdin()
	return th.ReadStdout()
}
