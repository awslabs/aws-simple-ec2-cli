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

package ec2helper

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"simple-ec2/pkg/cfn"
	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/tag"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/bytequantity"
	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/instancetypes"
	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/selector"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
)

const DefaultRegion = "us-east-2"
const tagNameKey = "Name"
const RegionEnv = "AWS_DEFAULT_REGION"
const cpuArchitecture = "x86_64"

func New(sess *session.Session) *EC2Helper {
	return &EC2Helper{
		Svc:  ec2.New(sess),
		Sess: sess,
	}
}

/*
Given a new region, change the region in session and reinitialize client,
if the new region value is different from the previous region value
*/
func (h *EC2Helper) ChangeRegion(newRegion string) {
	if newRegion != *h.Sess.Config.Region {
		h.Sess.Config.Region = &newRegion
		h.Svc = ec2.New(h.Sess)
	}
}

// Get the appropriate region, put it into the session
func GetDefaultRegion(sess *session.Session) string {
	// If a region is not picked up by the SDK, try to decide a region
	if sess.Config.Region == nil {
		// Try the environment variable
		envRegion := os.Getenv(RegionEnv)
		if envRegion != "" {
			sess.Config.Region = &envRegion
		} else {
			// Fallback to the hardcoded region value
			sess.Config.Region = aws.String(DefaultRegion)
		}
	}

	return *sess.Config.Region
}

// Sort interface for launch instance versions
type byRegionName []*ec2.Region

func (a byRegionName) Len() int           { return len(a) }
func (a byRegionName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRegionName) Less(i, j int) bool { return *a[i].RegionName < *a[j].RegionName }

/*
Get all regions enabled for the account, sorted by region name.
Empty result is not allowed.
*/
func (h *EC2Helper) GetEnabledRegions() ([]*ec2.Region, error) {
	input := &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false),
	}

	output, err := h.Svc.DescribeRegions(input)
	if err != nil {
		return nil, err
	}
	if output == nil || output.Regions == nil || len(output.Regions) <= 0 {
		return nil, errors.New("No enabled region available")
	}

	sort.Sort(byRegionName(output.Regions))

	return output.Regions, nil
}

/*
Get all available availability zone.
Empty result is not allowed.
*/
func (h *EC2Helper) GetAvailableAvailabilityZones() ([]*ec2.AvailabilityZone, error) {
	input := &ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: aws.StringSlice([]string{ec2.AvailabilityZoneStateAvailable}),
			},
		},
	}

	azOutput, err := h.Svc.DescribeAvailabilityZones(input)
	if err != nil {
		return nil, err
	}
	if azOutput == nil || azOutput.AvailabilityZones == nil || len(azOutput.AvailabilityZones) <= 0 {
		return nil, errors.New("No availability zone available")
	}

	return azOutput.AvailabilityZones, nil
}

/*
Get all launch templates.
Empty result is allowed.
*/
func (h *EC2Helper) GetLaunchTemplatesInRegion() ([]*ec2.LaunchTemplate, error) {
	input := &ec2.DescribeLaunchTemplatesInput{}

	launchTemplates, err := h.getLaunchTemplates(input)
	if err != nil {
		return nil, err
	}
	if len(launchTemplates) <= 0 {
		return nil, nil
	}

	return launchTemplates, nil
}

/*
Get the launch template with the specified launch template id.
Empty result is not allowed.
*/
func (h *EC2Helper) GetLaunchTemplateById(launchTemplateId string) (*ec2.LaunchTemplate, error) {
	input := &ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateIds: []*string{
			&launchTemplateId,
		},
	}

	launchTemplates, err := h.getLaunchTemplates(input)
	if err != nil {
		return nil, err
	}
	if len(launchTemplates) <= 0 {
		return nil, errors.New("Launch template is not found")
	}

	return (launchTemplates)[0], nil
}

// Get the launch templates based on input, with all pages concatenated
func (h *EC2Helper) getLaunchTemplates(input *ec2.DescribeLaunchTemplatesInput) ([]*ec2.LaunchTemplate, error) {

	allLaunchTemplate := []*ec2.LaunchTemplate{}

	err := h.Svc.DescribeLaunchTemplatesPages(input, func(page *ec2.DescribeLaunchTemplatesOutput, lastPage bool) bool {
		allLaunchTemplate = append(allLaunchTemplate, page.LaunchTemplates...)
		return !lastPage
	})

	return allLaunchTemplate, err
}

// Sort interface for launch instance versions
type byVersionNumber []*ec2.LaunchTemplateVersion

func (a byVersionNumber) Len() int           { return len(a) }
func (a byVersionNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byVersionNumber) Less(i, j int) bool { return *a[i].VersionNumber < *a[j].VersionNumber }

/*
Get launch template versions by a launch template id, with all pages concatenated.
Empty result is not allowed.
*/
func (h *EC2Helper) GetLaunchTemplateVersions(launchTemplateId string,
	versionId *string) ([]*ec2.LaunchTemplateVersion, error) {
	input := &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: aws.String(launchTemplateId),
	}

	if versionId != nil {
		input.Versions = []*string{versionId}
	}

	allVersions := []*ec2.LaunchTemplateVersion{}

	err := h.Svc.DescribeLaunchTemplateVersionsPages(input, func(page *ec2.DescribeLaunchTemplateVersionsOutput,
		lastPage bool) bool {
		allVersions = append(allVersions, page.LaunchTemplateVersions...)
		return !lastPage
	})
	if err != nil {
		return nil, err
	}
	if len(allVersions) <= 0 {
		return nil, errors.New("No launch template version available")
	}

	sort.Sort(byVersionNumber(allVersions))
	return allVersions, nil
}

