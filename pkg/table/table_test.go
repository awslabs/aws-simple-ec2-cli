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
	correctTable := `+---------+------------+------------+
| ROW NUM | ELEMENTS 1 | ELEMENTS 2 |
+---------+------------+------------+
| Row 1   | Element 1  | Element 2  |
| Row 2   | Element 3  | Element 4  |
| Row 3   | Element 5  | Element 6  |
+---------+------------+------------+
`

	data := [][]string{
		[]string{"Row 1", "Element 1", "Element 2"},
		[]string{"Row 2", "Element 3", "Element 4"},
		[]string{"Row 3", "Element 5", "Element 6"},
	}

	header := []string{"Row Num", "Elements 1", "Elements 2"}

	builtTable := table.BuildTable(data, header)

	if builtTable != correctTable {
		t.Errorf("Table built is not correct. \nexpect:\n%s\ngot:\n%s",
			correctTable, builtTable)
	}
}

func TestAppendTemplateEbs(t *testing.T) {
	correctData := [][]string{
		[]string{"EBS Volumes", "dev1(gp2): 16 GiB"},
		[]string{"", "dev2(gp2)"},
		[]string{"", "dev3: 32 GiB"},
		[]string{"", "dev4"},
	}

	data := [][]string{}

	mappings := []*ec2.LaunchTemplateBlockDeviceMapping{
		&ec2.LaunchTemplateBlockDeviceMapping{
			DeviceName: aws.String("dev1"),
			Ebs: &ec2.LaunchTemplateEbsBlockDevice{
				VolumeType: aws.String("gp2"),
				VolumeSize: aws.Int64(16),
			},
		},
		&ec2.LaunchTemplateBlockDeviceMapping{
			DeviceName: aws.String("dev2"),
			Ebs: &ec2.LaunchTemplateEbsBlockDevice{
				VolumeType: aws.String("gp2"),
			},
		},
		&ec2.LaunchTemplateBlockDeviceMapping{
			DeviceName: aws.String("dev3"),
			Ebs: &ec2.LaunchTemplateEbsBlockDevice{
				VolumeSize: aws.Int64(32),
			},
		},
		&ec2.LaunchTemplateBlockDeviceMapping{
			DeviceName: aws.String("dev4"),
		},
	}

	data = table.AppendTemplateEbs(data, mappings)

	for i := 0; i < len(correctData); i++ {
		if !th.StringSliceEqual(correctData[i], data[i]) {
			t.Errorf("Appended template ebs data incorrect.\nexpect:%s\ngot:%s",
				correctData, data)
			break
		}
	}
}

func TestAppendEbs(t *testing.T) {
	correctData := [][]string{
		[]string{"EBS Volumes", "dev1(gp2): 16 GiB"},
		[]string{"", "dev2(gp2)"},
		[]string{"", "dev3: 32 GiB"},
		[]string{"", "dev4"},
	}

	data := [][]string{}

	mappings := []*ec2.BlockDeviceMapping{
		&ec2.BlockDeviceMapping{
			DeviceName: aws.String("dev1"),
			Ebs: &ec2.EbsBlockDevice{
				VolumeType: aws.String("gp2"),
				VolumeSize: aws.Int64(16),
			},
		},
		&ec2.BlockDeviceMapping{
			DeviceName: aws.String("dev2"),
			Ebs: &ec2.EbsBlockDevice{
				VolumeType: aws.String("gp2"),
			},
		},
		&ec2.BlockDeviceMapping{
			DeviceName: aws.String("dev3"),
			Ebs: &ec2.EbsBlockDevice{
				VolumeSize: aws.Int64(32),
			},
		},
		&ec2.BlockDeviceMapping{
			DeviceName: aws.String("dev4"),
		},
	}

	data = table.AppendEbs(data, mappings)

	for i := 0; i < len(correctData); i++ {
		if !th.StringSliceEqual(correctData[i], data[i]) {
			t.Errorf("Appended Ebs data incorrect.\nexpect:%s\ngot:%s",
				correctData, data)
			break
		}
	}
}

func TestAppendSecurityGroups(t *testing.T) {
	correctData := [][]string{
		[]string{"Security Group", "Security Group 1(sg-12345)"},
		[]string{"", "sg-67890"},
	}

	data := [][]string{}

	securityGroups := []*ec2.SecurityGroup{
		&ec2.SecurityGroup{
			GroupId: aws.String("sg-12345"),
			Tags: []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Name"),
					Value: aws.String("Security Group 1"),
				},
			},
		},
		&ec2.SecurityGroup{
			GroupId: aws.String("sg-67890"),
		},
	}

	data = table.AppendSecurityGroups(data, securityGroups)

	for i := 0; i < len(correctData); i++ {
		if !th.StringSliceEqual(correctData[i], data[i]) {
			t.Errorf("Appended security group data incorrect.\nexpect:%s\ngot:%s",
				correctData, data)
			break
		}
	}
}

