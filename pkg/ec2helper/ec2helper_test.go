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

package ec2helper_test

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	th "simple-ec2/test/testhelper"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/instancetypes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var testEC2 = &ec2helper.EC2Helper{}

func TestNew(t *testing.T) {
	testEC2 = ec2helper.New(session.Must(session.NewSession()))
	th.Assert(t, testEC2 != nil, "EC2Helper was not created successfully")
}

/*
Region Tests
*/

const testRegion = "Test Region"

func TestChangeRegion(t *testing.T) {
	testEC2.Sess = session.Must(session.NewSession())
	testEC2.ChangeRegion(testRegion)
	th.Equals(t, testRegion, *testEC2.Sess.Config.Region)
}

func TestGetDefaultRegion_Env(t *testing.T) {
	// Backup environment variable
	backupEnv := os.Getenv(ec2helper.RegionEnv)

	os.Setenv(ec2helper.RegionEnv, testRegion)
	testEC2.Sess = session.Must(session.NewSession())
	testEC2.Sess.Config.Region = nil
	ec2helper.GetDefaultRegion(testEC2.Sess)

	// Restore environment variable here in case assert fails
	os.Setenv(ec2helper.RegionEnv, backupEnv)
	th.Equals(t, testRegion, *testEC2.Sess.Config.Region)
}

func TestGetDefaultRegion_Default(t *testing.T) {
	// Backup environment variable
	backupEnv := os.Getenv(ec2helper.RegionEnv)

	os.Setenv(ec2helper.RegionEnv, "")
	testEC2.Sess = session.Must(session.NewSession())
	testEC2.Sess.Config.Region = nil
	ec2helper.GetDefaultRegion(testEC2.Sess)

	// Restore environment variable
	os.Setenv(ec2helper.RegionEnv, backupEnv)
	th.Equals(t, ec2helper.DefaultRegion, *testEC2.Sess.Config.Region)
}

func TestGetEnabledRegions_Success(t *testing.T) {
	expectedRegions := []*ec2.Region{
		{
			RegionName: aws.String("region-b"),
		},
		{
			RegionName: aws.String("region-a"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Regions: expectedRegions,
	}

	actualRegions, err := testEC2.GetEnabledRegions()
	th.Ok(t, err)
	th.Equals(t, expectedRegions, actualRegions)
}

func TestGetEnabledRegions_DescribeRegionsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeRegionsError: errors.New("Test error"),
	}

	_, err := testEC2.GetEnabledRegions()
	th.Nok(t, err)
}

func TestGetEnabledRegions_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}

	_, err := testEC2.GetEnabledRegions()
	th.Nok(t, err)
}

/*
Availability Zone Tests
*/

func TestGetAvailableAvailabilityZones_Success(t *testing.T) {
	expectedZones := []*ec2.AvailabilityZone{
		{
			ZoneName: aws.String("us-east-1a"),
		},
		{
			ZoneName: aws.String("us-east-1b"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		AvailabilityZones: expectedZones,
	}

	actualZones, err := testEC2.GetAvailableAvailabilityZones()
	th.Ok(t, err)
	th.Equals(t, expectedZones, actualZones)
}

func TestGetAvailableAvailabilityZones_DescribeAvailabilityZonesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeAvailabilityZonesError: errors.New("Test error"),
	}

	_, err := testEC2.GetAvailableAvailabilityZones()
	th.Nok(t, err)
}

func TestGetAvailableAvailabilityZones_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}

	_, err := testEC2.GetAvailableAvailabilityZones()
	th.Nok(t, err)
}

/*
Launch Template Tests
*/

var testLaunchId = "lt-12345"

func TestGetLaunchTemplatesInRegion_Success(t *testing.T) {
	expectedTemplates := []*ec2.LaunchTemplate{
		{
			LaunchTemplateId: aws.String("lt-12345"),
		},
		{
			LaunchTemplateId: aws.String("lt-67890"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: expectedTemplates,
	}

	actualTemplates, err := testEC2.GetLaunchTemplatesInRegion()
	th.Ok(t, err)
	th.Equals(t, expectedTemplates, actualTemplates)
}

func TestGetLaunchTemplatesInRegion_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{},
	}

	_, err := testEC2.GetLaunchTemplatesInRegion()
	th.Ok(t, err)
}
func TestGetLaunchTemplatesInRegion_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLaunchTemplatesInRegion()
	th.Nok(t, err)
}