/*
Get a default instance type, which is a free-tier eligible type.
Empty result is allowed.
*/
func (h *EC2Helper) GetDefaultFreeTierInstanceType() (*ec2.InstanceTypeInfo, error) {
	input := &ec2.DescribeInstanceTypesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("free-tier-eligible"),
				Values: []*string{
					aws.String("true"),
				},
			},
		},
	}

	instanceTypes, err := h.getInstanceTypes(input)
	if err != nil {
		return nil, err
	}
	if len(instanceTypes) <= 0 {
		return nil, nil
	}

	// Simply return the first available free instance type
	return (instanceTypes)[0], nil
}

/*
Get all instance types available in this region.
Empty result is not allowed.
*/
func (h *EC2Helper) GetInstanceTypesInRegion() ([]*ec2.InstanceTypeInfo, error) {
	input := &ec2.DescribeInstanceTypesInput{}

	instanceTypes, err := h.getInstanceTypes(input)
	if err != nil {
		return nil, err
	}
	if len(instanceTypes) <= 0 {
		return nil, errors.New("No instance type available in region")
	}

	return instanceTypes, nil
}

/*
Get the specified instance type info given an instance type name.
Empty result is not allowed.
*/
func (h *EC2Helper) GetInstanceType(instanceType string) (*ec2.InstanceTypeInfo, error) {
	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []*string{
			aws.String(instanceType),
		},
	}

	instanceTypes, err := h.getInstanceTypes(input)
	if err != nil {
		return nil, err
	}
	if len(instanceTypes) <= 0 {
		return nil, errors.New("Instance type " + instanceType + " is not available")
	}

	return instanceTypes[0], err
}

/*
Get the instance types selected by instance selector.
Empty result is allowed.
*/
func (h *EC2Helper) GetInstanceTypesFromInstanceSelector(instanceSelector InstanceSelector, vcpus,
	memoryGib int) ([]*instancetypes.Details, error) {
	if vcpus <= 0 {
		return nil, errors.New("Invalid vCPUs: " + fmt.Sprint(vcpus))
	}
	if memoryGib <= 0 {
		return nil, errors.New("Invalid memory: " + fmt.Sprint(memoryGib))
	}

	vcpusLower, vcpusUpper := vcpus-1, vcpus+1
	memoryLower, memoryUpper := uint64(memoryGib-1), uint64(memoryGib+1)

	// Create filters for filtering instance types
	vcpusRange := selector.IntRangeFilter{
		LowerBound: vcpusLower,
		UpperBound: vcpusUpper,
	}
	memoryRange := selector.ByteQuantityRangeFilter{
		LowerBound: bytequantity.FromGiB(memoryLower),
		UpperBound: bytequantity.FromGiB(memoryUpper),
	}

	// Create a Filter struct with criteria you would like to filter
	// The full struct definition can be found here for all of the supported filters:
	// https://github.com/aws/amazon-ec2-instance-selector/blob/main/pkg/selector/types.go
	filters := selector.Filters{
		VCpusRange:      &vcpusRange,
		MemoryRange:     &memoryRange,
		CPUArchitecture: aws.String(cpuArchitecture),
	}

	// Pass the Filter struct to the Filter function of your selector instance
	instanceTypesSlice, err := instanceSelector.FilterVerbose(filters)
	if err != nil {
		return nil, err
	}

	return instanceTypesSlice, nil
}

// Get the instance types based on input, with all pages concatenated
func (h *EC2Helper) getInstanceTypes(input *ec2.DescribeInstanceTypesInput) ([]*ec2.InstanceTypeInfo, error) {

	allInstanceTypes := []*ec2.InstanceTypeInfo{}

	err := h.Svc.DescribeInstanceTypesPages(input, func(page *ec2.DescribeInstanceTypesOutput, lastPage bool) bool {
		allInstanceTypes = append(allInstanceTypes, page.InstanceTypes...)
		return !lastPage
	})

	return allInstanceTypes, err
}

// Define all OS and corresponding AMI name formats
var osDescs = map[string]map[string]string{
	"Amazon Linux": {
		"ebs":            "amzn-ami-hvm-????.??.?.????????.?-*-gp2",
		"instance-store": "amzn-ami-hvm-????.??.?.????????.?-*-s3",
	},
	"Amazon Linux 2": {
		"ebs": "amzn2-ami-hvm-2.?.????????.?-*-gp2",
	},
	"Red Hat": {
		"ebs": "RHEL-?.?.?_HVM-????????-*-?-Hourly2-GP2",
	},
	"SUSE Linux": {
		"ebs": "suse-sles-??-sp?-v????????-hvm-ssd-*",
	},
	// Ubuntu 18.04 LTS
	"Ubuntu": {
		"ebs":            "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-*-server-????????",
		"instance-store": "ubuntu/images/hvm-instance/ubuntu-bionic-18.04-*-server-????????",
	},
	// 64 bit Microsoft Windows Server with Desktop Experience Locale English AMI
	"Windows": {
		"ebs": "Windows_Server-????-English-Full-Base-????.??.??",
	},
}

