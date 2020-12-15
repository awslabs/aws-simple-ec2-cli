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

package testhelper

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type MockedEC2Svc struct {
	DescribeRegionsError                     error
	DescribeAvailabilityZonesError           error
	DescribeLaunchTemplatesPagesError        error
	DescribeLaunchTemplateVersionsPagesError error
	DescribeInstanceTypesPagesError          error
	DescribeImagesError                      error
	DescribeVpcsPagesError                   error
	DescribeSubnetsPagesError                error
	DescribeSecurityGroupsPagesError         error
	CreateSecurityGroupError                 error
	AuthorizeSecurityGroupIngressError       error
	DescribeInstancesPagesError              error
	CreateTagsError                          error
	RunInstancesError                        error
	TerminateInstancesError                  error
	Regions                                  []*ec2.Region
	AvailabilityZones                        []*ec2.AvailabilityZone
	LaunchTemplates                          []*ec2.LaunchTemplate
	LaunchTemplateVersions                   []*ec2.LaunchTemplateVersion
	InstanceTypes                            []*ec2.InstanceTypeInfo
	Images                                   []*ec2.Image
	Vpcs                                     []*ec2.Vpc
	Subnets                                  []*ec2.Subnet
	SecurityGroups                           []*ec2.SecurityGroup
	Instances                                []*ec2.Instance
}

func (e *MockedEC2Svc) New() {
	e.Subnets = []*ec2.Subnet{
		{
			SubnetId: aws.String("subnet-12345"),
			VpcId:    aws.String("vpc-12345"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("Subnet 1"),
				},
			},
		},
		{
			SubnetId: aws.String("subnet-67890"),
			VpcId:    aws.String("vpc-67890"),
		},
	}
	e.Regions = []*ec2.Region{
		{
			RegionName: aws.String("region-a"),
		},
		{
			RegionName: aws.String("region-b"),
		},
	}
	e.AvailabilityZones = []*ec2.AvailabilityZone{
		{},
		{},
	}
}

func (e *MockedEC2Svc) DescribeRegions(input *ec2.DescribeRegionsInput) (*ec2.DescribeRegionsOutput, error) {
	output := &ec2.DescribeRegionsOutput{
		Regions: e.Regions,
	}

	return output, e.DescribeRegionsError
}

func (e *MockedEC2Svc) DescribeAvailabilityZones(input *ec2.DescribeAvailabilityZonesInput) (*ec2.DescribeAvailabilityZonesOutput, error) {
	output := &ec2.DescribeAvailabilityZonesOutput{
		AvailabilityZones: e.AvailabilityZones,
	}

	return output, e.DescribeAvailabilityZonesError
}

func (e *MockedEC2Svc) DescribeLaunchTemplatesPages(input *ec2.DescribeLaunchTemplatesInput, fn func(*ec2.DescribeLaunchTemplatesOutput, bool) bool) error {
	templates := []*ec2.LaunchTemplate{}

	if input.LaunchTemplateIds != nil {
		// Find all templates
		for _, templateId := range input.LaunchTemplateIds {
			for _, template := range e.LaunchTemplates {
				if *template.LaunchTemplateId == *templateId {
					templates = append(templates, template)
				}
			}
		}
	} else {
		templates = e.LaunchTemplates
	}

	output := &ec2.DescribeLaunchTemplatesOutput{
		LaunchTemplates: templates,
	}

	for {
		if !fn(output, true) {
			return e.DescribeLaunchTemplatesPagesError
		}
	}
}

func (e *MockedEC2Svc) DescribeLaunchTemplateVersionsPages(input *ec2.DescribeLaunchTemplateVersionsInput, fn func(*ec2.DescribeLaunchTemplateVersionsOutput, bool) bool) error {
	templateId := *input.LaunchTemplateId
	versions := []*ec2.LaunchTemplateVersion{}

	if input.Versions != nil {
		// Find all templates with specified version ID and template ID
		for _, versionId := range input.Versions {
			for _, version := range e.LaunchTemplateVersions {
				if strconv.FormatInt(*version.VersionNumber, 10) == *versionId && *version.LaunchTemplateId == templateId {
					versions = append(versions, version)
				}
			}
		}
	} else {
		// Find all templates with specified template ID
		for _, version := range e.LaunchTemplateVersions {
			if *version.LaunchTemplateId == templateId {
				versions = append(versions, version)
			}
		}
	}

	output := &ec2.DescribeLaunchTemplateVersionsOutput{
		LaunchTemplateVersions: versions,
	}

	for {
		if !fn(output, true) {
			return e.DescribeLaunchTemplateVersionsPagesError
		}
	}
}