const testLaunchTemplateId = "lt-12345"

func TestGetLaunchTemplateById_Success(t *testing.T) {
	testLaunchTemplates := []*ec2.LaunchTemplate{
		{
			LaunchTemplateId: aws.String(testLaunchTemplateId),
		},
		{
			LaunchTemplateId: aws.String("lt-67890"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: testLaunchTemplates,
	}

	actualLaunchTemplates, err := testEC2.GetLaunchTemplateById(testLaunchTemplateId)
	th.Ok(t, err)
	th.Equals(t, testLaunchTemplateId, *actualLaunchTemplates.LaunchTemplateId)
}

func TestGetLaunchTemplateById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{},
	}

	_, err := testEC2.GetLaunchTemplateById(testLaunchTemplateId)
	th.Nok(t, err)
}

func TestGetLaunchTemplateById_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLaunchTemplateById(testLaunchTemplateId)
	th.Nok(t, err)
}

func TestCreateLaunchTemplate(t *testing.T) {
	config := config.NewSimpleInfo()
	config.ImageId = "ami-12345"
	config.InstanceType = "t2.micro"
	config.SubnetId = "subnet-12345"
	testEC2.Svc = &th.MockedEC2Svc{}
	testEC2.CreateLaunchTemplate(config)

	launchTemplatesOutput, _ := testEC2.Svc.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{})
	templates := launchTemplatesOutput.LaunchTemplates
	th.Equals(t, 1, len(templates))
	th.Equals(t, testLaunchId, *templates[0].LaunchTemplateId)
}

func TestDeleteLaunchTemplate(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{
			{LaunchTemplateId: &testLaunchId},
		},
	}
	testEC2.DeleteLaunchTemplate(&testLaunchId)

	launchTemplatesOutput, _ := testEC2.Svc.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{})
	templates := launchTemplatesOutput.LaunchTemplates

	th.Equals(t, 0, len(templates))
}

/*
Launch Template Version Tests
*/

func TestGetLaunchTemplateVersions_Success_AllVersions(t *testing.T) {
	expectedVersions := []*ec2.LaunchTemplateVersion{
		{
			LaunchTemplateId: aws.String(testLaunchTemplateId),
			VersionNumber:    aws.Int64(1),
		},
		{
			LaunchTemplateId: aws.String(testLaunchTemplateId),
			VersionNumber:    aws.Int64(2),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: append(expectedVersions, &ec2.LaunchTemplateVersion{
			LaunchTemplateId: aws.String("lt-67890"),
			VersionNumber:    aws.Int64(1),
		}),
	}

	actualVersions, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, nil)
	th.Ok(t, err)
	th.Equals(t, expectedVersions, actualVersions)
}

func TestGetLaunchTemplateVersions_Success_OneVersion(t *testing.T) {
	versions, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, aws.String("1"))

	th.Ok(t, err)
	th.Equals(t, 1, len(versions))
	th.Equals(t, 1, int(*versions[0].VersionNumber))
	th.Equals(t, testLaunchTemplateId, *versions[0].LaunchTemplateId)
}

func TestGetLaunchTemplateVersions_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{},
	}

	_, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, nil)
	th.Nok(t, err)
}

func TestGetLaunchTemplateVersions_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, nil)
	th.Nok(t, err)
}

/*
Instance Type Tests
*/

const freeInstanceType = "t2.micro"

var testInstanceTypes = []*ec2.InstanceTypeInfo{
	{
		InstanceType:     aws.String(freeInstanceType),
		FreeTierEligible: aws.Bool(true),
	},
	{
		InstanceType:     aws.String("t2.nano"),
		FreeTierEligible: aws.Bool(false),
	},
}

func TestGetDefaultFreeTierInstanceType_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: testInstanceTypes,
	}

	instanceType, err := testEC2.GetDefaultFreeTierInstanceType()
	th.Ok(t, err)
	th.Equals(t, freeInstanceType, *instanceType.InstanceType)
}