// Get the appropriate input for describing images
func getDescribeImagesInputs(rootDeviceType string, architectures []*string) *map[string]ec2.DescribeImagesInput {
	// Construct all the inputs
	imageInputs := map[string]ec2.DescribeImagesInput{}
	for osName, rootDeviceTypes := range osDescs {

		// Only add inputs if the corresponding root device type is applicable for the specified os
		desc, found := rootDeviceTypes[rootDeviceType]
		if found {
			imageInputs[osName] = ec2.DescribeImagesInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("name"),
						Values: []*string{
							aws.String(desc),
						},
					},
					{
						Name: aws.String("state"),
						Values: []*string{
							aws.String("available"),
						},
					},
					{
						Name: aws.String("root-device-type"),
						Values: []*string{
							aws.String(rootDeviceType),
						},
					},
					{
						Name:   aws.String("architecture"),
						Values: architectures,
					},
				},
			}
		}
	}

	return &imageInputs
}

// Sort interface for images
type byCreationDate []*ec2.Image

func (a byCreationDate) Len() int           { return len(a) }
func (a byCreationDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byCreationDate) Less(i, j int) bool { return *a[i].CreationDate < *a[j].CreationDate }

/*
Get the information about the latest AMIs.
Empty result is allowed.
*/
func (h *EC2Helper) GetLatestImages(rootDeviceType *string, architectures []*string) (*map[string]*ec2.Image, error) {
	var inputs *map[string]ec2.DescribeImagesInput
	if rootDeviceType == nil {
		inputs = getDescribeImagesInputs("ebs", architectures)
	} else {
		inputs = getDescribeImagesInputs(*rootDeviceType, architectures)
	}

	images := map[string]*ec2.Image{}
	for osName, input := range *inputs {
		output, err := h.Svc.DescribeImages(&input)
		if err != nil {
			return nil, err
		}
		if len(output.Images) <= 0 {
			continue
		}

		// Sort the images and get the latest one
		sort.Sort(byCreationDate(output.Images))
		images[osName] = output.Images[len(output.Images)-1]
	}
	if len(images) <= 0 {
		return nil, nil
	}

	return &images, nil
}

func GetImagePriority() []string {
	return []string{"Amazon Linux 2", "Ubuntu", "Amazon Linux", "Red Hat", "SUSE Linux", "Windows"}
}

/*
Get an appropriate default image, given the information about the latest AMIs.
Empty result is not allowed.
*/
func (h *EC2Helper) GetDefaultImage(rootDeviceType *string, architectures []*string) (*ec2.Image, error) {
	latestImages, err := h.GetLatestImages(rootDeviceType, architectures)
	if err != nil {
		return nil, err
	}
	if latestImages == nil || len(*latestImages) <= 0 {
		return nil, errors.New("No default image found")
	}

	var topImage *ec2.Image

	// Pick the available image with the highest priority as the default choice
	for _, osName := range GetImagePriority() {
		image, found := (*latestImages)[osName]
		if found {
			topImage = image
			break
		}
	}

	return topImage, nil
}

/*
Get the specified image by image ID.
Empty result is not allowed.
*/
func (h *EC2Helper) GetImageById(imageId string) (*ec2.Image, error) {
	input := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("state"),
				Values: []*string{
					aws.String("available"),
				},
			},
		},
		ImageIds: []*string{
			aws.String(imageId),
		},
	}

	output, err := h.Svc.DescribeImages(input)
	if err != nil {
		return nil, err
	}
	if output == nil || output.Images == nil || len(output.Images) <= 0 {
		return nil, errors.New("Image " + imageId + " is not found")
	}

	return output.Images[0], nil
}

/*
Get all VPCs.
Empty result is allowed.
*/
func (h *EC2Helper) GetAllVpcs() ([]*ec2.Vpc, error) {

	input := &ec2.DescribeVpcsInput{}

	vpcs, err := h.getVpcs(input)
	if err != nil {
		return nil, err
	}
	if len(vpcs) <= 0 {
		return nil, nil
	}

	return vpcs, err
}

/*
Get the specified VPC by VPC ID.
Empty result is not allowed.
*/
func (h *EC2Helper) GetVpcById(vpcId string) (*ec2.Vpc, error) {
	input := &ec2.DescribeVpcsInput{
		VpcIds: []*string{
			aws.String(vpcId),
		},
	}

	vpcs, err := h.getVpcs(input)
	if err != nil {
		return nil, err
	}
	if len(vpcs) <= 0 {
		return nil, errors.New("The specified VPC " + vpcId + " is not found")
	}

	return (vpcs)[0], err
}

/*
Get the default VPC. If no VPC is default, simply return nil.
Empty result is allowed.
*/
func (h *EC2Helper) getDefaultVpc() (*ec2.Vpc, error) {
	input := &ec2.DescribeVpcsInput{}

	vpcs, err := h.getVpcs(input)
	if err != nil {
		return nil, err
	}
	if len(vpcs) <= 0 {
		return nil, nil
	}

	// Find the default vpc
	for _, vpc := range vpcs {
		if *vpc.IsDefault {
			return vpc, err
		}
	}

	return nil, nil
}

// Get the VPCs based on input, with all pages concatenated
func (h *EC2Helper) getVpcs(input *ec2.DescribeVpcsInput) ([]*ec2.Vpc, error) {
	allVpcs := []*ec2.Vpc{}

	err := h.Svc.DescribeVpcsPages(input, func(page *ec2.DescribeVpcsOutput, lastPage bool) bool {
		allVpcs = append(allVpcs, page.Vpcs...)
		return !lastPage
	})

	return allVpcs, err
}

/*
Get all subnets given a VPC ID.
Empty result is not allowed.
*/
func (h *EC2Helper) GetSubnetsByVpc(vpcId string) ([]*ec2.Subnet, error) {
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	subnets, err := h.getSubnets(input)
	if err != nil {
		return nil, err
	}
	if len(subnets) <= 0 {
		return nil, errors.New("No subnet in the specified VPC " + vpcId)
	}

	return subnets, nil
}

