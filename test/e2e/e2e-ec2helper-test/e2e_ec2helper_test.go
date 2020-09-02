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
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"

	"simple-ec2/pkg/cfn"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	th "simple-ec2/test/testhelper"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/selector"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
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
	if err != nil {
		t.Fatal(err)
	}

	// The tests only work in us-east-2, so change the region if the region is not correct
	if *sess.Config.Region != correctRegion {
		sess.Config.Region = aws.String(correctRegion)
		c.Svc = cloudformation.New(sess)
	}

	vpcId, subnetIds, instanceId, resources, err = c.CreateStackAndGetResources(nil, aws.String(testStackName),
		cfn.E2eEc2helperTestCloudformationTemplate)
	if err != nil {
		t.Fatal(err)
	}

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

	checkClient(t)
}

func TestGetEnabledRegions(t *testing.T) {
	checkClient(t)

	regions, err := h.GetEnabledRegions()
	if err != nil {
		t.Fatal(err)
	} else if regions == nil || len(regions) <= 0 {
		t.Fatal("Incorrect regions: empty result")
	}
}

func TestGetAvailableAvailabilityZones(t *testing.T) {
	checkClient(t)

	zones, err := h.GetAvailableAvailabilityZones()
	if err != nil {
		t.Fatal(err)
	} else if zones == nil || len(zones) <= 0 {
		t.Fatal("Incorrect availability zones: empty result")
	}
}