func TestGetDefaultFreeTierInstanceType_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	_, err := testEC2.GetDefaultFreeTierInstanceType()
	th.Ok(t, err)
}

func TestGetDefaultFreeTierInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetDefaultFreeTierInstanceType()
	th.Nok(t, err)
}

func TestGetInstanceTypesInRegion_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: testInstanceTypes,
	}

	actualInstanceTypes, err := testEC2.GetInstanceTypesInRegion()
	th.Ok(t, err)
	th.Equals(t, testInstanceTypes, actualInstanceTypes)
}

func TestGetInstanceTypesInRegion_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	_, err := testEC2.GetInstanceTypesInRegion()
	th.Nok(t, err)
}

func TestGetInstanceTypesInRegion_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceTypesInRegion()
	th.Nok(t, err)
}

func TestGetInstanceType_Success(t *testing.T) {
	const testInstanceType = "t2.micro"
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: testInstanceTypes,
	}

	instanceType, err := testEC2.GetInstanceType(testInstanceType)
	th.Ok(t, err)
	th.Equals(t, testInstanceType, *instanceType.InstanceType)
}

func TestGetInstanceType_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	_, err := testEC2.GetInstanceType(testInstanceType)
	th.Nok(t, err)
}

func TestGetInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceType(testInstanceType)
	th.Nok(t, err)
}

/*
Instance Selector Tests
*/

var testInstanceTypeInfos = []*instancetypes.Details{
	{
		InstanceTypeInfo: ec2.InstanceTypeInfo{
			InstanceType: aws.String("t2.micro"),
		},
	},
	{
		InstanceTypeInfo: ec2.InstanceTypeInfo{
			InstanceType: aws.String("t2.nano"),
		},
	},
}
var selector = &th.MockedSelector{
	InstanceTypes: testInstanceTypeInfos,
}

func TestGetInstanceTypesFromInstanceSelector_Success(t *testing.T) {
	actualInstanceTypes, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, 2, 4)
	th.Ok(t, err)
	th.Equals(t, testInstanceTypeInfos, actualInstanceTypes)
}

func TestGetInstanceTypesFromInstanceSelector_BadVCpus(t *testing.T) {
	_, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, -1, 4)
	th.Nok(t, err)
}

func TestGetInstanceTypesFromInstanceSelector_BadMemory(t *testing.T) {
	_, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, 2, -1)
	th.Nok(t, err)
}

func TestGetInstanceTypesFromInstanceSelector_SelectorError(t *testing.T) {
	selector = &th.MockedSelector{
		InstanceTypes: testInstanceTypeInfos,
		SelectorError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, 2, 4)
	th.Nok(t, err)
}

/*
Image Tests
*/

var lastImage = &ec2.Image{
	ImageId:      aws.String("ami-67890"),
	CreationDate: aws.String("1"),
}
var testImages = []*ec2.Image{
	{
		ImageId:      aws.String("ami-12345"),
		CreationDate: aws.String("0"),
	},
	lastImage,
}
var testMapEbs = map[string]*ec2.Image{
	"Amazon Linux":   lastImage,
	"Amazon Linux 2": lastImage,
	"Red Hat":        lastImage,
	"SUSE Linux":     lastImage,
	"Ubuntu":         lastImage,
	"Windows":        lastImage,
}
var testMapInstanceStore = map[string]*ec2.Image{
	"Amazon Linux": lastImage,
	"Ubuntu":       lastImage,
}

func TestGetLatestImages_Success_Ebs(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: testImages,
	}

	actualImages, err := testEC2.GetLatestImages(nil, aws.StringSlice([]string{"x86_64"}))
	th.Ok(t, err)
	th.Equals(t, testMapEbs, *actualImages)
}

func TestGetLatestImages_Success_InstanceStore(t *testing.T) {
	actualImages, err := testEC2.GetLatestImages(aws.String("instance-store"), aws.StringSlice([]string{"x86_64"}))
	th.Ok(t, err)
	th.Equals(t, testMapInstanceStore, *actualImages)
}

func TestGetLatestImages_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	_, err := testEC2.GetLatestImages(nil, aws.StringSlice([]string{"x86_64"}))
	th.Ok(t, err)
}