/*
Get the specified subnet given a subnet ID.
Empty result is not allowed.
*/
func (h *EC2Helper) GetSubnetById(subnetId string) (*ec2.Subnet, error) {
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("subnet-id"),
				Values: []*string{
					aws.String(subnetId),
				},
			},
		},
	}

	subnets, err := h.getSubnets(input)

	if err != nil {
		return nil, err
	}
	if len(subnets) <= 0 {
		return nil, errors.New("Specified subnet " + subnetId + " does not exist")
	}

	// Find the only subnet in the output
	return (subnets)[0], err
}

// Get the subnets based on the input, with all pages concatenated
func (h *EC2Helper) getSubnets(input *ec2.DescribeSubnetsInput) ([]*ec2.Subnet, error) {
	allSubnets := []*ec2.Subnet{}

	err := h.Svc.DescribeSubnetsPages(input, func(page *ec2.DescribeSubnetsOutput, lastPage bool) bool {
		allSubnets = append(allSubnets, page.Subnets...)
		return !lastPage
	})

	return allSubnets, err
}

// Get the security groups based on the input, with all pages concatenated
func (h *EC2Helper) getSecurityGroups(input *ec2.DescribeSecurityGroupsInput) ([]*ec2.SecurityGroup, error) {
	allSecurityGroups := []*ec2.SecurityGroup{}

	err := h.Svc.DescribeSecurityGroupsPages(input, func(page *ec2.DescribeSecurityGroupsOutput, lastPage bool) bool {
		allSecurityGroups = append(allSecurityGroups, page.SecurityGroups...)
		return !lastPage
	})

	return allSecurityGroups, err
}

/*
Get security groups by IDs.
Empty result is not allowed.
*/
func (h *EC2Helper) GetSecurityGroupsByIds(ids []string) ([]*ec2.SecurityGroup, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(ids),
	}

	securityGroups, err := h.getSecurityGroups(input)
	if err != nil {
		return nil, err
	}
	if len(securityGroups) <= 0 {
		return nil, errors.New("The specified security groups do not exist")
	}

	return securityGroups, err
}

/*
Get the default Security Group, given a VPC ID.
Empty result is allowed.
https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-security-groups.html#default-security-group
*/
func (h *EC2Helper) getDefaultSecurityGroup(vpcId string) (*ec2.SecurityGroup, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
			{
				Name: aws.String("group-name"),
				Values: []*string{
					aws.String("default"),
				},
			},
		},
	}

	// default security group cannot be deleted. So, the result here will always include a default security group.
	defaultSg, err := h.getSecurityGroups(input)
	if err != nil {
		return nil, err
	}

	return defaultSg[0], err
}

/*
Get security groups by VPC id.
Empty result is allowed.
*/
func (h *EC2Helper) GetSecurityGroupsByVpc(vpcId string) ([]*ec2.SecurityGroup, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	securityGroups, err := h.getSecurityGroups(input)
	if err != nil {
		return nil, err
	}
	if len(securityGroups) <= 0 {
		return nil, nil
	}

	return securityGroups, nil
}

// Create a security group that enables SSH connection to instances
func (h *EC2Helper) CreateSecurityGroupForSsh(vpcId string) (*string, error) {
	fmt.Println("Creating new security group...")

	groupNameUuid := uuid.New()
	// Create a new security group
	creationInput := &ec2.CreateSecurityGroupInput{
		Description: aws.String("Created by simple-ec2 for SSH connection to instances"),
		GroupName:   aws.String(fmt.Sprintf("simple-ec2 SSH-%s", groupNameUuid)),
		VpcId:       aws.String(vpcId),
	}

	creationOutput, err := h.Svc.CreateSecurityGroup(creationInput)
	if err != nil {
		return nil, err
	}

	// Add ingress rule for SSH
	groupId := *creationOutput.GroupId
	ingressInput := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(groupId),
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(22),
				IpProtocol: aws.String("tcp"),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
				},
				Ipv6Ranges: []*ec2.Ipv6Range{
					{
						CidrIpv6: aws.String("::/0"),
					},
				},
				ToPort: aws.Int64(22),
			},
		},
	}

	_, err = h.Svc.AuthorizeSecurityGroupIngress(ingressInput)
	if err != nil {
		return nil, err
	}

	// Create tags
	tags := append(getSimpleEc2Tags(), &ec2.Tag{
		Key:   aws.String("Name"),
		Value: aws.String("simple-ec2 SSH Security Group"),
	})
	err = h.createTags([]string{groupId}, tags)
	if err != nil {
		return nil, err
	}

	fmt.Println("New security group created successfully")

	return creationOutput.GroupId, nil
}

// Get the reservations based on the input, with all pages concatenated
func (h *EC2Helper) getInstances(input *ec2.DescribeInstancesInput) ([]*ec2.Instance, error) {
	allReservations := []*ec2.Reservation{}

	err := h.Svc.DescribeInstancesPages(input, func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
		allReservations = append(allReservations, page.Reservations...)
		return !lastPage
	})

	allInstances := []*ec2.Instance{}
	for _, reservation := range allReservations {
		for _, instance := range reservation.Instances {
			allInstances = append(allInstances, instance)
		}
	}

	return allInstances, err
}