func (e *MockedEC2Svc) DescribeInstanceTypesPages(input *ec2.DescribeInstanceTypesInput, fn func(*ec2.DescribeInstanceTypesOutput, bool) bool) error {
	instanceTypeInfos := []*ec2.InstanceTypeInfo{}
	isFree := false

	// Find if looking for free instance types
	values := findFilter(input.Filters, "free-tier-eligible")
	if values != nil && len(values) > 0 && *values[0] == "true" {
		isFree = true
	}

	if input.InstanceTypes != nil {
		// Find all instance types
		for _, instanceType := range input.InstanceTypes {
			for _, instanceTypeInfo := range e.InstanceTypes {
				if *instanceTypeInfo.InstanceType == *instanceType {
					instanceTypeInfos = append(instanceTypeInfos, instanceTypeInfo)
				}
			}
		}
	} else {
		instanceTypeInfos = e.InstanceTypes
	}

	// Extract free instance types, if required
	if isFree {
		freeInstanceTypeInfo := []*ec2.InstanceTypeInfo{}

		for _, instanceTypeInfo := range instanceTypeInfos {
			if *instanceTypeInfo.FreeTierEligible {
				freeInstanceTypeInfo = append(freeInstanceTypeInfo, instanceTypeInfo)
			}
		}

		instanceTypeInfos = freeInstanceTypeInfo
	}

	output := &ec2.DescribeInstanceTypesOutput{
		InstanceTypes: instanceTypeInfos,
	}

	for {
		if !fn(output, true) {
			return e.DescribeInstanceTypesPagesError
		}
	}
}

func (e *MockedEC2Svc) DescribeImages(input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
	output := &ec2.DescribeImagesOutput{
		Images: e.Images,
	}

	return output, e.DescribeImagesError
}

func (e *MockedEC2Svc) DescribeVpcsPages(input *ec2.DescribeVpcsInput, fn func(*ec2.DescribeVpcsOutput, bool) bool) error {
	vpcs := []*ec2.Vpc{}

	if input.VpcIds != nil {
		// Find all VPCs
		for _, vpcId := range input.VpcIds {
			for _, vpc := range e.Vpcs {
				if *vpc.VpcId == *vpcId {
					vpcs = append(vpcs, vpc)
				}
			}
		}
	} else {
		vpcs = e.Vpcs
	}

	output := &ec2.DescribeVpcsOutput{
		Vpcs: vpcs,
	}

	for {
		if !fn(output, true) {
			return e.DescribeVpcsPagesError
		}
	}
}

func (e *MockedEC2Svc) DescribeSubnetsPages(input *ec2.DescribeSubnetsInput, fn func(*ec2.DescribeSubnetsOutput, bool) bool) error {
	subnets := []*ec2.Subnet{}

	// Find all subnet IDs in input
	if input.Filters != nil {
		subnetIdValues := findFilter(input.Filters, "subnet-id")
		vpcIdValues := findFilter(input.Filters, "vpc-id")

		// Find all subnets
		if subnetIdValues != nil {
			for _, subnetId := range subnetIdValues {
				for _, subnet := range e.Subnets {
					if *subnet.SubnetId == *subnetId {
						subnets = append(subnets, subnet)
					}
				}
			}
		} else if vpcIdValues != nil {
			for _, vpcId := range vpcIdValues {
				for _, subnet := range e.Subnets {
					if *subnet.VpcId == *vpcId {
						subnets = append(subnets, subnet)
					}
				}
			}
		}
	} else {
		subnets = e.Subnets
	}

	output := &ec2.DescribeSubnetsOutput{
		Subnets: subnets,
	}

	for {
		if !fn(output, true) {
			return e.DescribeSubnetsPagesError
		}
	}
}

func (e *MockedEC2Svc) DescribeSecurityGroupsPages(input *ec2.DescribeSecurityGroupsInput, fn func(*ec2.DescribeSecurityGroupsOutput, bool) bool) error {
	output := &ec2.DescribeSecurityGroupsOutput{
		SecurityGroups: e.SecurityGroups,
	}

	for {
		if !fn(output, true) {
			return e.DescribeSecurityGroupsPagesError
		}
	}
}

func (e *MockedEC2Svc) CreateSecurityGroup(input *ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error) {
	output := &ec2.CreateSecurityGroupOutput{
		GroupId: aws.String("sg-12345"),
	}

	return output, e.CreateSecurityGroupError
}

func (e *MockedEC2Svc) AuthorizeSecurityGroupIngress(input *ec2.AuthorizeSecurityGroupIngressInput) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	return nil, e.AuthorizeSecurityGroupIngressError
}

func (e *MockedEC2Svc) DescribeInstancesPages(input *ec2.DescribeInstancesInput, fn func(*ec2.DescribeInstancesOutput, bool) bool) error {
	output := &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{
				Instances: e.Instances,
			},
		},
	}

	for {
		if !fn(output, true) {
			return e.DescribeInstancesPagesError
		}
	}
}

func (e *MockedEC2Svc) CreateTags(input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return nil, e.CreateTagsError
}

func (e *MockedEC2Svc) RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error) {
	output := &ec2.Reservation{
		Instances: []*ec2.Instance{
			{
				InstanceId: aws.String("i-12345"),
			},
		},
	}

	return output, e.RunInstancesError
}

func (e *MockedEC2Svc) TerminateInstances(input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return nil, e.TerminateInstancesError
}

func findFilter(filters []*ec2.Filter, name string) []*string {
	if filters != nil {
		for _, filter := range filters {
			if *filter.Name == name {
				return filter.Values
			}
		}
	}

	return nil
}

// Placeholder functions
func (e *MockedEC2Svc) DeleteSecurityGroup(input *ec2.DeleteSecurityGroupInput) (*ec2.DeleteSecurityGroupOutput, error) {
	return nil, nil
}
