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
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var testEC2 = &ec2helper.EC2Helper{}

func TestNew(t *testing.T) {
	testEC2 = ec2helper.New(session.Must(session.NewSession()))
	if testEC2 == nil {
		t.Error("EC2Helper is not created successfully")
	}
}

/*
Region Tests
*/

const testRegion = "Test Region"

func TestChangeRegion(t *testing.T) {
	testEC2.Sess = session.Must(session.NewSession())

	testEC2.ChangeRegion(testRegion)
	if *testEC2.Sess.Config.Region != testRegion {
		t.Errorf("Region incorrect, expected: %s, got: %s", testRegion, *testEC2.Sess.Config.Region)
	}
}

func TestGetDefaultRegion_Env(t *testing.T) {
	// Backup environment variable
	backupEnv := os.Getenv(ec2helper.RegionEnv)

	os.Setenv(ec2helper.RegionEnv, testRegion)
	testEC2.Sess = session.Must(session.NewSession())
	testEC2.Sess.Config.Region = nil

	ec2helper.GetDefaultRegion(testEC2.Sess)
	if *testEC2.Sess.Config.Region != testRegion {
		t.Errorf("Region incorrect, expect: %s, got: %s", testRegion, *testEC2.Sess.Config.Region)
	}

	// Restore environment variable
	os.Setenv(ec2helper.RegionEnv, backupEnv)
}

func TestGetDefaultRegion_Default(t *testing.T) {
	// Backup environment variable
	backupEnv := os.Getenv(ec2helper.RegionEnv)

	os.Setenv(ec2helper.RegionEnv, "")
	testEC2.Sess = session.Must(session.NewSession())
	testEC2.Sess.Config.Region = nil

	ec2helper.GetDefaultRegion(testEC2.Sess)
	if *testEC2.Sess.Config.Region != ec2helper.DefaultRegion {
		t.Errorf("Region incorrect, expect: %s, got: %s", ec2helper.DefaultRegion, *testEC2.Sess.Config.Region)
	}

	// Restore environment variable
	os.Setenv(ec2helper.RegionEnv, backupEnv)
}

func TestGetEnabledRegions_Success(t *testing.T) {
	testRegions := []*ec2.Region{
		{
			RegionName: aws.String("region-b"),
		},
		{
			RegionName: aws.String("region-a"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		Regions: testRegions,
	}

	regions, err := testEC2.GetEnabledRegions()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
	if !th.Equal(regions, testRegions, ec2.Region{}) {
		t.Error("Incorrect regions")
	}
}

func TestGetEnabledRegions_DescribeRegionsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeRegionsError: errors.New("Test error"),
	}

	_, err := testEC2.GetEnabledRegions()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetEnabledRegions_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}

	_, err := testEC2.GetEnabledRegions()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

/*
Availability Zone Tests
*/

func TestGetAvailableAvailabilityZones_Success(t *testing.T) {
	testZones := []*ec2.AvailabilityZone{
		{
			ZoneName: aws.String("us-east-1a"),
		},
		{
			ZoneName: aws.String("us-east-1b"),
		},
	}

	testEC2.Svc = &th.MockedEC2Svc{
		AvailabilityZones: testZones,
	}

	zones, err := testEC2.GetAvailableAvailabilityZones()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(zones, testZones, ec2.AvailabilityZone{}) {
		t.Error("Incorrect availability zones")
	}
}

func TestGetAvailableAvailabilityZones_DescribeAvailabilityZonesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeAvailabilityZonesError: errors.New("Test error"),
	}

	_, err := testEC2.GetAvailableAvailabilityZones()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetAvailableAvailabilityZones_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}

	_, err := testEC2.GetAvailableAvailabilityZones()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

/*
Launch Template Tests
*/

