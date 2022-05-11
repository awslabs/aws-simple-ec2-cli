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

package ec2helper_e2e

import (
	"fmt"
	"strings"
	"testing"

	"simple-ec2/pkg/cfn"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	th "simple-ec2/test/testhelper"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/selector"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const testStackName = "simple-ec2-e2e-ec2helper-test"
const correctRegion = "us-east-2"
const testAmi = "ami-026dea5602e368e96"

var sess = session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
var c = cfn.New(sess)
var h *ec2helper.EC2Helper
var vpcId, instanceId *string
var subnetIds []string
var launchTemplateId *string
var securityGroupId *string
var resources []*cloudformation.StackResource

// This is not a test function. it just sets up the testing environment. The tests are tailored for us-east-2 region only
func TestSetupEnvironment(t *testing.T) {
	// Parse CloudFormation templates
	err := cfn.DecodeTemplateVariables()
	th.Ok(t, err)

	// The tests only work in us-east-2, so change the region if the region is not correct
	if *sess.Config.Region != correctRegion {
		sess.Config.Region = aws.String(correctRegion)
		c.Svc = cloudformation.New(sess)
	}

	vpcId, subnetIds, instanceId, resources, err = c.CreateStackAndGetResources(nil, aws.String(testStackName),
		cfn.E2eEc2helperTestCloudformationTemplate)
	th.Ok(t, err)

	// Find the launch template and the securiy group
	for _, resource := range resources {
		if *resource.ResourceType == cfn.ResourceTypeLaunchTemplate {
			launchTemplateId = resource.PhysicalResourceId
		} else if *resource.ResourceType == cfn.ResourceTypeSecurityGroup {
			securityGroupId = resource.PhysicalResourceId
		}
	}
}

func TestNew(t *testing.T) {
	h = ec2helper.New(sess)
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
}

func TestGetEnabledRegions(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	regions, err := h.GetEnabledRegions()
	th.Ok(t, err)
	th.Assert(t, regions != nil, "Regions should not be nil")
	th.Assert(t, len(regions) > 0, "Regions should not be empty")
}

func TestGetAvailableAvailabilityZones(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	zones, err := h.GetAvailableAvailabilityZones()
	th.Ok(t, err)
	th.Assert(t, zones != nil, "Zones should not be nil")
	th.Assert(t, len(zones) > 0, "Zones should not be empty")
}

func TestGetLaunchTemplatesInRegion(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	_, err := h.GetLaunchTemplatesInRegion()
	th.Ok(t, err)
}

func TestGetLaunchTemplateById(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, launchTemplateId != nil, "No test launch template found")

	template, err := h.GetLaunchTemplateById(*launchTemplateId)
	th.Ok(t, err)
	th.Assert(t, template != nil, "Launch template should not be nil")
	th.Equals(t, *launchTemplateId, *template.LaunchTemplateId)
}

func TestGetLaunchTemplateVersions(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, launchTemplateId != nil, "No test launch template found")

	versions, err := h.GetLaunchTemplateVersions(*launchTemplateId, nil)
	th.Ok(t, err)
	th.Assert(t, versions != nil, "Versions should not be nil")
	th.Assert(t, len(versions) > 0, "Versions should not be empty")
}

func TestGetDefaultFreeTierInstanceType(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	_, err := h.GetDefaultFreeTierInstanceType()
	th.Ok(t, err)
}

func TestGetInstanceTypesInRegion(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	instanceTypes, err := h.GetInstanceTypesInRegion()
	th.Ok(t, err)
	th.Assert(t, instanceTypes != nil, "InstanceTypes should not be nil")
	th.Assert(t, len(instanceTypes) > 0, "InstanceTypes should not be empty")
}

func TestGetInstanceType(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	instanceType, err := h.GetInstanceType("t2.micro")
	th.Ok(t, err)
	th.Assert(t, instanceType != nil, "InstanceType should not be nil")
}

func TestGetInstanceTypesFromInstanceSelector(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	instanceSelector := selector.New(h.Sess)
	_, err := h.GetInstanceTypesFromInstanceSelector(instanceSelector, 2, 4)
	th.Ok(t, err)
}

func TestGetLatestImages(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	_, err := h.GetLatestImages(nil, aws.StringSlice([]string{"x86_64"}))
	th.Ok(t, err)
}

func TestGetDefaultImageForAmd(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	image, err := h.GetDefaultImage(nil, aws.StringSlice([]string{"x86_64"}))
	th.Ok(t, err)
	th.Assert(t, image != nil, "Image should not be nil")
}