func TestGetLatestImages_DescribeImagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLatestImages(nil, aws.StringSlice([]string{"x86_64"}))
	th.Nok(t, err)
}

func TestGetDefaultImage_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: testImages,
	}

	actualImage, err := testEC2.GetDefaultImage(nil, aws.StringSlice([]string{"x86_64"}))
	th.Ok(t, err)
	th.Equals(t, *lastImage.ImageId, *actualImage.ImageId)
}

func TestGetDefaultImage_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	_, err := testEC2.GetDefaultImage(nil, aws.StringSlice([]string{"x86_64"}))
	th.Nok(t, err)
}

func TestGetDefaultImage_DescribeImagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetDefaultImage(nil, aws.StringSlice([]string{"x86_64"}))
	th.Nok(t, err)
}

func TestGetImageById_Success(t *testing.T) {
	const testAmi = "ami-12345"
	testEC2.Svc = &th.MockedEC2Svc{
		Images: testImages,
	}

	actualImage, err := testEC2.GetImageById(testAmi)
	th.Ok(t, err)
	th.Equals(t, *testImages[0].ImageId, *actualImage.ImageId)
}

func TestGetImageById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	_, err := testEC2.GetImageById("")
	th.Nok(t, err)
}

func TestGetImageById_DescribeImagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetImageById("")
	th.Nok(t, err)
}

/*
VPC Tests
*/

var testVpcs = []*ec2.Vpc{
	{
		VpcId: aws.String("vpc-12345"),
	},
	{
		VpcId: aws.String("vpc-67890"),
	},
}

func TestGetAllVpcs_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: testVpcs,
	}

	actualVpcs, err := testEC2.GetAllVpcs()
	th.Ok(t, err)
	th.Equals(t, testVpcs, actualVpcs)
}

func TestGetAllVpcs_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{},
	}

	_, err := testEC2.GetAllVpcs()
	//TODO: some _NoResult apis return error; some don't
	th.Ok(t, err)
}

func TestGetAllVpcs_DescribeVpcsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetAllVpcs()
	th.Nok(t, err)
}

func TestGetVpcById_Success(t *testing.T) {
	const testVpcId = "vpc-12345"
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: testVpcs,
	}

	actualVpc, err := testEC2.GetVpcById(testVpcId)
	th.Ok(t, err)
	th.Equals(t, testVpcId, *actualVpc.VpcId)
}

func TestGetVpcById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{},
	}

	_, err := testEC2.GetVpcById(testVpcId)
	th.Nok(t, err)
}

func TestGetVpcById_DescribeVpcsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetVpcById(testVpcId)
	th.Nok(t, err)
}

/*
Subnet Tests
*/

func TestGetSubnetsByVpc_Success(t *testing.T) {
	const testVpcId = "vpc-12345"
	testSubnets := []*ec2.Subnet{
		{
			SubnetId: aws.String("subnet-12345"),
			VpcId:    aws.String(testVpcId),
		},
		{
			SubnetId: aws.String("subnet-67890"),
			VpcId:    aws.String(testVpcId),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: testSubnets,
	}

	actualSubnets, err := testEC2.GetSubnetsByVpc(testVpcId)
	th.Ok(t, err)
	th.Equals(t, testSubnets, actualSubnets)
}

func TestGetSubnetsByVpc_DescribeSubnetsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSubnetsByVpc("")
	th.Nok(t, err)
}

func TestGetSubnetsByVpc_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{},
	}

	_, err := testEC2.GetSubnetsByVpc("")
	th.Nok(t, err)
}

func TestGetSubnetById_Success(t *testing.T) {
	const testSubnetId = "subnet-12345"
	testSubnets := []*ec2.Subnet{
		{
			SubnetId: aws.String(testSubnetId),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: testSubnets,
	}

	actualSubnet, err := testEC2.GetSubnetById(testSubnetId)
	th.Ok(t, err)
	th.Equals(t, testSubnetId, *actualSubnet.SubnetId)
}

func TestGetSubnetById_DescribeSubnetsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSubnetById("")
	th.Nok(t, err)
}

func TestGetSubnetById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{},
	}

	_, err := testEC2.GetSubnetById("")
	th.Nok(t, err)
}