/*
Get the instance with specified instance ID.
Empty result is not allowed.
*/
func (h *EC2Helper) GetInstanceById(instanceId string) (*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{
			instanceId,
		}),
	}

	instances, err := h.getInstances(input)
	if err != nil {
		return nil, err
	}
	if len(instances) <= 0 {
		return nil, errors.New("No instance found")
	}

	return instances[0], nil
}

/*
Get all instances based on states provided.
Empty result is allowed.
*/
func (h *EC2Helper) GetInstancesByState(states []string) ([]*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: aws.StringSlice(states),
			},
		},
	}

	instances, err := h.getInstances(input)
	if err != nil {
		return nil, err
	}
	if len(instances) <= 0 {
		return nil, nil
	}

	return instances, nil
}

func (h *EC2Helper) GetInstancesByFilter(instanceIds []string, filters []*ec2.Filter) ([]string, error) {
	input := &ec2.DescribeInstancesInput{}
	if len(instanceIds) > 0 {
		input.InstanceIds = aws.StringSlice(instanceIds)
	}
	if len(filters) > 0 {
		input.Filters = filters
	}

	instances, err := h.getInstances(input)
	if err != nil {
		return nil, err
	}
	if len(instances) <= 0 {
		return nil, nil
	}

	var result []string
	for _, instance := range instances {
		result = append(result, *instance.InstanceId)
	}

	return result, nil
}

// Create tags for the resources specified
func (h *EC2Helper) createTags(resources []string, tags []*ec2.Tag) error {
	input := &ec2.CreateTagsInput{
		Resources: aws.StringSlice(resources),
		Tags:      tags,
	}

	_, err := h.Svc.CreateTags(input)
	if err != nil {
		return err
	}

	return nil
}

// Parse the simple config into detailed config
func (h *EC2Helper) ParseConfig(simpleConfig *config.SimpleInfo) (*config.DetailedInfo, error) {
	// If new VPC and subnets will be created, skip formatting subnet and vpc
	var subnet *ec2.Subnet
	var vpc *ec2.Vpc
	var securityGroups []*ec2.SecurityGroup
	var tagSpecs []*ec2.TagSpecification
	var err error
	if !simpleConfig.NewVPC {
		// Decide format of vpc and subnet
		subnet, err = h.GetSubnetById(simpleConfig.SubnetId)
		if err != nil {
			return nil, err
		}

		vpc, err = h.GetVpcById(*subnet.VpcId)
		if err != nil {
			return nil, err
		}

		securityGroups, err = h.GetSecurityGroupsByIds(simpleConfig.SecurityGroupIds)
		if err != nil {
			return nil, err
		}
	}

	// Add simple-ec2 tags to created resources
	resourceTags := getSimpleEc2Tags()
	if len(simpleConfig.UserTags) > 0 {
		for k, v := range simpleConfig.UserTags {
			resourceTags = append(resourceTags, &ec2.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
	}
	tagSpecs = []*ec2.TagSpecification{
		{
			ResourceType: aws.String("instance"),
			Tags:         resourceTags,
		},
	}
	image, err := h.GetImageById(simpleConfig.ImageId)
	if err != nil {
		return nil, err
	} else {
		if *image.RootDeviceType == "ebs" {
			tagSpecs = append(tagSpecs,
				&ec2.TagSpecification{
					ResourceType: aws.String("volume"),
					Tags:         resourceTags,
				})
		}
	}

	instanceTypeInfo, err := h.GetInstanceType(simpleConfig.InstanceType)
	if err != nil {
		return nil, err
	}

	detailedConfig := config.DetailedInfo{
		Image:            image,
		Vpc:              vpc,
		Subnet:           subnet,
		InstanceTypeInfo: instanceTypeInfo,
		SecurityGroups:   securityGroups,
		TagSpecs:         tagSpecs,
	}

	return &detailedConfig, nil
}

// Get a RunInstanceInput given a structured config
func getRunInstanceInput(simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo) *ec2.RunInstancesInput {
	dataConfig := createRequestInstanceConfig(simpleConfig, detailedConfig)
	return &ec2.RunInstancesInput{
		MaxCount:                          aws.Int64(1),
		MinCount:                          aws.Int64(1),
		LaunchTemplate:                    dataConfig.LaunchTemplate,
		ImageId:                           dataConfig.ImageId,
		InstanceType:                      dataConfig.InstanceType,
		SubnetId:                          dataConfig.SubnetId,
		SecurityGroupIds:                  dataConfig.SecurityGroupIds,
		IamInstanceProfile:                dataConfig.IamInstanceProfile,
		BlockDeviceMappings:               dataConfig.BlockDeviceMappings,
		InstanceInitiatedShutdownBehavior: dataConfig.InstanceInitiatedShutdownBehavior,
		UserData:                          dataConfig.UserData,
	}
}

// Get the default string config
func (h *EC2Helper) GetDefaultSimpleConfig() (*config.SimpleInfo, error) {
	simpleConfig := config.NewSimpleInfo()
	simpleConfig.Region = *h.Sess.Config.Region

	// get info about the instance type
	simpleConfig.InstanceType = "t2.micro"
	defaultInstanceType, err := h.GetDefaultFreeTierInstanceType()
	if err != nil {
		return nil, err
	}
	if defaultInstanceType != nil {
		simpleConfig.InstanceType = *defaultInstanceType.InstanceType
	}

	instanceTypeInfo, err := h.GetInstanceType(simpleConfig.InstanceType)
	if err != nil {
		return nil, err
	}

	// Use instance-store if supported
	rootDeviceType := "ebs"
	if *instanceTypeInfo.InstanceStorageSupported {
		rootDeviceType = "instance-store"
	}

	image, err := h.GetDefaultImage(&rootDeviceType, instanceTypeInfo.ProcessorInfo.SupportedArchitectures)
	if err != nil {
		return nil, err
	}

	simpleConfig.ImageId = *image.ImageId

	vpc, err := h.getDefaultVpc()
	if err != nil {
		return nil, err
	}

	// Only set up network configuration when default VPC exists
	if vpc != nil {
		// Simply get all subnets and pick the first available subnet
		subnets, err := h.GetSubnetsByVpc(*vpc.VpcId)
		if err != nil {
			return nil, err
		}
		subnet := subnets[0]
		simpleConfig.SubnetId = *subnet.SubnetId

		// Get the default security group
		defaultSg, err := h.getDefaultSecurityGroup(*vpc.VpcId)
		if err != nil {
			return nil, err
		}
		simpleConfig.SecurityGroupIds = []string{*defaultSg.GroupId}
	}

	simpleConfig.CapacityType = "On-Demand"

	return simpleConfig, nil
}

// Launch instances based on input and confirmation. Returning an error means failure, otherwise success
func (h *EC2Helper) LaunchInstance(simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo,
	confirmation bool) ([]string, error) {
	if simpleConfig == nil {
		return nil, errors.New("No config found")
	}

	if confirmation {
		fmt.Println("Options confirmed! Launching instance...")

		input := getRunInstanceInput(simpleConfig, detailedConfig)
		launchedInstances := []string{}

		// Create new stack, if specified.
		if simpleConfig.NewVPC {
			err := h.createNetworkConfiguration(simpleConfig, input)
			if err != nil {
				return nil, err
			}
		}

		input.TagSpecifications = detailedConfig.TagSpecs

		resp, err := h.Svc.RunInstances(input)
		if err != nil {
			return nil, err
		} else {
			fmt.Println("Launch Instance Success!")
			for _, instance := range resp.Instances {
				fmt.Println("Instance ID:", *instance.InstanceId)
				launchedInstances = append(launchedInstances, *instance.InstanceId)
			}
			return launchedInstances, nil
		}
	} else {
		// Abort
		return nil, errors.New("Options not confirmed")
	}
}

func (h *EC2Helper) LaunchSpotInstance(simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo, confirmation bool) error {
	var err error
	if confirmation {
		fmt.Println("Options confirmed! Launching spot instance...")
		if simpleConfig.LaunchTemplateId != "" {
			_, err = h.LaunchFleet(aws.String(simpleConfig.LaunchTemplateId))
		} else {
			// Create new stack, if specified.
			if simpleConfig.NewVPC {
				err := h.createNetworkConfiguration(simpleConfig, nil)
				if err != nil {
					return err
				}
			}

			template, err := h.CreateLaunchTemplate(simpleConfig, detailedConfig)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					fmt.Println(aerr.Error())
				} else {
					fmt.Println(err.Error())
				}
				return err
			}
			_, err = h.LaunchFleet(template.LaunchTemplateId)
			err = h.DeleteLaunchTemplate(template.LaunchTemplateId)
		}
	} else {
		// Abort
		return errors.New("Options not confirmed")
	}

	return err
}