func TestGetLaunchTemplatesInRegion_Success(t *testing.T) {
	testLaunchTemplates := []*ec2.LaunchTemplate{
		{
			LaunchTemplateId: aws.String("lt-12345"),
		},
		{
			LaunchTemplateId: aws.String("lt-67890"),
		},
	}
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: testLaunchTemplates,
	}

	templates, err := testEC2.GetLaunchTemplatesInRegion()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(templates, testLaunchTemplates, ec2.LaunchTemplate{}) {
		t.Error("Incorrect launch templates")
	}
}

func TestGetLaunchTemplatesInRegion_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{},
	}

	_, err := testEC2.GetLaunchTemplatesInRegion()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}
func TestGetLaunchTemplatesInRegion_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLaunchTemplatesInRegion()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	template, err := testEC2.GetLaunchTemplateById(testLaunchTemplateId)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *template.LaunchTemplateId != testLaunchTemplateId {
		t.Errorf(th.IncorrectValueFormat,
			"launch template ID", testLaunchTemplateId, *template.LaunchTemplateId)
	}
}

func TestGetLaunchTemplateById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplates: []*ec2.LaunchTemplate{},
	}

	_, err := testEC2.GetLaunchTemplateById(testLaunchTemplateId)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetLaunchTemplateById_DescribeLaunchTemplatesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplatesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLaunchTemplateById(testLaunchTemplateId)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

/*
Launch Template Version Tests
*/

func TestGetLaunchTemplateVersions_Success_AllVersions(t *testing.T) {
	testLaunchTemplateVersions := []*ec2.LaunchTemplateVersion{
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
		LaunchTemplateVersions: append(testLaunchTemplateVersions, &ec2.LaunchTemplateVersion{
			LaunchTemplateId: aws.String("lt-67890"),
			VersionNumber:    aws.Int64(1),
		}),
	}

	versions, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, nil)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(versions, testLaunchTemplateVersions, ec2.LaunchTemplateVersion{}) {
		t.Error("Incorrect launch template versions")
	}
}

func TestGetLaunchTemplateVersions_Success_OneVersion(t *testing.T) {
	versions, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, aws.String("1"))
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if len(versions) != 1 {
		t.Errorf(th.IncorrectElementNumberFormat,
			"launch template versions", 1, len(versions))
	} else if *versions[0].VersionNumber != 1 || *versions[0].LaunchTemplateId != testLaunchTemplateId {
		t.Errorf(th.IncorrectValueFormat,
			"launch template version", testLaunchTemplateId+":1",
			*versions[0].LaunchTemplateId+":"+fmt.Sprint(*versions[0].VersionNumber))
	}
}

func TestGetLaunchTemplateVersions_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		LaunchTemplateVersions: []*ec2.LaunchTemplateVersion{},
	}

	_, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, nil)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetLaunchTemplateVersions_DescribeLaunchTemplateVersionsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeLaunchTemplateVersionsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLaunchTemplateVersions(testLaunchTemplateId, nil)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *instanceType.InstanceType != freeInstanceType {
		t.Errorf(th.IncorrectValueFormat, "instance type", freeInstanceType, *instanceType.InstanceType)
	}
}

func TestGetDefaultFreeTierInstanceType_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	_, err := testEC2.GetDefaultFreeTierInstanceType()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestGetDefaultFreeTierInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetDefaultFreeTierInstanceType()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceTypesInRegion_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: testInstanceTypes,
	}

	instanceTypes, err := testEC2.GetInstanceTypesInRegion()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(instanceTypes, testInstanceTypes, ec2.InstanceTypeInfo{}) {
		t.Error("Incorrect instance types")
	}
}

func TestGetInstanceTypesInRegion_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	_, err := testEC2.GetInstanceTypesInRegion()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceTypesInRegion_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceTypesInRegion()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceType_Success(t *testing.T) {
	const testInstanceType = "t2.micro"
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: testInstanceTypes,
	}

	instanceType, err := testEC2.GetInstanceType(testInstanceType)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *instanceType.InstanceType != testInstanceType {
		t.Errorf(th.IncorrectValueFormat, "instance type", testInstanceType, *instanceType.InstanceType)
	}
}