/*
Security Group Tests
*/

var testSecurityGroups = []*ec2.SecurityGroup{
	{
		GroupId: aws.String("sg-12345"),
	},
	{
		GroupId: aws.String("sg-67890"),
	},
}

func TestGetSecurityGroupsByIds_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: testSecurityGroups,
	}

	actualSecurityGroups, err := testEC2.GetSecurityGroupsByIds([]string{})
	th.Ok(t, err)
	th.Equals(t, testSecurityGroups, actualSecurityGroups)
}

func TestGetSecurityGroupsByIds_DescribeSecurityGroupsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSecurityGroupsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSecurityGroupsByIds([]string{})
	th.Nok(t, err)
}

func TestGetSecurityGroupsByIds_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: []*ec2.SecurityGroup{},
	}

	_, err := testEC2.GetSecurityGroupsByIds([]string{})
	th.Nok(t, err)
}

func TestGetSecurityGroupsByVpc_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: testSecurityGroups,
	}

	actualSecurityGroups, err := testEC2.GetSecurityGroupsByVpc("")
	th.Ok(t, err)
	th.Equals(t, testSecurityGroups, actualSecurityGroups)
}

func TestGetSecurityGroupsByVpc_DescribeSecurityGroupsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSecurityGroupsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSecurityGroupsByVpc("")
	th.Nok(t, err)
}

func TestGetSecurityGroupsByVpc_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: []*ec2.SecurityGroup{},
	}

	_, err := testEC2.GetSecurityGroupsByVpc("")
	th.Ok(t, err)
}

func TestCreateSecurityGroupForSsh_Success(t *testing.T) {
	_, err := testEC2.CreateSecurityGroupForSsh("")
	th.Ok(t, err)
}

func TestCreateSecurityGroupForSsh_CreateSecurityGroupError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		CreateSecurityGroupError: errors.New("Test error"),
	}

	_, err := testEC2.CreateSecurityGroupForSsh("")
	th.Nok(t, err)
}

func TestCreateSecurityGroupForSsh_AuthorizeSecurityGroupIngressError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		AuthorizeSecurityGroupIngressError: errors.New("Test error"),
	}

	_, err := testEC2.CreateSecurityGroupForSsh("")
	th.Nok(t, err)
}

func TestCreateSecurityGroupForSsh_CreateTagsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		CreateTagsError: errors.New("Test error"),
	}

	_, err := testEC2.CreateSecurityGroupForSsh("")
	th.Nok(t, err)
}

/*
Instance Tests
*/

func TestGetInstanceById_Success(t *testing.T) {
	const testInstanceId = ("i-12345")
	testInstances := []*ec2.Instance{
		{
			InstanceId: aws.String(testInstanceId),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: testInstances,
	}

	actualInstance, err := testEC2.GetInstanceById(testInstanceId)
	th.Ok(t, err)
	th.Equals(t, testInstanceId, *actualInstance.InstanceId)
}

func TestGetInstanceById_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceById("")
	th.Nok(t, err)
}

func TestGetInstanceById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{},
	}

	_, err := testEC2.GetInstanceById("")
	th.Nok(t, err)
}