// Create a new stack and update simpleConfig for config saving
func (h *EC2Helper) createNetworkConfiguration(simpleConfig *config.SimpleInfo,
	input *ec2.RunInstancesInput) error {
	// Get all available azs for later use
	availabilityZones, err := h.GetAvailableAvailabilityZones()
	if err != nil {
		return err
	}

	// Retrieve resources from the stack
	c := cfn.New(h.Sess)
	vpcId, subnetIds, _, _, err := c.CreateStackAndGetResources(availabilityZones, nil,
		cfn.SimpleEc2CloudformationTemplate)
	if err != nil {
		return err
	}

	// Find the subnetId with the correct availability zone
	var selectedSubnetId *string
	for _, subnetId := range subnetIds {
		subnet, err := h.GetSubnetById(subnetId)
		if err != nil {
			return err
		}

		if *subnet.AvailabilityZone == simpleConfig.SubnetId {
			selectedSubnetId = subnet.SubnetId
			break
		}
	}
	if selectedSubnetId == nil {
		return errors.New("No subnet with the selected availability zone found")
	}

	if input != nil {
		input.SubnetId = selectedSubnetId
	}

	/*
		Get the security group.
		If the users choose "all", simply add all existing security groups to the selection.
		If the users choose "new", create a new security group for SSH and add it to the selection.
		Otherwise, report an error.
	*/
	securityGroupPlaceholder := simpleConfig.SecurityGroupIds[0]
	selectedSecurityGroupIds := []string{}

	if securityGroupPlaceholder == cli.ResponseAll {
		securityGroups, err := h.GetSecurityGroupsByVpc(*vpcId)
		if err != nil {
			return err
		}

		if securityGroups != nil {
			for _, group := range securityGroups {
				selectedSecurityGroupIds = append(selectedSecurityGroupIds, *group.GroupId)
			}
		}
	} else if securityGroupPlaceholder == cli.ResponseNew {
		groupId, err := h.CreateSecurityGroupForSsh(*vpcId)
		if err != nil {
			return err
		}

		selectedSecurityGroupIds = append(selectedSecurityGroupIds, *groupId)
	} else {
		return errors.New("Unknown security group placeholder")
	}

	if len(selectedSecurityGroupIds) <= 0 {
		return errors.New("No security group available for stack")
	}

	if input != nil {
		input.SecurityGroupIds = aws.StringSlice(selectedSecurityGroupIds)
	}

	// Update simpleConfig for config saving
	simpleConfig.NewVPC = false
	simpleConfig.SubnetId = *selectedSubnetId
	simpleConfig.SecurityGroupIds = selectedSecurityGroupIds

	return nil
}

