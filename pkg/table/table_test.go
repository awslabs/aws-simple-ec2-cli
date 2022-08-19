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

package table_test

import (
	"errors"
	"testing"

	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/table"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var testEC2 = &ec2helper.EC2Helper{}

func TestBuildTable(t *testing.T) {
	expectedTable := `+---------+------------+------------+
| ROW NUM | ELEMENTS 1 | ELEMENTS 2 |
+---------+------------+------------+
| Row 1   | Element 1  | Element 2  |
| Row 2   | Element 3  | Element 4  |
| Row 3   | Element 5  | Element 6  |
+---------+------------+------------+
`

	data := [][]string{
		{"Row 1", "Element 1", "Element 2"},
		{"Row 2", "Element 3", "Element 4"},
		{"Row 3", "Element 5", "Element 6"},
	}

	header := []string{"Row Num", "Elements 1", "Elements 2"}

	builtTable := table.BuildTable(data, header)
	th.Equals(t, expectedTable, builtTable)
}

func TestAppendTemplateEbs(t *testing.T) {
	expectedData := [][]string{
		{"EBS Volumes", "dev1(gp2): 16 GiB"},
		{"", "dev2(gp2)"},
		{"", "dev3: 32 GiB"},
		{"", "dev4"},
	}

	data := [][]string{}

	mappings := []*ec2.LaunchTemplateBlockDeviceMapping{
		{
			DeviceName: aws.String("dev1"),
			Ebs: &ec2.LaunchTemplateEbsBlockDevice{
				VolumeType: aws.String("gp2"),
				VolumeSize: aws.Int64(16),
			},
		},
		{
			DeviceName: aws.String("dev2"),
			Ebs: &ec2.LaunchTemplateEbsBlockDevice{
				VolumeType: aws.String("gp2"),
			},
		},
		{
			DeviceName: aws.String("dev3"),
			Ebs: &ec2.LaunchTemplateEbsBlockDevice{
				VolumeSize: aws.Int64(32),
			},
		},
		{
			DeviceName: aws.String("dev4"),
		},
	}

	data = table.AppendTemplateEbs(data, mappings)
	th.Equals(t, expectedData, data)
}

func TestAppendEbs(t *testing.T) {
	expectedData := [][]string{
		{"EBS Volumes", "dev1(gp2): 16 GiB"},
		{"", "dev2(gp2)"},
		{"", "dev3: 32 GiB"},
		{"", "dev4"},
	}

	data := [][]string{}

	mappings := []*ec2.BlockDeviceMapping{
		{
			DeviceName: aws.String("dev1"),
			Ebs: &ec2.EbsBlockDevice{
				VolumeType: aws.String("gp2"),
				VolumeSize: aws.Int64(16),
			},
		},
		{
			DeviceName: aws.String("dev2"),
			Ebs: &ec2.EbsBlockDevice{
				VolumeType: aws.String("gp2"),
			},
		},
		{
			DeviceName: aws.String("dev3"),
			Ebs: &ec2.EbsBlockDevice{
				VolumeSize: aws.Int64(32),
			},
		},
		{
			DeviceName: aws.String("dev4"),
		},
	}

	data, _ = table.AppendEbs(data, mappings)
	th.Equals(t, expectedData, data)
}

func TestAppendSecurityGroups(t *testing.T) {
	expectedData := [][]string{
		{"Security Group", "Security Group 1(sg-12345)"},
		{"", "sg-67890"},
	}

	data := [][]string{}

	securityGroups := []*ec2.SecurityGroup{
		{
			GroupId: aws.String("sg-12345"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("Security Group 1"),
				},
			},
		},
		{
			GroupId: aws.String("sg-67890"),
		},
	}

	data, _ = table.AppendSecurityGroups(data, securityGroups)
	th.Equals(t, expectedData, data)
}

var mockedNetworkInterfaces = []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecification{
	{
		SubnetId: aws.String("subnet-12345"),
	},
	{
		SubnetId: aws.String("subnet-67890"),
	},
}

func TestAppendTemplateNetworkInterfaces_Success(t *testing.T) {
	expectedData := [][]string{
		{"Subnets", "1.Subnet 1(vpc-12345:subnet-12345)"},
		{"", "2.vpc-67890:subnet-67890"},
	}

	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{
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
		},
	}

	data, err := table.AppendTemplateNetworkInterfaces(testEC2, [][]string{}, mockedNetworkInterfaces)
	th.Ok(t, err)
	th.Equals(t, expectedData, data)
}

func TestAppendTemplateNetworkInterfaces_NoNetworkInterface(t *testing.T) {
	expectedData := [][]string{
		{"Subnets", "not specified"},
	}

	data, err := table.AppendTemplateNetworkInterfaces(testEC2, [][]string{}, nil)
	th.Ok(t, err)
	th.Equals(t, expectedData, data)
}

func TestAppendTemplateNetworkInterfaces_ApiError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}

	_, err := table.AppendTemplateNetworkInterfaces(testEC2, [][]string{}, mockedNetworkInterfaces)
	th.Nok(t, err)
}

func TestAppendInstances(t *testing.T) {
	expectedData := [][]string{
		{"Instance 2(i-67890)", "", ""},
		{"Instance 3(i-54321)", "CreatedBy", "simple-ec2"},
		{"", "CreatedTime", "just now"},
		{"i-09876", "", ""},
	}
	expectedOptions := []string{
		"i-67890",
		"i-54321",
		"i-09876",
	}

	data := [][]string{}
	indexedOptions := []string{}
	addedInstanceIds := []string{}

	instances := []*ec2.Instance{
		{
			InstanceId: aws.String("i-12345"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("Instance 1"),
				},
			},
		},
		{
			InstanceId: aws.String("i-67890"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("Instance 2"),
				},
			},
		},
		{
			InstanceId: aws.String("i-54321"),
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("Instance 3"),
				},
				{
					Key:   aws.String("CreatedBy"),
					Value: aws.String("simple-ec2"),
				},
				{
					Key:   aws.String("CreatedTime"),
					Value: aws.String("just now"),
				},
			},
		},
		{
			InstanceId: aws.String("i-09876"),
		},
	}

	addedInstanceIds = append(addedInstanceIds, *instances[0].InstanceId)
	data, indexedOptions, _, _ = table.AppendInstances(data, indexedOptions, instances, addedInstanceIds)
	th.Equals(t, expectedData, data)
	th.Equals(t, expectedOptions, indexedOptions)
}