func TestGetInstancesByState_Success(t *testing.T) {
	testInstances := []*ec2.Instance{
		{
			InstanceId: aws.String("i-12345"),
		},
		{
			InstanceId: aws.String("i-67890"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: testInstances,
	}

	actualInstances, err := testEC2.GetInstancesByState([]string{})
	th.Ok(t, err)
	th.Equals(t, testInstances, actualInstances)
}

func TestGetInstancesByFilter_Success(t *testing.T) {
	testTags := []*ec2.Tag{
		{
			Key:   aws.String("TestedBy"),
			Value: aws.String("meh"),
		},
	}
	testInstances := []*ec2.Instance{
		{
			InstanceId: aws.String("i-12345"),
			Tags:       testTags,
		},
		{
			InstanceId: aws.String("i-67890"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: testInstances,
	}
	testFilters := []*ec2.Filter{
		{
			Name:   aws.String("tag:TestedBy"),
			Values: aws.StringSlice([]string{"meh"}),
		},
	}

	actualInstances, err := testEC2.GetInstancesByFilter([]string{"i-12345", "i-67890"}, testFilters)
	th.Ok(t, err)
	th.Equals(t, 1, len(actualInstances))
	th.Equals(t, "i-12345", actualInstances[0])
}

func TestGetInstancesByFilter_NoResults(t *testing.T) {
	testInstances := []*ec2.Instance{
		{
			InstanceId: aws.String("i-12345"),
		},
		{
			InstanceId: aws.String("i-67890"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: testInstances,
	}

	testFilters := []*ec2.Filter{
		{
			Name:   aws.String("tag:TestedBy"),
			Values: aws.StringSlice([]string{"meh"}),
		},
	}

	actualInstances, err := testEC2.GetInstancesByFilter([]string{"i-12345", "i-67890"}, testFilters)
	th.Ok(t, err)
	th.Equals(t, 0, len(actualInstances))
}

func TestGetInstancesByState_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstancesByState([]string{})
	th.Nok(t, err)
}

/*
Config Tests
*/

const testSubnetId = "subnet-12345"
const testVpcId = "vpc-12345"
const testImageId = "ami-12345"
const testInstanceType = "t2.micro"
const testDeviceType = "ebs"

var testSecurityGroupIds = []string{
	"sg-12345",
	"sg-67890",
}
var testSimpleConfig = config.SimpleInfo{
	SubnetId:         testSubnetId,
	ImageId:          testImageId,
	InstanceType:     testInstanceType,
	SecurityGroupIds: testSecurityGroupIds,
}

var parseConfigSvc = &th.MockedEC2Svc{
	Subnets: []*ec2.Subnet{
		{
			SubnetId: aws.String(testSubnetId),
			VpcId:    aws.String(testVpcId),
		},
	},
	Vpcs: []*ec2.Vpc{
		{
			VpcId: aws.String(testVpcId),
		},
	},
	Images: []*ec2.Image{
		{
			ImageId:        aws.String(testImageId),
			RootDeviceType: aws.String(testDeviceType),
		},
	},
	InstanceTypes: []*ec2.InstanceTypeInfo{
		{
			InstanceType: aws.String(testInstanceType),
		},
	},
	SecurityGroups: []*ec2.SecurityGroup{
		{
			GroupId: aws.String(testSecurityGroupIds[0]),
		},
		{
			GroupId: aws.String(testSecurityGroupIds[1]),
		},
	},
}

func TestParseConfig_Success(t *testing.T) {
	testEC2.Svc = parseConfigSvc

	actualDetailedConfig, err := testEC2.ParseConfig(&testSimpleConfig)
	th.Ok(t, err)
	th.Equals(t, testImageId, *actualDetailedConfig.Image.ImageId)
	th.Equals(t, testVpcId, *actualDetailedConfig.Vpc.VpcId)
	th.Equals(t, testSubnetId, *actualDetailedConfig.Subnet.SubnetId)
	th.Equals(t, testInstanceType, *actualDetailedConfig.InstanceTypeInfo.InstanceType)
}

func TestParseConfig_DescribeInstanceTypesPagesError(t *testing.T) {
	parseConfigSvc.DescribeInstanceTypesPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	th.Nok(t, err)
}

func TestParseConfig_DescribeImagesError(t *testing.T) {
	parseConfigSvc.DescribeImagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	th.Nok(t, err)
}

func TestParseConfig_DescribeSecurityGroupsPagesError(t *testing.T) {
	parseConfigSvc.DescribeSecurityGroupsPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	th.Nok(t, err)
}

func TestParseConfig_DescribeVpcsPagesError(t *testing.T) {
	parseConfigSvc.DescribeVpcsPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	th.Nok(t, err)
}

func TestParseConfig_DescribeSubnetsPagesError(t *testing.T) {
	parseConfigSvc.DescribeSubnetsPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	th.Nok(t, err)
}

var defaultConfigSvc = &th.MockedEC2Svc{
	InstanceTypes: []*ec2.InstanceTypeInfo{
		{
			InstanceType:             aws.String(testInstanceType),
			FreeTierEligible:         aws.Bool(true),
			InstanceStorageSupported: aws.Bool(true),
		},
	},
	Images: []*ec2.Image{
		{
			ImageId: aws.String(testImageId),
		},
	},
	Vpcs: []*ec2.Vpc{
		{
			VpcId:     aws.String(testVpcId),
			IsDefault: aws.Bool(true),
		},
	},
	Subnets: []*ec2.Subnet{
		{
			SubnetId: aws.String(testSubnetId),
			VpcId:    aws.String(testVpcId),
		},
	},
	SecurityGroups: []*ec2.SecurityGroup{
		{
			GroupId: aws.String(testSecurityGroupIds[0]),
		},
		{
			GroupId: aws.String(testSecurityGroupIds[1]),
		},
	},
}

func TestGetDefaultSimpleConfig_Success(t *testing.T) {
	testEC2.Svc = defaultConfigSvc

	actualSimpleConfig, err := testEC2.GetDefaultSimpleConfig()
	th.Ok(t, err)
	th.Equals(t, testImageId, actualSimpleConfig.ImageId)
	th.Equals(t, testSubnetId, actualSimpleConfig.SubnetId)
	th.Equals(t, testInstanceType, actualSimpleConfig.InstanceType)
}

func TestGetDefaultSimpleConfig_DescribeSecurityGroupsPagesError(t *testing.T) {
	defaultConfigSvc.DescribeSecurityGroupsPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Nok(t, err)
}

func TestGetDefaultSimpleConfig_DescribeSubnetsPagesError(t *testing.T) {
	defaultConfigSvc.DescribeSubnetsPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Nok(t, err)
}

func TestGetDefaultSimpleConfig_NoDefaultVpc(t *testing.T) {
	defaultConfigSvc.Vpcs[0].SetIsDefault(false)

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Ok(t, err)
}

func TestGetDefaultSimpleConfig_NoVpc(t *testing.T) {
	defaultConfigSvc.Vpcs = []*ec2.Vpc{}

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Ok(t, err)
}

func TestGetDefaultSimpleConfig_DescribeVpcsPagesError(t *testing.T) {
	defaultConfigSvc.DescribeVpcsPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Nok(t, err)
}

func TestGetDefaultSimpleConfig_DescribeImagesError(t *testing.T) {
	defaultConfigSvc.DescribeImagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Nok(t, err)
}

func TestGetDefaultSimpleConfig_DescribeInstanceTypesPagesError(t *testing.T) {
	defaultConfigSvc.DescribeInstanceTypesPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	th.Nok(t, err)
}

/*
Launch Tests
*/

var launchSvc = &th.MockedEC2Svc{
	Subnets: []*ec2.Subnet{
		{
			SubnetId: aws.String(testSubnetId),
			VpcId:    aws.String(testVpcId),
		},
	},
	Vpcs: []*ec2.Vpc{
		{
			VpcId: aws.String(testVpcId),
		},
	},
	Images: []*ec2.Image{
		{
			ImageId:        aws.String(testImageId),
			RootDeviceType: aws.String("ebs"),
		},
	},
	InstanceTypes: []*ec2.InstanceTypeInfo{
		{
			InstanceType: aws.String(testInstanceType),
		},
	},
	SecurityGroups: []*ec2.SecurityGroup{
		{
			GroupId: aws.String(testSecurityGroupIds[0]),
		},
		{
			GroupId: aws.String(testSecurityGroupIds[1]),
		},
	},
}

var testDetailedConfig = config.DetailedInfo{
	Image: &ec2.Image{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				Ebs: &ec2.EbsBlockDevice{},
			},
		},
		PlatformDetails: aws.String(ec2.CapacityReservationInstancePlatformLinuxUnix),
	},
}

func TestLaunchInstance_Success_NoTemplate(t *testing.T) {
	testEC2.Svc = launchSvc
	testSimpleConfig.AutoTerminationTimerMinutes = 5
	testSimpleConfig.KeepEbsVolumeAfterTermination = true

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	th.Ok(t, err)
}

func TestLaunchInstance_Success_Template(t *testing.T) {
	testSimpleConfig.LaunchTemplateId = "lt-12345"
	testSimpleConfig.LaunchTemplateVersion = "2"
	testEC2.Svc = launchSvc

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	th.Ok(t, err)
}

func TestLaunchInstance_Abort(t *testing.T) {
	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, false)
	th.Nok(t, err)
}

func TestLaunchInstance_NoConfig(t *testing.T) {
	_, err := testEC2.LaunchInstance(nil, nil, true)
	th.Nok(t, err)
}

func TestLaunchInstance_RunInstancesError(t *testing.T) {
	launchSvc.RunInstancesError = errors.New("Test error")

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	th.Nok(t, err)
}

func TestLaunchInstance_DescribeImagesError(t *testing.T) {
	launchSvc.DescribeImagesError = errors.New("Test error")

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	th.Nok(t, err)
}

/*
Terminate Tests
*/

func TestTerminateInstances_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}

	err := testEC2.TerminateInstances([]string{})
	th.Ok(t, err)
}

