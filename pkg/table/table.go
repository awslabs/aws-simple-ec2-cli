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

package table

import (
	"fmt"
	"sort"
	"strings"

	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/questionModel"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/olekukonko/tablewriter"
)

// Build a table
func BuildTable(data [][]string, header []string) string {

	tableBuilder := &strings.Builder{}
	table := tablewriter.NewWriter(tableBuilder)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	// Fill the table with data
	for _, v := range data {
		table.Append(v)
	}

	// Set header if provided
	if header != nil {
		table.SetHeader(header)
	}

	table.Render()
	tableString := tableBuilder.String()

	return tableString
}

// Append all launch template EBS blocks, if applicable
func AppendTemplateEbs(data [][]string, mappings []*ec2.LaunchTemplateBlockDeviceMapping) [][]string {
	if mappings != nil && len(mappings) > 0 {

		ebsData := [][]string{}
		for _, block := range mappings {
			ebsName := fmt.Sprintf("%s", *block.DeviceName)
			if block.Ebs != nil {
				if block.Ebs.VolumeType != nil {
					ebsName += fmt.Sprintf("(%s)", *block.Ebs.VolumeType)
				}
				if block.Ebs.VolumeSize != nil {
					ebsName += fmt.Sprintf(": %d GiB", *block.Ebs.VolumeSize)
				}
			}
			ebsData = append(ebsData, []string{"", ebsName})
		}
		ebsData[0][0] = "EBS Volumes"
		data = append(data, ebsData...)
	}

	return data
}

// Append all  EBS blocks, if applicable
func AppendEbs(data [][]string, mappings []*ec2.BlockDeviceMapping) ([][]string, questionModel.Row) {
	ebsData := [][]string{}
	if mappings != nil && len(mappings) > 0 {
		for _, block := range mappings {
			ebsName := fmt.Sprintf("%s", *block.DeviceName)
			if block.Ebs != nil {
				if block.Ebs.VolumeType != nil {
					ebsName += fmt.Sprintf("(%s)", *block.Ebs.VolumeType)
				}
				if block.Ebs.VolumeSize != nil {
					ebsName += fmt.Sprintf(": %d GiB", *block.Ebs.VolumeSize)
				}
			}
			ebsData = append(ebsData, []string{"", ebsName})
		}
		ebsData[0][0] = "EBS Volumes"
		data = append(data, ebsData...)
	}

	return data, ebsData
}

// Append all security groups
func AppendSecurityGroups(data [][]string, securityGroups []*ec2.SecurityGroup) ([][]string, questionModel.Row) {
	securityGroupData := [][]string{}
	if securityGroups != nil && len(securityGroups) > 0 {
		for _, group := range securityGroups {
			groupName := *group.GroupId
			groupTagName := ec2helper.GetTagName(group.Tags)
			if groupTagName != nil {
				groupName = fmt.Sprintf("%s(%s)", *groupTagName, groupName)
			}
			securityGroupData = append(securityGroupData, []string{"", groupName})
		}
		securityGroupData[0][0] = cli.ResourceSecurityGroup
		data = append(data, securityGroupData...)
	}

	return data, securityGroupData
}

// Append all launch template network interfaces, if applicable
func AppendTemplateNetworkInterfaces(h *ec2helper.EC2Helper, data [][]string,
	networkInterfaces []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecification) ([][]string, error) {
	if networkInterfaces != nil && len(networkInterfaces) > 0 {
		subnetsMap := map[string]string{}

		// Gather information about subnets
		for _, networkInterface := range networkInterfaces {
			if networkInterface.SubnetId != nil {
				subnet, err := h.GetSubnetById(*networkInterface.SubnetId)
				if err != nil {
					return nil, err
				}

				// Format the name of the subnet
				vpcId := *subnet.VpcId
				subnetName := fmt.Sprintf("%s:%s", vpcId, *subnet.SubnetId)
				subnetTagName := ec2helper.GetTagName(subnet.Tags)
				if subnetTagName != nil {
					subnetName = fmt.Sprintf("%s(%s:%s)", *subnetTagName, vpcId, *subnet.SubnetId)
				}

				subnetsMap[*subnet.SubnetId] = subnetName
			}
		}

		if len(subnetsMap) > 0 {
			// Append to the table
			subnetsData := [][]string{}
			counter := 1

			// Sort the keys
			var keys []string
			for k := range subnetsMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				subnetsData = append(subnetsData, []string{"", fmt.Sprintf("%d.%s", counter, subnetsMap[key])})
				counter++
			}
			subnetsData[0][0] = "Subnets"
			data = append(data, subnetsData...)
			return data, nil
		}
	}

	data = append(data, []string{"Subnets", "not specified"})

	return data, nil
}

/*
Append all instances. When a list of already added instance IDs is provided, the function will identify
which instance IDs are already added to selection and exclude the added instance IDs from the table
*/
func AppendInstances(data [][]string, indexedOptions []string, instances []*ec2.Instance,
	addedInstanceIds []string) ([][]string, []string, int, []questionModel.Row) {
	counter := 0
	rows := []questionModel.Row{}
	for _, instance := range instances {
		if addedInstanceIds != nil {
			// If this instance is already added, just don't display it here
			isFound := false
			for _, addedInstanceId := range addedInstanceIds {
				if addedInstanceId == *instance.InstanceId {
					isFound = true
					break
				}
			}
			if isFound {
				continue
			}
		}

		instanceName := *instance.InstanceId
		instanceTagName := ec2helper.GetTagName(instance.Tags)
		if instanceTagName != nil {
			instanceName = fmt.Sprintf("%s(%s)", *instanceTagName, *instance.InstanceId)
		}
		firstRow := []string{instanceName, "", ""}
		indexedOptions = append(indexedOptions, *instance.InstanceId)
		counter++

		// Extract all tags that are not Name
		displayTags := []*ec2.Tag{}
		for _, tag := range instance.Tags {
			if *tag.Key != "Name" {
				displayTags = append(displayTags, tag)
			}
		}

		// Append the first tag, if applicable
		if len(displayTags) > 0 {
			firstRow[1] = *displayTags[0].Key
			firstRow[2] = *displayTags[0].Value
		}

		// Append the main row
		rowData := [][]string{firstRow}

		// Append subrows, if applicable
		for i := 1; i < len(displayTags); i++ {
			rowData = append(rowData, []string{"", *displayTags[i].Key, *displayTags[i].Value})
		}
		data = append(data, rowData...)
		rows = append(rows, rowData)
	}

	return data, indexedOptions, counter, rows
}