var mockedNetworkInterfaces = []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecification{
	&ec2.LaunchTemplateInstanceNetworkInterfaceSpecification{
		SubnetId: aws.String("subnet-12345"),
	},
	&ec2.LaunchTemplateInstanceNetworkInterfaceSpecification{
		SubnetId: aws.String("subnet-67890"),
	},
}

func TestAppendTemplateNetworkInterfaces_Success(t *testing.T) {
	correctData := [][]string{
		[]string{"Subnets", "1.Subnet 1(vpc-12345:subnet-12345)"},
		[]string{"", "2.vpc-67890:subnet-67890"},
	}

	data := [][]string{}

	testEC2.Svc = &th.MockedEC2Svc{
		Subnets: []*ec2.Subnet{
			&ec2.Subnet{
				SubnetId: aws.String("subnet-12345"),
				VpcId:    aws.String("vpc-12345"),
				Tags: []*ec2.Tag{
					&ec2.Tag{
						Key:   aws.String("Name"),
						Value: aws.String("Subnet 1"),
					},
				},
			},
			&ec2.Subnet{
				SubnetId: aws.String("subnet-67890"),
				VpcId:    aws.String("vpc-67890"),
			},
		},
	}

	var err error
	data, err = table.AppendTemplateNetworkInterfaces(testEC2, data, mockedNetworkInterfaces)
	if err != nil {
		t.Error(err)
	} else {
		for i := 0; i < len(correctData); i++ {
			if !th.StringSliceEqual(correctData[i], data[i]) {
				t.Errorf("Appended network interfaces incorrect when network interfaces are specified.\nexpect:%s\ngot:%s",
					correctData, data)
				break
			}
		}
	}
}

func TestAppendTemplateNetworkInterfaces_NoNetworkInterface(t *testing.T) {
	correctData := [][]string{
		[]string{"Subnets", "not specified"},
	}
	data := [][]string{}

	var err error
	data, err = table.AppendTemplateNetworkInterfaces(testEC2, data, nil)
	if err != nil {
		t.Error(err)
	} else {
		for i := 0; i < len(correctData); i++ {
			if !th.StringSliceEqual(correctData[i], data[i]) {
				t.Errorf("Appended network interfaces incorrect when network interfaces are not specified.\nexpect:%s\ngot:%s",
					correctData, data)
				break
			}
		}
	}
}

func TestAppendTemplateNetworkInterfaces_ApiError(t *testing.T) {
	testEC2.Svc = &th.MockedEC2Svc{
		DescribeSubnetsPagesError: errors.New("Test error"),
	}
	data := [][]string{}

	var err error
	data, err = table.AppendTemplateNetworkInterfaces(testEC2, data, mockedNetworkInterfaces)
	if err == nil {
		t.Error(th.ExpectErrorMsg)
	}
}

func TestAppendInstances(t *testing.T) {
	correctData := [][]string{
		[]string{"1.", "Instance 2(i-67890)", "", ""},
		[]string{"2.", "Instance 3(i-54321)", "CreatedBy", "simple-ec2"},
		[]string{"", "", "CreatedTime", "just now"},
		[]string{"3.", "i-09876", "", ""},
	}
	correctOptions := []string{
		"i-67890",
		"i-54321",
		"i-09876",
	}

	data := [][]string{}
	indexedOptions := []string{}
	addedInstanceIds := []string{}

	instances := []*ec2.Instance{
		&ec2.Instance{
			InstanceId: aws.String("i-12345"),
			Tags: []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Name"),
					Value: aws.String("Instance 1"),
				},
			},
		},
		&ec2.Instance{
			InstanceId: aws.String("i-67890"),
			Tags: []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Name"),
					Value: aws.String("Instance 2"),
				},
			},
		},
		&ec2.Instance{
			InstanceId: aws.String("i-54321"),
			Tags: []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Name"),
					Value: aws.String("Instance 3"),
				},
				&ec2.Tag{
					Key:   aws.String("CreatedBy"),
					Value: aws.String("simple-ec2"),
				},
				&ec2.Tag{
					Key:   aws.String("CreatedTime"),
					Value: aws.String("just now"),
				},
			},
		},
		&ec2.Instance{
			InstanceId: aws.String("i-09876"),
		},
	}

	addedInstanceIds = append(addedInstanceIds, *instances[0].InstanceId)

	data, indexedOptions, _ = table.AppendInstances(data, indexedOptions, instances, addedInstanceIds)

	// Check appended data
	for i := 0; i < len(correctData); i++ {
		if !th.StringSliceEqual(correctData[i], data[i]) {
			t.Errorf("Appended instance data incorrect.\nexpect:%s\ngot:%s",
				correctData, data)
			break
		}
	}

	// Check options
	if !th.StringSliceEqual(correctOptions, indexedOptions) {
		t.Errorf("Appended instance options incorrect.\nexpect:%s\ngot:%s",
			correctOptions, indexedOptions)
	}
}