func TestTerminateInstances_TerminateInstancesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		TerminateInstancesError: errors.New("Test error"),
	}

	err := testEC2.TerminateInstances([]string{})
	th.Nok(t, err)
}

/*
Tag Tests
*/

func TestGetTagName_Success(t *testing.T) {
	const testName = "Test Name"
	testTags := []*ec2.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(testName),
		},
		{
			Key:   aws.String("CreatedBy"),
			Value: aws.String("simple-ec2"),
		},
	}

	actualName := ec2helper.GetTagName(testTags)
	th.Equals(t, testName, *actualName)
}

func TestGetTagName_NoResult(t *testing.T) {
	testTags := []*ec2.Tag{
		{
			Key:   aws.String("CreatedTime"),
			Value: aws.String("012345"),
		},
		{
			Key:   aws.String("CreatedBy"),
			Value: aws.String("simple-ec2"),
		},
	}

	actualName := ec2helper.GetTagName(testTags)
	th.Equals(t, (*string)(nil), actualName)
}

/*
Validation Tests
*/

func TestValidateImageId_True(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{
			{
				ImageId: aws.String(testImageId),
			},
		},
	}

	result := ec2helper.ValidateImageId(testEC2, testImageId)
	th.Equals(t, true, result)
}

func TestValidateImageId_False(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	result := ec2helper.ValidateImageId(testEC2, testImageId)
	th.Equals(t, false, result)
}