func TestGetInstanceType_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		InstanceTypes: []*ec2.InstanceTypeInfo{},
	}

	_, err := testEC2.GetInstanceType(testInstanceType)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceType_DescribeInstanceTypesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstanceTypesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceType(testInstanceType)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

/*
Instance Selector Tests
*/

var testInstanceTypeInfos = []*ec2.InstanceTypeInfo{
	{
		InstanceType: aws.String("t2.micro"),
	},
	{
		InstanceType: aws.String("t2.nano"),
	},
}
var selector = &th.MockedSelector{
	InstanceTypes: testInstanceTypeInfos,
}

func TestGetInstanceTypesFromInstanceSelector_Success(t *testing.T) {
	instanceTypes, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, 2, 4)
	if err != nil {
		t.Fatalf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(instanceTypes, testInstanceTypeInfos, ec2.InstanceTypeInfo{}) {
		t.Error("Incorrect instance types")
	}
}

func TestGetInstanceTypesFromInstanceSelector_BadVCpus(t *testing.T) {
	_, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, -1, 4)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceTypesFromInstanceSelector_BadMemory(t *testing.T) {
	_, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, 2, -1)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceTypesFromInstanceSelector_SelectorError(t *testing.T) {
	selector = &th.MockedSelector{
		InstanceTypes: testInstanceTypeInfos,
		SelectorError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceTypesFromInstanceSelector(selector, 2, 4)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	images, err := testEC2.GetLatestImages(nil)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(*images, testMapEbs, ec2.Image{}) {
		t.Error("Incorrect images")
	}
}

func TestGetLatestImages_Success_InstanceStore(t *testing.T) {
	images, err := testEC2.GetLatestImages(aws.String("instance-store"))
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(*images, testMapInstanceStore, ec2.Image{}) {
		t.Error("Incorrect images")
	}
}

func TestGetLatestImages_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	_, err := testEC2.GetLatestImages(nil)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestGetLatestImages_DescribeImagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetLatestImages(nil)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetDefaultImage_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: testImages,
	}

	image, err := testEC2.GetDefaultImage(nil)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *image.ImageId != *lastImage.ImageId {
		t.Errorf(th.IncorrectValueFormat, "image ID", *lastImage.ImageId, *image.ImageId)
	}
}

func TestGetDefaultImage_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	_, err := testEC2.GetDefaultImage(nil)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetDefaultImage_DescribeImagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetDefaultImage(nil)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetImageById_Success(t *testing.T) {
	const testAmi = "ami-12345"
	testEC2.Svc = &th.MockedEC2Svc{
		Images: testImages,
	}

	image, err := testEC2.GetImageById(testAmi)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *image.ImageId != *testImages[0].ImageId {
		t.Errorf(th.IncorrectValueFormat, "image ID", *testImages[0].ImageId, *image.ImageId)
	}
}

func TestGetImageById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	_, err := testEC2.GetImageById("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetImageById_DescribeImagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeImagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetImageById("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	vpcs, err := testEC2.GetAllVpcs()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(vpcs, testVpcs, ec2.Vpc{}) {
		t.Error("Incorrect VPCs")
	}
}

func TestGetAllVpcs_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{},
	}

	_, err := testEC2.GetAllVpcs()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestGetAllVpcs_DescribeVpcsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetAllVpcs()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetVpcById_Success(t *testing.T) {
	const testVpcId = "vpc-12345"
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: testVpcs,
	}

	vpc, err := testEC2.GetVpcById(testVpcId)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *vpc.VpcId != testVpcId {
		t.Errorf(th.IncorrectValueFormat, "VPC ID", testVpcId, *vpc.VpcId)
	}
}

func TestGetVpcById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Vpcs: []*ec2.Vpc{},
	}

	_, err := testEC2.GetVpcById(testVpcId)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetVpcById_DescribeVpcsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeVpcsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetVpcById(testVpcId)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	subnets, err := testEC2.GetSubnetsByVpc(testVpcId)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(subnets, testSubnets, ec2.Subnet{}) {
		t.Error("Incorrect subnets")
	}
}