func TestGetLaunchTemplatesInRegion(t *testing.T) {
	checkClient(t)

	_, err := h.GetLaunchTemplatesInRegion()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLaunchTemplateById(t *testing.T) {
	checkClient(t)
	checkLaunchTemplateId(t)

	template, err := h.GetLaunchTemplateById(*launchTemplateId)
	if err != nil {
		t.Fatal(err)
	} else if template == nil {
		t.Fatal("Incorrect launch template: empty result")
	} else if *launchTemplateId != *template.LaunchTemplateId {
		t.Errorf(th.IncorrectValueFormat, "launch template ID", *launchTemplateId,
			*template.LaunchTemplateId)
	}
}

func TestGetLaunchTemplateVersions(t *testing.T) {
	checkClient(t)
	checkLaunchTemplateId(t)

	versions, err := h.GetLaunchTemplateVersions(*launchTemplateId, nil)
	if err != nil {
		t.Fatal(err)
	} else if versions == nil || len(versions) <= 0 {
		t.Fatal("Incorrect launch template versions: empty result")
	}
}

func TestGetDefaultFreeTierInstanceType(t *testing.T) {
	checkClient(t)

	_, err := h.GetDefaultFreeTierInstanceType()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetInstanceTypesInRegion(t *testing.T) {
	checkClient(t)

	instanceTypes, err := h.GetInstanceTypesInRegion()
	if err != nil {
		t.Fatal(err)
	} else if instanceTypes == nil || len(instanceTypes) <= 0 {
		t.Fatal("Incorrect instance types: empty result")
	}
}

func TestGetInstanceType(t *testing.T) {
	checkClient(t)

	instanceType, err := h.GetInstanceType("t2.micro")
	if err != nil {
		t.Fatal(err)
	} else if instanceType == nil {
		t.Fatal("Incorrect instance type: empty result")
	}
}

func TestGetInstanceTypesFromInstanceSelector(t *testing.T) {
	checkClient(t)

	instanceSelector := selector.New(h.Sess)
	_, err := h.GetInstanceTypesFromInstanceSelector(instanceSelector, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLatestImages(t *testing.T) {
	checkClient(t)

	_, err := h.GetLatestImages(nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDefaultImage(t *testing.T) {
	checkClient(t)

	image, err := h.GetDefaultImage(nil)
	if err != nil {
		t.Fatal(err)
	} else if image == nil {
		t.Fatal("Incorrect image: empty result")
	}
}

func TestGetImageById(t *testing.T) {
	checkClient(t)

	image, err := h.GetImageById(testAmi)
	if err != nil {
		t.Fatal(err)
	} else if image == nil {
		t.Fatal("Incorrect image: empty result")
	}
}

func TestGetAllVpcs(t *testing.T) {
	checkClient(t)

	vpcs, err := h.GetAllVpcs()
	if err != nil {
		t.Fatal(err)
	} else if vpcs == nil || len(vpcs) <= 0 {
		t.Fatal("Incorrect VPCs: empty result")
	}
}

func TestGetVpcById(t *testing.T) {
	checkClient(t)
	checkVpcId(t)

	vpc, err := h.GetVpcById(*vpcId)
	if err != nil {
		t.Fatal(err)
	} else if vpc == nil {
		t.Fatal("Incorrect VPC: empty result")
	} else if *vpcId != *vpc.VpcId {
		t.Errorf(th.IncorrectValueFormat, "VPC ID", *vpcId, *vpc.VpcId)
	}
}

func TestGetSubnetsByVpc(t *testing.T) {
	checkClient(t)
	checkVpcId(t)

	subnets, err := h.GetSubnetsByVpc(*vpcId)
	if err != nil {
		t.Fatal(err)
	} else if subnets == nil || len(subnets) <= 0 {
		t.Fatal("Incorrect subnets: empty result")
	}
}

func TestGetSubnetById(t *testing.T) {
	checkClient(t)
	checkSubnetIds(t)

	subnetId := subnetIds[0]
	subnet, err := h.GetSubnetById(subnetId)
	if err != nil {
		t.Fatal(err)
	} else if subnet == nil {
		t.Fatal("Incorrect subnet: empty result")
	} else if *subnet.SubnetId != subnetId {
		t.Errorf(th.IncorrectValueFormat, "subnet ID", subnetId, *subnet.SubnetId)
	}
}

func TestGetSecurityGroupsByIds(t *testing.T) {
	checkClient(t)
	checkSecurityGroupId(t)

	securityGroups, err := h.GetSecurityGroupsByIds([]string{*securityGroupId})
	if err != nil {
		t.Fatal(err)
	} else if securityGroups == nil || len(securityGroups) <= 0 {
		t.Fatal("Incorrect security groups: empty result")
	} else if *securityGroups[0].GroupId != *securityGroupId {
		t.Errorf(th.IncorrectValueFormat, "security group ID", *securityGroupId,
			*securityGroups[0].GroupId)
	}
}

func TestGetSecurityGroupsByVpc(t *testing.T) {
	checkClient(t)
	checkVpcId(t)

	securityGroups, err := h.GetSecurityGroupsByVpc(*vpcId)
	if err != nil {
		t.Fatal(err)
	} else if securityGroups == nil || len(securityGroups) <= 0 {
		t.Fatal("Incorrect security groups: empty result")
	}
}

func TestCreateSecurityGroupForSsh(t *testing.T) {
	checkClient(t)
	checkVpcId(t)

	// Create the security group
	newSecurityGroupId, err := h.CreateSecurityGroupForSsh(*vpcId)
	if err != nil {
		t.Fatal(err)
	} else if newSecurityGroupId == nil {
		t.Fatal("Incorrect new security group: empty result")
	}

	// Verify the new security group
	securityGroups, err := h.GetSecurityGroupsByIds([]string{*newSecurityGroupId})
	if err != nil {
		t.Error(err)
	} else if securityGroups == nil || len(securityGroups) <= 0 {
		t.Error("Incorrect security groups: empty result")
	} else if *securityGroups[0].VpcId != *vpcId {
		t.Errorf(th.IncorrectValueFormat, "VPC ID", *vpcId, *securityGroups[0].VpcId)
	}

	// Clean up the security group
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: newSecurityGroupId,
	}
	_, err = h.Svc.DeleteSecurityGroup(input)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstanceById(t *testing.T) {
	checkClient(t)
	checkInstanceId(t)

	instance, err := h.GetInstanceById(*instanceId)
	if err != nil {
		t.Fatal(err)
	} else if instance == nil {
		t.Fatal("Incorrect instance: empty result")
	} else if *instance.InstanceId != *instanceId {
		t.Errorf(th.IncorrectValueFormat, "instance ID", *instanceId, *instance.InstanceId)
	}
}

func TestGetInstancesByState(t *testing.T) {
	checkClient(t)

	states := []string{
		ec2.InstanceStateNamePending,
		ec2.InstanceStateNameRunning,
		ec2.InstanceStateNameStopping,
		ec2.InstanceStateNameStopped,
	}
	instances, err := h.GetInstancesByState(states)
	if err != nil {
		t.Fatal(err)
	} else if instances == nil || len(instances) <= 0 {
		t.Fatal("Incorrect instances: empty result")
	}
}

func TestParseConfig(t *testing.T) {
	checkClient(t)
	checkSubnetIds(t)
	checkSecurityGroupId(t)

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
	if err != nil {
		t.Fatal(err)
	} else {
		if *detailedConfig.Subnet.SubnetId != subnetId {
			t.Errorf(th.IncorrectValueFormat, "subnet ID", subnetId, *detailedConfig.Subnet.SubnetId)
		}
		if *detailedConfig.Vpc.VpcId != *vpcId {
			t.Errorf(th.IncorrectValueFormat, "VPC ID", *vpcId, *detailedConfig.Vpc.VpcId)
		}
		if *detailedConfig.SecurityGroups[0].GroupId != *securityGroupId {
			t.Errorf(th.IncorrectValueFormat, "security group ID", *securityGroupId,
				*detailedConfig.SecurityGroups[0].GroupId)
		}
		if *detailedConfig.Image.ImageId != testAmi {
			t.Errorf(th.IncorrectValueFormat, "image ID", testAmi, *detailedConfig.Image.ImageId)
		}
		if *detailedConfig.InstanceTypeInfo.InstanceType != instanceType {
			t.Errorf(th.IncorrectValueFormat, "instance type", instanceType,
				*detailedConfig.InstanceTypeInfo.InstanceType)
		}
	}
}

func TestGetDefaultSimpleConfig(t *testing.T) {
	checkClient(t)

	simpleConfig, err := h.GetDefaultSimpleConfig()
	if err != nil {
		t.Fatal(err)
	} else {
		if simpleConfig.InstanceType == "" {
			t.Fatal("Incorrect instance type: empty result")
		}
		if simpleConfig.ImageId == "" {
			t.Fatal("Incorrect image: empty result")
		}
	}
}

func TestLaunchInstance(t *testing.T) {
	checkClient(t)
	checkSubnetIds(t)
	checkSecurityGroupId(t)

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
	instanceIds, err := h.LaunchInstance(testSimpleConfig, nil, true)
	if err != nil {
		t.Fatal(err)
	} else if instanceIds == nil || len(instanceIds) <= 0 {
		t.Fatal("Incorrect instance IDs: empty result")
	}

	// Clean up
	input := &ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	}
	_, err = h.Svc.TerminateInstances(input)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTerminateInstances(t *testing.T) {
	checkClient(t)
	checkInstanceId(t)

	err := h.TerminateInstances([]string{*instanceId})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateImageId(t *testing.T) {
	checkClient(t)

	isValid := ec2helper.ValidateImageId(h, testAmi)
	if !isValid {
		t.Errorf(th.IncorrectValueFormat, "image validation", "true", "false")
	}
}

func TestCleanupEnvironment(t *testing.T) {
	err := c.DeleteStack(testStackName)
	if err != nil {
		t.Error(err)
	}
}

func checkClient(t *testing.T) {
	if h == nil {
		t.Fatal("EC2Helper is not initialized successfully")
	}
}

func checkLaunchTemplateId(t *testing.T) {
	if launchTemplateId == nil {
		t.Fatal("No test launch template found")
	}
}

func checkVpcId(t *testing.T) {
	if vpcId == nil {
		t.Fatal("No test VPC found")
	}
}

func checkSubnetIds(t *testing.T) {
	if subnetIds == nil || len(subnetIds) <= 0 {
		t.Fatal("No test subnet found")
	}
}

func checkSecurityGroupId(t *testing.T) {
	if securityGroupId == nil {
		t.Fatal("No test security group found")
	}
}

func checkInstanceId(t *testing.T) {
	if instanceId == nil {
		t.Fatal("No test instance ID found")
	}
}