func TestValidateFilepath_True(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "mocked_filepath")
	defer os.Remove(tmpFile.Name())
	if err != nil {
		t.Errorf("There was an error creating tempfile: %v", err)
	}
	result := ec2helper.ValidateFilepath(testEC2, tmpFile.Name())
	th.Equals(t, true, result)
}

func TestValidateFilepath_False(t *testing.T) {
	result := ec2helper.ValidateFilepath(testEC2, "file/does/not/exist")
	th.Equals(t, false, result)
}

func TestValidateTags_True(t *testing.T) {
	testUserInput := "tag1|val1,tag2|val2"
	result := ec2helper.ValidateTags(testEC2, testUserInput)
	th.Equals(t, true, result)
}

func TestValidateTags_False(t *testing.T) {
	testUserInput := "tag1|val1,tag2|val2,tag3"
	result := ec2helper.ValidateTags(testEC2, testUserInput)
	th.Equals(t, false, result)
}

func TestIsLinux_True(t *testing.T) {
	actualIsLinux := ec2helper.IsLinux(ec2.CapacityReservationInstancePlatformLinuxUnix)
	th.Equals(t, true, actualIsLinux)
}

func TestIsLinux_False(t *testing.T) {
	actualIsLinux := ec2helper.IsLinux(ec2.CapacityReservationInstancePlatformWindows)
	th.Equals(t, false, actualIsLinux)
}

func TestHasEbsVolume_True(t *testing.T) {
	testImage := &ec2.Image{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				Ebs: &ec2.EbsBlockDevice{},
			},
		},
	}

	actualHasEbsVolume := ec2helper.HasEbsVolume(testImage)
	th.Equals(t, true, actualHasEbsVolume)
}

func TestHasEbsVolume_False(t *testing.T) {
	testImage := &ec2.Image{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{},
		},
	}

	actualHasEbsVolume := ec2helper.HasEbsVolume(testImage)
	th.Equals(t, false, actualHasEbsVolume)
}