// Terminate the instances based on ids
func (h *EC2Helper) TerminateInstances(instanceIds []string) error {
	// Get instance id
	input := &ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	}

	fmt.Println("Terminating instances")

	_, err := h.Svc.TerminateInstances(input)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("Instances %s terminated successfully", instanceIds))

	return nil
}

// Get the name tag of the resource
func GetTagName(tags []*ec2.Tag) *string {
	for _, tag := range tags {
		if *tag.Key == tagNameKey {
			return tag.Value
		}
	}
	return nil
}

// Get the tags for resources created by simple-ec2
func getSimpleEc2Tags() []*ec2.Tag {
	simpleEc2Tags := []*ec2.Tag{}

	tags := tag.GetSimpleEc2Tags()
	for key, value := range *tags {
		simpleEc2Tags = append(simpleEc2Tags, &ec2.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return simpleEc2Tags
}

// Validate an image id. Used as a function interface to validate question input
func ValidateImageId(h *EC2Helper, imageId string) bool {
	image, _ := h.GetImageById(imageId)
	return image != nil
}

// Validate a filepath. Used as a function interface to validate question input
func ValidateFilepath(h *EC2Helper, userFilePath string) bool {
	_, err := os.Stat(userFilePath)
	return err == nil
}

// Validate user's tag input. Used as a function interface to validate question input
func ValidateTags(h *EC2Helper, userTags string) bool {
	//tag1|val1,tag2|val2
	for _, rawTag := range strings.Split(userTags, ",") { //[tag1|val1, tag2|val2]
		if len(strings.Split(rawTag, "|")) != 2 { //[tag1,val1]
			return false
		}
	}
	return true
}

// ValidateInteger checks if a given string is an integer
func ValidateInteger(h *EC2Helper, intString string) bool {
	_, err := strconv.Atoi(intString)
	if err != nil {
		return false
	}
	return true
}

// Given an AWS platform string, tell if it's a Linux platform
func IsLinux(platform string) bool {
	return platform == ec2.CapacityReservationInstancePlatformLinuxUnix ||
		platform == ec2.CapacityReservationInstancePlatformRedHatEnterpriseLinux ||
		platform == ec2.CapacityReservationInstancePlatformSuselinux ||
		platform == ec2.CapacityReservationInstancePlatformLinuxwithSqlserverStandard ||
		platform == ec2.CapacityReservationInstancePlatformLinuxwithSqlserverWeb ||
		platform == ec2.CapacityReservationInstancePlatformLinuxwithSqlserverEnterprise
}

// Determine if an image contains at least one EBS volume
func HasEbsVolume(image *ec2.Image) bool {
	if image.BlockDeviceMappings != nil {
		for _, block := range image.BlockDeviceMappings {
			if block.Ebs != nil {
				return true
			}
		}
	}

	return false
}

func (h *EC2Helper) CreateLaunchTemplate(simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo) (*ec2.LaunchTemplate, error) {
	launchIdentifier := uuid.New()

	fmt.Println("Creating Launch Template...")

	dataConfig := createRequestInstanceConfig(simpleConfig, detailedConfig)
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				{
					AssociatePublicIpAddress: aws.Bool(true),
					DeviceIndex:              aws.Int64(0),
					Groups:                   dataConfig.SecurityGroupIds,
					SubnetId:                 dataConfig.SubnetId,
				},
			},
			IamInstanceProfile:                (*ec2.LaunchTemplateIamInstanceProfileSpecificationRequest)(dataConfig.IamInstanceProfile),
			ImageId:                           dataConfig.ImageId,
			InstanceType:                      dataConfig.InstanceType,
			BlockDeviceMappings:               dataConfig.LaunchTemplateBlockMappings,
			InstanceInitiatedShutdownBehavior: dataConfig.InstanceInitiatedShutdownBehavior,
			UserData:                          dataConfig.UserData,
		},
		LaunchTemplateName: aws.String(fmt.Sprintf("SimpleEC2LaunchTemplate-%s", launchIdentifier)),
		VersionDescription: aws.String(fmt.Sprintf("Launch Template %s", launchIdentifier)),
	}

	result, err := h.Svc.CreateLaunchTemplate(input)
	return result.LaunchTemplate, err
}