func TestGetDefaultImageForArm(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	image, err := h.GetDefaultImage(nil, aws.StringSlice([]string{"arm64"}))
	th.Ok(t, err)
	th.Assert(t, image != nil, "Image should not be nil")
}

func TestGetImageById(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	image, err := h.GetImageById(testAmi)
	th.Ok(t, err)
	th.Assert(t, image != nil, "Image should not be nil")
}

func TestGetAllVpcs(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	vpcs, err := h.GetAllVpcs()
	th.Ok(t, err)
	th.Assert(t, vpcs != nil, "Vpcs should not be nil")
	th.Assert(t, len(vpcs) > 0, "Vpcs should not be empty")
}

func TestGetVpcById(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, vpcId != nil, "No test VPC found")

	vpc, err := h.GetVpcById(*vpcId)
	th.Ok(t, err)
	th.Assert(t, vpc != nil, "Vpc should not be nil")
	th.Equals(t, *vpcId, *vpc.VpcId)
}

func TestGetSubnetsByVpc(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, vpcId != nil, "No test VPC found")

	subnets, err := h.GetSubnetsByVpc(*vpcId)
	th.Ok(t, err)
	th.Assert(t, subnets != nil, "subnets should not be nil")
	th.Assert(t, len(subnets) > 0, "subnets should not be empty")
}

func TestGetSubnetById(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, subnetIds != nil, "No test subnets found")
	th.Assert(t, len(subnetIds) > 0, "No test subnets found")

	subnetId := subnetIds[0]
	subnet, err := h.GetSubnetById(subnetId)
	th.Ok(t, err)
	th.Assert(t, subnet != nil, "subnet should not be nil")
	th.Equals(t, subnetId, *subnet.SubnetId)
}

func TestGetSecurityGroupsByIds(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, securityGroupId != nil, "No test security group found")

	securityGroups, err := h.GetSecurityGroupsByIds([]string{*securityGroupId})
	th.Ok(t, err)
	th.Assert(t, securityGroups != nil, "securityGroups should not be nil")
	th.Assert(t, len(securityGroups) > 0, "securityGroups should not be empty")
	th.Equals(t, *securityGroupId, *securityGroups[0].GroupId)
}

func TestGetSecurityGroupsByVpc(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, vpcId != nil, "No test VPC found")

	securityGroups, err := h.GetSecurityGroupsByVpc(*vpcId)
	th.Ok(t, err)
	th.Assert(t, securityGroups != nil, "securityGroupsByVpc should not be nil")
	th.Assert(t, len(securityGroups) > 0, "securityGroupsByVpc should not be empty")
}

func TestCreateSecurityGroupForSsh(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, vpcId != nil, "No test VPC found")

	// Create the security group
	newSecurityGroupId, err := h.CreateSecurityGroupForSsh(*vpcId)
	th.Ok(t, err)
	th.Assert(t, newSecurityGroupId != nil, "new security group should not be nil")

	// Verify the new security group
	securityGroups, err := h.GetSecurityGroupsByIds([]string{*newSecurityGroupId})
	th.Ok(t, err)
	th.Assert(t, securityGroups != nil, "securityGroupsForSsh should not be nil")
	th.Assert(t, len(securityGroups) > 0, "securityGroupsForSsh should not be empty")
	th.Equals(t, *vpcId, *securityGroups[0].VpcId)

	// Clean up the security group
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: newSecurityGroupId,
	}
	_, err = h.Svc.DeleteSecurityGroup(input)
	th.Ok(t, err)
}

func TestGetInstanceById(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, instanceId != nil, "No test instance ID found")

	instance, err := h.GetInstanceById(*instanceId)
	th.Ok(t, err)
	th.Assert(t, instance != nil, "instance should not be nil")
	th.Equals(t, *instanceId, *instance.InstanceId)
}

func TestGetInstancesByState(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	states := []string{
		ec2.InstanceStateNamePending,
		ec2.InstanceStateNameRunning,
		ec2.InstanceStateNameStopping,
		ec2.InstanceStateNameStopped,
	}
	instances, err := h.GetInstancesByState(states)
	th.Ok(t, err)
	th.Assert(t, instances != nil, "instancesByState should not be nil")
	th.Assert(t, len(instances) > 0, "instancesByState should not be empty")
}