func TestGetSubnetsByVpc_DescribeSubnetsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSubnetsByVpc("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetSubnetsByVpc_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{},
	}

	_, err := testEC2.GetSubnetsByVpc("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	subnet, err := testEC2.GetSubnetById(testSubnetId)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *subnet.SubnetId != testSubnetId {
		t.Errorf(th.IncorrectValueFormat, "subnet ID", testSubnetId, *subnet.SubnetId)
	}
}

func TestGetSubnetById_DescribeSubnetsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSubnetById("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetSubnetById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{},
	}

	_, err := testEC2.GetSubnetById("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	securityGroups, err := testEC2.GetSecurityGroupsByIds([]string{})
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(securityGroups, testSecurityGroups, ec2.SecurityGroup{}) {
		t.Error("Incorrect security groups")
	}
}

func TestGetSecurityGroupsByIds_DescribeSecurityGroupsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSecurityGroupsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSecurityGroupsByIds([]string{})
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetSecurityGroupsByIds_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: []*ec2.SecurityGroup{},
	}

	_, err := testEC2.GetSecurityGroupsByIds([]string{})
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetSecurityGroupsByVpc_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: testSecurityGroups,
	}

	securityGroups, err := testEC2.GetSecurityGroupsByVpc("")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(securityGroups, testSecurityGroups, ec2.SecurityGroup{}) {
		t.Error("Incorrect security groups")
	}
}

func TestGetSecurityGroupsByVpc_DescribeSecurityGroupsPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSecurityGroupsPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetSecurityGroupsByVpc("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetSecurityGroupsByVpc_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		SecurityGroups: []*ec2.SecurityGroup{},
	}

	_, err := testEC2.GetSecurityGroupsByVpc("")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestCreateSecurityGroupForSsh_Success(t *testing.T) {
	_, err := testEC2.CreateSecurityGroupForSsh("")
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestCreateSecurityGroupForSsh_CreateSecurityGroupError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		CreateSecurityGroupError: errors.New("Test error"),
	}

	_, err := testEC2.CreateSecurityGroupForSsh("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateSecurityGroupForSsh_AuthorizeSecurityGroupIngressError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		AuthorizeSecurityGroupIngressError: errors.New("Test error"),
	}

	_, err := testEC2.CreateSecurityGroupForSsh("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestCreateSecurityGroupForSsh_CreateTagsError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		CreateTagsError: errors.New("Test error"),
	}

	_, err := testEC2.CreateSecurityGroupForSsh("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	instance, err := testEC2.GetInstanceById(testInstanceId)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if *instance.InstanceId != testInstanceId {
		t.Errorf(th.IncorrectValueFormat, "instance ID", testInstanceId, *instance.InstanceId)
	}
}

func TestGetInstanceById_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstanceById("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetInstanceById_NoResult(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Instances: []*ec2.Instance{},
	}

	_, err := testEC2.GetInstanceById("")
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	instances, err := testEC2.GetInstancesByState([]string{})
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if !th.Equal(instances, testInstances, ec2.Instance{}) {
		t.Error("Incorrect instances")
	}
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

	instances, err := testEC2.GetInstancesByFilter([]string{"i-12345", "i-67890"}, testFilters)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
	if len(instances) != 1 {
		t.Error("Incorrect instance(s) returned after filtering")
		fmt.Printf("instances: %v\n", instances)
	} else {
		if instances[0] != "i-12345" {
			t.Error("Incorrect instance(s) returned after filtering")
		}
	}
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

	instances, err := testEC2.GetInstancesByFilter([]string{"i-12345", "i-67890"}, testFilters)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	} else if len(instances) != 0 {
		t.Error("Instances should NOT have been returned after filtering")
	}
}