func createRequestInstanceConfig(simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo) config.RequestInstanceInfo {
	requestInstanceConfig := config.RequestInstanceInfo{}

	if simpleConfig.LaunchTemplateId != "" {
		requestInstanceConfig.LaunchTemplate = &ec2.LaunchTemplateSpecification{
			LaunchTemplateId: aws.String(simpleConfig.LaunchTemplateId),
			Version:          aws.String(simpleConfig.LaunchTemplateVersion),
		}
	}

	if simpleConfig.ImageId != "" {
		requestInstanceConfig.ImageId = aws.String(simpleConfig.ImageId)
	}
	if simpleConfig.InstanceType != "" {
		requestInstanceConfig.InstanceType = aws.String(simpleConfig.InstanceType)
	}
	if simpleConfig.SubnetId != "" {
		requestInstanceConfig.SubnetId = aws.String(simpleConfig.SubnetId)
	}
	if simpleConfig.SecurityGroupIds != nil && len(simpleConfig.SecurityGroupIds) > 0 {
		requestInstanceConfig.SecurityGroupIds = aws.StringSlice(simpleConfig.SecurityGroupIds)
	}
	if simpleConfig.IamInstanceProfile != "" {
		requestInstanceConfig.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Name: aws.String(simpleConfig.IamInstanceProfile),
		}
	}

	setAutoTermination := false
	if detailedConfig != nil {
		// Set all EBS volumes not to be deleted, if specified
		if HasEbsVolume(detailedConfig.Image) && simpleConfig.KeepEbsVolumeAfterTermination {
			requestInstanceConfig.BlockDeviceMappings = detailedConfig.Image.BlockDeviceMappings
			for _, block := range requestInstanceConfig.BlockDeviceMappings {
				if block.Ebs != nil {
					block.Ebs.DeleteOnTermination = aws.Bool(false)
				}
			}
			blockDevices := []*ec2.LaunchTemplateBlockDeviceMappingRequest{}
			for index, block := range detailedConfig.Image.BlockDeviceMappings {
				blockDevices = append(blockDevices, &ec2.LaunchTemplateBlockDeviceMappingRequest{
					DeviceName:  block.DeviceName,
					NoDevice:    block.NoDevice,
					VirtualName: block.VirtualName,
				})
				if block.Ebs != nil {
					blockDeviceEbs := &ec2.LaunchTemplateEbsBlockDeviceRequest{
						DeleteOnTermination: aws.Bool(false),
						Encrypted:           block.Ebs.Encrypted,
						Iops:                block.Ebs.Iops,
						KmsKeyId:            block.Ebs.KmsKeyId,
						SnapshotId:          block.Ebs.SnapshotId,
						Throughput:          block.Ebs.Throughput,
						VolumeSize:          block.Ebs.VolumeSize,
						VolumeType:          block.Ebs.VolumeType,
					}
					blockDevices[index].SetEbs(blockDeviceEbs)
				}
			}
			requestInstanceConfig.LaunchTemplateBlockMappings = blockDevices
		}
		setAutoTermination = IsLinux(*detailedConfig.Image.PlatformDetails) && simpleConfig.AutoTerminationTimerMinutes > 0
	}

	if setAutoTermination {
		requestInstanceConfig.InstanceInitiatedShutdownBehavior = aws.String("terminate")
		autoTermCmd := fmt.Sprintf("#!/bin/bash\necho \"sudo poweroff\" | at now + %d minutes\n",
			simpleConfig.AutoTerminationTimerMinutes)
		if simpleConfig.BootScriptFilePath == "" {
			requestInstanceConfig.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(autoTermCmd)))
		} else {
			bootScriptRaw, _ := ioutil.ReadFile(simpleConfig.BootScriptFilePath)
			bootScriptLines := strings.Split(string(bootScriptRaw), "\n")
			//if #!/bin/bash is first, then replace first line otherwise, prepend termination
			if len(bootScriptLines) >= 1 && bootScriptLines[0] == "#!/bin/bash" {
				bootScriptLines[0] = autoTermCmd
			} else {
				bootScriptLines = append([]string{autoTermCmd}, bootScriptLines...)
			}
			bootScriptRaw = []byte(strings.Join(bootScriptLines, "\n"))
			requestInstanceConfig.UserData = aws.String(base64.StdEncoding.EncodeToString(bootScriptRaw))
		}
	} else {
		if simpleConfig.BootScriptFilePath != "" {
			bootScriptRaw, _ := ioutil.ReadFile(simpleConfig.BootScriptFilePath)
			requestInstanceConfig.UserData = aws.String(base64.StdEncoding.EncodeToString(bootScriptRaw))
		}
	}

	return requestInstanceConfig
}

func (h *EC2Helper) DeleteLaunchTemplate(templateId *string) error {
	fmt.Println("Deleting Launch Template...")
	input := &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: templateId,
	}

	_, err := h.Svc.DeleteLaunchTemplate(input)
	return err
}

func (h *EC2Helper) LaunchFleet(templateId *string) (*ec2.CreateFleetOutput, error) {
	fleetTemplateSpecs := &ec2.FleetLaunchTemplateSpecificationRequest{
		LaunchTemplateId: templateId,
		Version:          aws.String("$Latest"),
	}

	fleetTemplateConfig := []*ec2.FleetLaunchTemplateConfigRequest{
		{
			LaunchTemplateSpecification: fleetTemplateSpecs,
		},
	}

	spotRequest := &ec2.SpotOptionsRequest{
		AllocationStrategy: aws.String("capacity-optimized"),
	}

	targetCapacity := &ec2.TargetCapacitySpecificationRequest{
		DefaultTargetCapacityType: aws.String("spot"),
		OnDemandTargetCapacity:    aws.Int64(0),
		SpotTargetCapacity:        aws.Int64(1),
		TotalTargetCapacity:       aws.Int64(1),
	}

	input := &ec2.CreateFleetInput{
		LaunchTemplateConfigs:       fleetTemplateConfig,
		SpotOptions:                 spotRequest,
		TargetCapacitySpecification: targetCapacity,
		Type:                        aws.String("instant"),
	}

	result, err := h.Svc.CreateFleet(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	} else {
		if len(result.Errors) != 0 {
			err = errors.New(*result.Errors[0].ErrorMessage)
			cli.ShowError(err, "Creating spot instance failed")
			return nil, err
		}
	}

	fmt.Println("Launch Spot Instance Success!")
	for _, instance := range result.Instances {
		for _, id := range instance.InstanceIds {
			fmt.Printf("Spot Instance ID: %s\n", *id)
		}
	}

	return result, err
}