func TestParseConfig(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, subnetIds != nil, "No test subnets found")
	th.Assert(t, len(subnetIds) > 0, "No test subnets found")
	th.Assert(t, securityGroupId != nil, "No test security group found")

	instanceType := "t2.micro"
	subnetId := subnetIds[0]
	testSimpleConfig := &config.SimpleInfo{
		SubnetId: subnetId,
		SecurityGroupIds: []string{
			*securityGroupId,
		},
		ImageId:      testAmi,
		InstanceType: instanceType,
	}

	detailedConfig, err := h.ParseConfig(testSimpleConfig)
	th.Ok(t, err)
	th.Equals(t, subnetId, *detailedConfig.Subnet.SubnetId)
	th.Equals(t, *vpcId, *detailedConfig.Vpc.VpcId)
	th.Equals(t, *securityGroupId, *detailedConfig.SecurityGroups[0].GroupId)
	th.Equals(t, testAmi, *detailedConfig.Image.ImageId)
	th.Equals(t, instanceType, *detailedConfig.InstanceTypeInfo.InstanceType)
}

func TestGetDefaultSimpleConfig(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	simpleConfig, err := h.GetDefaultSimpleConfig()
	th.Ok(t, err)
	th.Assert(t, simpleConfig.InstanceType != "", "InstanceType should not be empty")
	th.Assert(t, simpleConfig.ImageId != "", "ImageId should not be empty")
}

func TestLaunchInstance(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, subnetIds != nil, "No test subnets found")
	th.Assert(t, len(subnetIds) > 0, "No test subnets found")
	th.Assert(t, securityGroupId != nil, "No test security group found")

	// Create instances
	instanceType := "t2.micro"
	subnetId := subnetIds[0]
	testSimpleConfig := &config.SimpleInfo{
		SubnetId: subnetId,
		SecurityGroupIds: []string{
			*securityGroupId,
		},
		ImageId:      testAmi,
		InstanceType: instanceType,
	}

	detailedConfig, err := h.ParseConfig(testSimpleConfig)
	th.Ok(t, err)

	instanceIds, err := h.LaunchInstance(testSimpleConfig, detailedConfig, true)
	th.Ok(t, err)

	// Defer the clean up so that even if one of the assertions fail, we still terminate the instance
	defer func() {
		input := &ec2.TerminateInstancesInput{
			InstanceIds: aws.StringSlice(instanceIds),
		}
		_, err = h.Svc.TerminateInstances(input)
		th.Ok(t, err)
	}()

	th.Assert(t, instanceIds != nil, "instanceIds should not be nil")
	th.Assert(t, len(instanceIds) > 0, "instanceIds should not be empty")

	for _, instanceID := range instanceIds {
		ValidateInstanceMatchesDesiredSpecs(t, instanceID, testSimpleConfig)
	}
}

func ValidateInstanceMatchesDesiredSpecs(t *testing.T, instanceID string, simpleConfig *config.SimpleInfo) {
	instance, err := h.GetInstanceById(instanceID)
	if err != nil {
		th.Nok(t, err)
	}
	th.Assert(t, strings.EqualFold(*instance.InstanceType, simpleConfig.InstanceType), "Instance type does not match")
	th.Assert(t, strings.EqualFold(*instance.SubnetId, simpleConfig.SubnetId), "Subnet ID does not match")
	ValidateInstanceTags(t, instance.Tags, simpleConfig.UserTags)
}

func ValidateInstanceTags(t *testing.T, actualTags []*ec2.Tag, expectedTags map[string]string) {
	countOfExpectedTags := len(expectedTags)
	countOfActualTagsMatched := 0
	for _, tag := range actualTags {
		if val, ok := expectedTags[*tag.Key]; ok {
			th.Assert(t, strings.EqualFold(*tag.Value, val), fmt.Sprintf("Tag values for key %s don't match (expected: %s, actual: %s)", *tag.Key, val, *tag.Value))
			countOfActualTagsMatched++
		}
	}
	th.Assert(t, countOfExpectedTags == countOfActualTagsMatched, "Didn't find all of the expected tags on the actual instance")
}

func TestTerminateInstances(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")
	th.Assert(t, instanceId != nil, "No test instance ID found")

	err := h.TerminateInstances([]string{*instanceId})
	th.Ok(t, err)
}

func TestValidateImageId(t *testing.T) {
	th.Assert(t, h != nil, "EC2Helper was not initialized successfully")

	isValid := ec2helper.ValidateImageId(h, testAmi)
	th.Equals(t, true, isValid)
}

func TestCleanupEnvironment(t *testing.T) {
	err := c.DeleteStack(testStackName)
	th.Ok(t, err)
}