func TestGetInstancesByState_DescribeInstancesPagesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeInstancesPagesError: errors.New("Test error"),
	}

	_, err := testEC2.GetInstancesByState([]string{})
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	detailedConfig, err := testEC2.ParseConfig(&testSimpleConfig)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
	if *detailedConfig.Image.ImageId != testImageId {
		t.Errorf(th.IncorrectValueFormat, "image ID", testImageId, *detailedConfig.Image.ImageId)
	}
	if *detailedConfig.Vpc.VpcId != testVpcId {
		t.Errorf(th.IncorrectValueFormat, "VPC ID", testVpcId, *detailedConfig.Vpc.VpcId)
	}
	if *detailedConfig.Subnet.SubnetId != testSubnetId {
		t.Errorf(th.IncorrectValueFormat, "subnet ID", testSubnetId, *detailedConfig.Subnet.SubnetId)
	}
	if *detailedConfig.InstanceTypeInfo.InstanceType != testInstanceType {
		t.Errorf(th.IncorrectValueFormat, "instance type", testInstanceType, *detailedConfig.InstanceTypeInfo.InstanceType)
	}
}

func TestParseConfig_DescribeInstanceTypesPagesError(t *testing.T) {
	parseConfigSvc.DescribeInstanceTypesPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestParseConfig_DescribeImagesError(t *testing.T) {
	parseConfigSvc.DescribeImagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestParseConfig_DescribeSecurityGroupsPagesError(t *testing.T) {
	parseConfigSvc.DescribeSecurityGroupsPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestParseConfig_DescribeVpcsPagesError(t *testing.T) {
	parseConfigSvc.DescribeVpcsPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestParseConfig_DescribeSubnetsPagesError(t *testing.T) {
	parseConfigSvc.DescribeSubnetsPagesError = errors.New("Test error")

	_, err := testEC2.ParseConfig(&testSimpleConfig)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	simpleConfig, err := testEC2.GetDefaultSimpleConfig()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
	if simpleConfig.ImageId != testImageId {
		t.Errorf(th.IncorrectValueFormat, "image ID", testImageId, simpleConfig.ImageId)
	}
	if simpleConfig.SubnetId != testSubnetId {
		t.Errorf(th.IncorrectValueFormat, "subnet ID", testSubnetId, simpleConfig.SubnetId)
	}
	if simpleConfig.InstanceType != testInstanceType {
		t.Errorf(th.IncorrectValueFormat, "instance type", testInstanceType, simpleConfig.InstanceType)
	}
}

func TestGetDefaultSimpleConfig_DescribeSecurityGroupsPagesError(t *testing.T) {
	defaultConfigSvc.DescribeSecurityGroupsPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetDefaultSimpleConfig_DescribeSubnetsPagesError(t *testing.T) {
	defaultConfigSvc.DescribeSubnetsPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetDefaultSimpleConfig_NoDefaultVpc(t *testing.T) {
	defaultConfigSvc.Vpcs[0].SetIsDefault(false)

	_, err := testEC2.GetDefaultSimpleConfig()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestGetDefaultSimpleConfig_NoVpc(t *testing.T) {
	defaultConfigSvc.Vpcs = []*ec2.Vpc{}

	_, err := testEC2.GetDefaultSimpleConfig()
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestGetDefaultSimpleConfig_DescribeVpcsPagesError(t *testing.T) {
	defaultConfigSvc.DescribeVpcsPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetDefaultSimpleConfig_DescribeImagesError(t *testing.T) {
	defaultConfigSvc.DescribeImagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestGetDefaultSimpleConfig_DescribeInstanceTypesPagesError(t *testing.T) {
	defaultConfigSvc.DescribeInstanceTypesPagesError = errors.New("Test error")

	_, err := testEC2.GetDefaultSimpleConfig()
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestLaunchInstance_Success_Template(t *testing.T) {
	testSimpleConfig.LaunchTemplateId = "lt-12345"
	testSimpleConfig.LaunchTemplateVersion = "2"
	testEC2.Svc = launchSvc

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestLaunchInstance_Abort(t *testing.T) {
	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, false)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestLaunchInstance_NoConfig(t *testing.T) {
	_, err := testEC2.LaunchInstance(nil, nil, true)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestLaunchInstance_RunInstancesError(t *testing.T) {
	launchSvc.RunInstancesError = errors.New("Test error")

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestLaunchInstance_DescribeImagesError(t *testing.T) {
	launchSvc.DescribeImagesError = errors.New("Test error")

	_, err := testEC2.LaunchInstance(&testSimpleConfig, &testDetailedConfig, true)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

/*
Terminate Tests
*/

func TestTerminateInstances_Success(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{}

	err := testEC2.TerminateInstances([]string{})
	if err != nil {
		t.Errorf(th.UnexpectedErrorFormat, err)
	}
}

func TestTerminateInstances_TerminateInstancesError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		TerminateInstancesError: errors.New("Test error"),
	}

	err := testEC2.TerminateInstances([]string{})
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
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

	name := ec2helper.GetTagName(testTags)
	if *name != testName {
		t.Errorf(th.IncorrectValueFormat, "tag name", testName, *name)
	}
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

	name := ec2helper.GetTagName(testTags)
	if name != nil {
		t.Errorf(th.IncorrectValueFormat, "tag name", "nil", *name)
	}
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
	if !result {
		t.Errorf("Incorrect image validation, expect: %t, got: %t", true, result)
	}
}

func TestValidateImageId_False(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		Images: []*ec2.Image{},
	}

	result := ec2helper.ValidateImageId(testEC2, testImageId)
	if result {
		t.Errorf("Incorrect image validation, expect: %t, got: %t", false, result)
	}
}

func TestValidateFilepath_True(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "mocked_filepath")
	defer os.Remove(tmpFile.Name())
	if err != nil {
		t.Errorf("There was an error creating tempfile: %v", err)
	}
	result := ec2helper.ValidateFilepath(testEC2, tmpFile.Name())
	if !result {
		t.Errorf("Incorrect filepath validation, expect: %t, got: %t", true, result)
	}
}

func TestValidateFilepath_False(t *testing.T) {
	result := ec2helper.ValidateFilepath(testEC2, "file/does/not/exist")
	if result {
		t.Errorf("Incorrect filepath validation, expect: %t, got: %t", false, result)
	}
}

func TestValidateTags_True(t *testing.T) {
	testUserInput := "tag1|val1,tag2|val2"
	result := ec2helper.ValidateTags(testEC2, testUserInput)
	if !result {
		t.Errorf("Incorrect tag validation, expect: %t, got: %t", true, result)
	}
}

func TestValidateTags_False(t *testing.T) {
	testUserInput := "tag1|val1,tag2|val2,tag3"
	result := ec2helper.ValidateTags(testEC2, testUserInput)
	if result {
		t.Errorf("Incorrect image validation, expect: %t, got: %t", false, result)
	}
}

func TestIsLinux_True(t *testing.T) {
	if !ec2helper.IsLinux(ec2.CapacityReservationInstancePlatformLinuxUnix) {
		t.Errorf(th.IncorrectValueFormat, "IsLinux result", "true", "false")
	}
}

func TestIsLinux_False(t *testing.T) {
	if ec2helper.IsLinux(ec2.CapacityReservationInstancePlatformWindows) {
		t.Errorf(th.IncorrectValueFormat, "IsLinux result", "false", "true")
	}
}

func TestHasEbsVolume_True(t *testing.T) {
	testImage := &ec2.Image{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				Ebs: &ec2.EbsBlockDevice{},
			},
		},
	}

	if !ec2helper.HasEbsVolume(testImage) {
		t.Errorf(th.IncorrectValueFormat, "TestHasEbsVolume result", "true", "false")
	}
}

func TestHasEbsVolume_False(t *testing.T) {
	testImage := &ec2.Image{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{},
		},
	}

	if ec2helper.HasEbsVolume(testImage) {
		t.Errorf(th.IncorrectValueFormat, "TestHasEbsVolume result", "false", "true")
	}
}
