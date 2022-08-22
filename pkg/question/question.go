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

package question

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"simple-ec2/pkg/cfn"
	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/iamhelper"
	"simple-ec2/pkg/questionModel"
	"simple-ec2/pkg/table"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/ec2pricing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/briandowns/spinner"
	"golang.org/x/exp/slices"
)

var DefaultCapacityTypeText = struct {
	OnDemand, Spot string
}{
	OnDemand: "On-Demand",
	Spot:     "Spot",
}

type CheckInput func(*ec2helper.EC2Helper, string) bool

type AskQuestionInput struct {
	QuestionString    string
	OptionsString     *string
	DefaultOptionRepr *string
	DefaultOption     *string
	IndexedOptions    []string
	StringOptions     []string
	AcceptAnyString   bool
	AcceptAnyInteger  bool
	EC2Helper         *ec2helper.EC2Helper
	Fns               []CheckInput
}

// Ask for the region to use
func AskRegion(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultRegion string) (*string, error) {
	regionDescription := getRegionDescriptions()

	// Get all enabled regions and make sure no error
	regions, err := h.GetEnabledRegions()
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Fill the data used for drawing a table and the options map
	// var row []string
	for _, region := range regions {
		row := []string{}
		indexedOptions = append(indexedOptions, *region.RegionName)

		row = append(row, fmt.Sprintf("%s", *region.RegionName))
		desc, found := (*regionDescription)[*region.RegionName]
		if found {
			row = append(row, desc)
			data = append(data, row)
		}
	}

	defaultOption := h.Sess.Config.Region
	if slices.Contains(indexedOptions, defaultRegion) {
		defaultOption = &defaultRegion
	}

	headers := []string{"Region", "Description"}
	question := "Select a region for the instance:"

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		Rows:           questionModel.CreateSingleLineRows(data),
		QuestionString: question,
		DefaultOption:  *defaultOption,
		IndexedOptions: indexedOptions,
		HeaderStrings:  headers,
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

func getRegionDescriptions() *map[string]string {
	partition := endpoints.AwsPartition()
	regions := partition.Regions()

	// Put in all descriptions
	descs := map[string]string{}
	for id, region := range regions {
		descs[id] = region.Description()
	}

	// Hardcode ap-northeast-3, because it's not included in the SDK
	descs["ap-northeast-3"] = "Asia Pacific (Osaka)"

	return &descs
}

/*
Ask for the launch template to use. The result will either be a launch template id or response.No,
indicating not using a launch template.
*/
func AskLaunchTemplate(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultLaunchTemplateId string) (*string, error) {
	// Get all launch templates. If no launch template is available, skip this question
	launchTemplates, err := h.GetLaunchTemplatesInRegion()
	if err != nil || len(launchTemplates) <= 0 {
		return aws.String(cli.ResponseNo), nil
	}

	data := [][]string{}
	indexedOptions := []string{}

	noUseOptionRepr, noUseOptionValue := "Do not use launch template", cli.ResponseNo
	defaultOption := noUseOptionValue
	// Fill the data used for drawing a table and the options map
	for _, launchTemplate := range launchTemplates {
		if *launchTemplate.LaunchTemplateId == defaultLaunchTemplateId {
			defaultOption = defaultLaunchTemplateId
		}
		indexedOptions = append(indexedOptions, *launchTemplate.LaunchTemplateId)

		launchTemplateName := fmt.Sprintf("%s(%s)", *launchTemplate.LaunchTemplateName,
			*launchTemplate.LaunchTemplateId)
		data = append(data, []string{launchTemplateName,
			strconv.FormatInt(*launchTemplate.LatestVersionNumber, 10)})
	}

	// Add the do not use launch template option at the end
	indexedOptions = append(indexedOptions, noUseOptionValue)
	data = append(data, []string{noUseOptionRepr})
	question := "Select a Launch Template:"
	headers := []string{"Launch Template", "Latest Version"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		DefaultOption:  defaultOption,
		HeaderStrings:  headers,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
		QuestionString: question,
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

// Ask for the launch template version to use. The result will be a launch template version
func AskLaunchTemplateVersion(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	launchTemplateId string, defaultTemplateVersion string) (*string, error) {
	launchTemplateVersions, err := h.GetLaunchTemplateVersions(launchTemplateId, nil)
	if err != nil || launchTemplateVersions == nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Fill the data used for drawing a table and the options map
	var defaultOption string
	for _, launchTemplateVersion := range launchTemplateVersions {
		versionString := strconv.FormatInt(*launchTemplateVersion.VersionNumber, 10)
		// Note that launch template versions are sorted, so basically index+1 = version
		indexedOptions = append(indexedOptions, versionString)

		// If the version has description, show it
		var versionDescription string
		if launchTemplateVersion.VersionDescription != nil {
			versionDescription = *launchTemplateVersion.VersionDescription
		} else {
			versionDescription = "-"
		}

		data = append(data, []string{fmt.Sprintf("%s.", versionString), versionDescription})

		if versionString == defaultTemplateVersion || defaultOption == "" && *launchTemplateVersion.DefaultVersion {
			defaultOption = versionString
		}
	}

	question := "Select the Launch Template version:"
	headers := []string{"Version Number", "Description"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		DefaultOption:  defaultOption,
		HeaderStrings:  headers,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
		QuestionString: question,
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

// Ask whether the users want to enter instance type themselves or seek advice
func AskIfEnterInstanceType(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultInstanceType string) (*string, error) {
	instanceTypes, err := h.GetInstanceTypesInRegion()
	if err != nil {
		return nil, err
	}

	// Use user default instance type if applicable. If not, find the default free instance type.
	// If no default instance type available, simply don't give default option
	var defaultOption *string
	instanceTypeNames := []string{}

	for _, instanceTypeInfo := range instanceTypes {
		instanceTypeNames = append(instanceTypeNames, *instanceTypeInfo.InstanceType)
	}

	if slices.Contains(instanceTypeNames, defaultInstanceType) {
		defaultOption = &defaultInstanceType
	} else {
		defaultInstanceType, err := h.GetDefaultFreeTierInstanceType()
		if err != nil {
			return nil, err
		}
		if defaultInstanceType != nil {
			defaultOption = defaultInstanceType.InstanceType
		}
	}

	data := [][]string{{"Enter the instance type"}, {"Provide vCPUs and memory information for advice"},
		{fmt.Sprintf("Use the default instance type, [%s]", *defaultOption)}}
	indexedOptions := []string{cli.ResponseYes, cli.ResponseNo, *defaultOption}
	question := "How do you want to choose the instance type?"

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		IndexedOptions: indexedOptions,
		DefaultOption:  *defaultOption,
		Rows:           questionModel.CreateSingleLineRows(data),
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

// Ask the users to enter instace type
func AskInstanceType(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultInstanceType string) (*string, error) {
	instanceTypes, err := h.GetInstanceTypesInRegion()
	if err != nil {
		return nil, err
	}

	stringOptions := []string{}

	// Add all queried instance types to options
	for _, instanceTypeInfo := range instanceTypes {
		stringOptions = append(stringOptions, *instanceTypeInfo.InstanceType)
	}

	// Use user default instance type if applicable. If not, find the default free instance type.
	// If no default instance type available, simply don't give default option
	var defaultOption *string
	if slices.Contains(stringOptions, defaultInstanceType) {
		defaultOption = &defaultInstanceType // Set to User default instance type
	} else {
		defaultInstanceType, err := h.GetDefaultFreeTierInstanceType()
		if err != nil {
			return nil, err
		}
		if defaultInstanceType != nil {
			defaultOption = defaultInstanceType.InstanceType
		}
	}

	question := "Enter the instance type to be used: (eg. m5.xlarge, c5.xlarge)"
	instanceValidation := func(h *ec2helper.EC2Helper, instanceType string) bool {
		for _, instance := range instanceTypes {
			if *instance.InstanceType == instanceType {
				return true
			}
		}
		return false
	}

	model := &questionModel.PlainText{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  *defaultOption,
		EC2Helper:      h,
		Fns:            []questionModel.CheckInput{instanceValidation},
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetTextAnswer()
	return &answer, nil
}

// Ask the users to enter instance type vCPUs
func AskInstanceTypeVCpu(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper) (string, error) {
	question := "Enter the number of vCPUs to be used:"

	model := &questionModel.PlainText{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  "2",
		EC2Helper:      h,
		Fns:            []questionModel.CheckInput{ec2helper.ValidateInteger},
	})

	if err != nil {
		return "", err
	}

	return model.GetTextAnswer(), nil
}

// Ask the users to enter instace type memory
func AskInstanceTypeMemory(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper) (string, error) {
	question := "Enter the amount of memory(in GiB) to be used:"

	model := &questionModel.PlainText{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  "2",
		EC2Helper:      h,
		Fns:            []questionModel.CheckInput{ec2helper.ValidateInteger},
	})

	if err != nil {
		return "", err
	}

	return model.GetTextAnswer(), nil
}

// Ask the users to select an instance type given the options from Instance Selector
func AskInstanceTypeInstanceSelector(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	instanceSelector ec2helper.InstanceSelector,
	vcpus, memory string) (*string, error) {
	// Parse string to numbers
	vcpusInt, err := strconv.Atoi(vcpus)
	if err != nil {
		return nil, err
	}
	memoryInt, err := strconv.Atoi(memory)
	if err != nil {
		return nil, err
	}

	// get instance types from instance selector
	instanceTypes, err := h.GetInstanceTypesFromInstanceSelector(instanceSelector, vcpusInt, memoryInt)
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	if len(instanceTypes) > 0 {
		for _, instanceType := range instanceTypes {
			// Fill the data with properties
			data = append(data, []string{
				*instanceType.InstanceType,
				strconv.FormatInt(*instanceType.VCpuInfo.DefaultVCpus, 10),
				strconv.FormatFloat(float64(*instanceType.MemoryInfo.SizeInMiB)/1024, 'f', 2, 64) + " GiB",
				strconv.FormatBool(*instanceType.InstanceStorageSupported),
			})

			indexedOptions = append(indexedOptions, *instanceType.InstanceType)
		}
	} else {
		return nil, errors.New("No suggested instance types available. Please enter vCPUs and memory again. ")
	}

	question := "Select an instance type:"
	headers := []string{"Instance Type", "vCPUs", "Memory", "Instance Storage"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
		HeaderStrings:  headers,
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

/*
Ask the users to select an image. This function is different from other question-asking functions.
It returns not a string but an ec2.Image object
*/
func AskImage(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	instanceType string, defaultImageId string) (*ec2.Image, error) {
	// get info about the instance type
	instanceTypeInfo, err := h.GetInstanceType(instanceType)
	if err != nil {
		return nil, err
	}

	// Use instance-store if supported
	rootDeviceType := "ebs"
	if *instanceTypeInfo.InstanceStorageSupported {
		rootDeviceType = "instance-store"
	}

	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " fetching images"
	s.Color("blue", "bold")
	s.Start()
	defaultImages, err := h.GetLatestImages(&rootDeviceType, instanceTypeInfo.ProcessorInfo.SupportedArchitectures)
	if err != nil {
		return nil, err
	}
	s.Stop()

	data := [][]string{}
	indexedOptions := []string{}

	var defaultOption string
	if defaultImages != nil && len(*defaultImages) > 0 {
		priority := ec2helper.GetImagePriority()
		// Use the user default image if available as the default choice. If not then pick the available image with the highest priority.
		for _, image := range *defaultImages {
			if *image.ImageId == defaultImageId {
				defaultOption = defaultImageId
				break
			}
		}
		if defaultOption == "" {
			for _, osName := range priority {
				image, found := (*defaultImages)[osName]
				if found {
					defaultOption = *image.ImageId
					break
				}
			}
		}

		// Add all default images to indexed options, with priority
		for _, osName := range priority {
			image, found := (*defaultImages)[osName]
			if found {
				indexedOptions = append(indexedOptions, *image.ImageId)
				data = append(data, []string{osName, *image.ImageId, *image.CreationDate})
			}
		}
	}

	headers := []string{"Operating System", "Image ID", "Creation Date"}
	question := "Select an AMI for the instance:"

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		HeaderStrings:  headers,
		QuestionString: question,
		DefaultOption:  defaultOption,
		Rows:           questionModel.CreateSingleLineRows(data),
		IndexedOptions: indexedOptions,
		EC2Helper:      h,
		Fns:            []questionModel.CheckInput{ec2helper.ValidateImageId},
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()

	// Find the image information
	if defaultImages != nil {
		for _, image := range *defaultImages {
			if *image.ImageId == answer {
				return image, nil
			}
		}
	}

	return nil, errors.New(fmt.Sprintf("No image information for %s found", answer))
}

// Ask if the users want to keep EBS volumes after instance termination
func AskKeepEbsVolume(qh *questionModel.QuestionModelHelper, defaultKeepEbs bool) (string, error) {
	question := "Persist EBS Volume(s) after the instance is terminated?"
	answer, err := questionModel.AskYesNoQuestion(qh, question, defaultKeepEbs)

	if err != nil {
		return "", err
	}

	return answer, nil
}

// Ask if the users want to attach IAM profile to instance
func AskIamProfile(qh *questionModel.QuestionModelHelper, i *iamhelper.IAMHelper, defaultIamProfile string) (string, error) {
	input := &iam.ListInstanceProfilesInput{
		MaxItems: aws.Int64(10),
	}

	output, err := i.Client.ListInstanceProfiles(input)
	if err != nil {
		return "", err
	}

	instanceProfiles := output.InstanceProfiles
	for {
		if *output.IsTruncated {
			input = &iam.ListInstanceProfilesInput{
				MaxItems: aws.Int64(10),
				Marker:   aws.String(*output.Marker),
			}
			output, err = i.Client.ListInstanceProfiles(input)
			if err != nil {
				return "", err
			}
			if len(output.InstanceProfiles) > 0 {
				instanceProfiles = append(instanceProfiles, output.InstanceProfiles...)
			}
		} else {
			break
		}
	}

	defaultOptionValue := cli.ResponseNo
	noOptionRepr, noOptionValue := "Do not attach IAM profile", cli.ResponseNo

	data := [][]string{}
	indexedOptions := []string{}
	if len(instanceProfiles) > 0 {
		counter := 0
		for _, profile := range instanceProfiles {
			indexedOptions = append(indexedOptions, *profile.InstanceProfileName)
			data = append(data, []string{*profile.InstanceProfileName, *profile.InstanceProfileId, profile.CreateDate.String()})
			if defaultIamProfile == *profile.InstanceProfileName {
				defaultOptionValue = *profile.InstanceProfileName
			}
			counter++
		}
	}

	// Add the do not attach IAM profile option at the end
	indexedOptions = append(indexedOptions, noOptionValue)
	data = append(data, []string{noOptionRepr})

	question := "Select an IAM Profile:"
	headers := []string{"PROFILE NAME", "PROFILE ID", "Creation Date"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  defaultOptionValue,
		IndexedOptions: indexedOptions,
		HeaderStrings:  headers,
		Rows:           questionModel.CreateSingleLineRows(data),
	})

	if err != nil {
		return "", err
	}

	return model.GetChoice(), nil
}

// Ask if the users want to set an auto-termination timer for the instance
func AskAutoTerminationTimerMinutes(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultTimer int) (string, error) {
	question := "After how many minutes should the instance terminate? (0 for no auto-termination)"
	defaultOption := strconv.FormatInt(int64(0), 10)
	if defaultTimer != 0 {
		defaultOption = strconv.FormatInt(int64(defaultTimer), 10)
	}

	model := &questionModel.PlainText{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  defaultOption,
		EC2Helper:      h,
		Fns:            []questionModel.CheckInput{ec2helper.ValidateInteger},
	})

	if err != nil {
		return "", err
	}

	return model.GetTextAnswer(), nil
}

// Ask the users to select a VPC
func AskVpc(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper, defaultVpcId string) (*string, error) {
	vpcs, err := h.GetAllVpcs()
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}
	defaultOptionValue := cli.ResponseNew

	// Add VPCs to the data for table
	if vpcs != nil {
		for _, vpc := range vpcs {
			indexedOptions = append(indexedOptions, *vpc.VpcId)

			vpcName := *vpc.VpcId
			vpcTagName := ec2helper.GetTagName(vpc.Tags)
			if vpcTagName != nil {
				vpcName = fmt.Sprintf("%s(%s)", *vpcTagName, *vpc.VpcId)
			}

			if defaultVpcId != "" && *vpc.VpcId == defaultVpcId || *vpc.IsDefault && defaultOptionValue == cli.ResponseNew {
				defaultOptionValue = *vpc.VpcId
			}

			data = append(data, []string{vpcName, *vpc.CidrBlock})
		}
	}

	indexedOptions = append(indexedOptions, cli.ResponseNew)
	data = append(data, []string{fmt.Sprintf("Create new VPC with default CIDR and %d subnets", cfn.RequiredAvailabilityZones)})

	question := "Select the VPC for the instance:"
	headers := []string{"VPC", "CIDR Block"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  defaultOptionValue,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
		HeaderStrings:  headers,
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

// Ask the users to select a subnet
func AskSubnet(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	vpcId string, defaultSubnetId string) (*string, error) {
	subnets, err := h.GetSubnetsByVpc(vpcId)
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}
	var defaultOptionValue *string = nil

	// Add security groups to the data for table
	for _, subnet := range subnets {
		if defaultSubnetId != "" && *subnet.SubnetId == defaultSubnetId {
			defaultOptionValue = subnet.SubnetId
		}
		indexedOptions = append(indexedOptions, *subnet.SubnetId)

		subnetName := *subnet.SubnetId
		subnetTagName := ec2helper.GetTagName(subnet.Tags)
		if subnetTagName != nil {
			subnetName = fmt.Sprintf("%s(%s)", *subnetTagName, *subnet.SubnetId)
		}

		data = append(data, []string{subnetName, *subnet.AvailabilityZone, *subnet.CidrBlock})
	}

	if defaultOptionValue == nil {
		defaultOptionValue = subnets[0].SubnetId
	}

	question := "Select the subnet for the instance:"
	headers := []string{"Subnet", "Availability Zone", "CIDR Block"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  *defaultOptionValue,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
		HeaderStrings:  headers,
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

// Ask the users to select a subnet placeholder
func AskSubnetPlaceholder(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultAzId string) (*string, error) {
	availabilityZones, err := h.GetAvailableAvailabilityZones()
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Add availability zones to the data for table
	var defaultOptionValue *string
	for _, zone := range availabilityZones {
		if defaultAzId != "" && *zone.ZoneId == defaultAzId {
			defaultOptionValue = zone.ZoneName
		}
		indexedOptions = append(indexedOptions, *zone.ZoneName)

		data = append(data, []string{*zone.ZoneName, *zone.ZoneId})
	}

	if defaultOptionValue == nil {
		defaultOptionValue = &data[0][0]
	}

	question := "Select the availability zone for the new subnets:"
	headers := []string{"Zone Name", "Zone ID"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  *defaultOptionValue,
		IndexedOptions: indexedOptions,
		HeaderStrings:  headers,
		Rows:           questionModel.CreateSingleLineRows(data),
	})

	if err != nil {
		return nil, err
	}

	answer := model.GetChoice()
	return &answer, nil
}

// Ask the users to select security groups
func AskSecurityGroups(qh *questionModel.QuestionModelHelper,
	groups []*ec2.SecurityGroup, defaultSecurityGroups []*ec2.SecurityGroup) ([]string, error) {
	question := "Select the security groups for the instance:"
	data := [][]string{}
	indexedOptions := []string{}

	// Add security groups to the data for table
	if groups != nil {
		for _, group := range groups {
			indexedOptions = append(indexedOptions, *group.GroupId)

			groupName := *group.GroupId
			groupTagName := ec2helper.GetTagName(group.Tags)
			if groupTagName != nil {
				groupName = fmt.Sprintf("%s(%s)", *groupTagName, *group.GroupId)
			}

			data = append(data, []string{groupName, *group.Description})
		}
	}

	defaultOptionList := []string{}
	for _, group := range defaultSecurityGroups {
		defaultOptionList = append(defaultOptionList, *group.GroupId)
	}

	// Add "new" option
	indexedOptions = append(indexedOptions, cli.ResponseNew)
	data = append(data, []string{"Create a new security group that enables SSH"})

	headers := []string{"Security Group", "Description"}

	model := &questionModel.MultiSelectList{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString:    question,
		DefaultOptionList: defaultOptionList,
		IndexedOptions:    indexedOptions,
		HeaderStrings:     headers,
		Rows:              questionModel.CreateSingleLineRows(data),
	})

	if err != nil {
		return nil, err
	}

	return model.GetSelectedValues(), nil
}

// Ask the users to select a security group placeholder
func AskSecurityGroupPlaceholder(qh *questionModel.QuestionModelHelper) (string, error) {
	data := [][]string{}
	rows := []questionModel.Row{}
	_ = rows
	indexedOptions := []string{}

	// Add the options
	indexedOptions = append(indexedOptions, cli.ResponseAll)
	indexedOptions = append(indexedOptions, cli.ResponseNew)
	data = append(data, []string{"Use the default security group"})
	data = append(data, []string{"Create and use a new security group for SSH"})

	question := "Select the security group for the new VPC:"

	model := &questionModel.SingleSelectList{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
	})

	if err != nil {
		return "", err
	}

	return model.GetChoice(), nil
}

// Print confirmation information for instance launch and ask for confirmation
func AskConfirmationWithTemplate(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo) (*string, error) {
	versions, err := h.GetLaunchTemplateVersions(simpleConfig.LaunchTemplateId,
		&simpleConfig.LaunchTemplateVersion)
	if err != nil {
		return nil, err
	}

	templateData := versions[0].LaunchTemplateData

	data := [][]string{}
	data = append(data, []string{"Region", *h.Sess.Config.Region})

	// Find and append all the subnets specified in the templates, if applicable
	if simpleConfig.SubnetId != "" {
		data = append(data, []string{"Subnet", simpleConfig.SubnetId})
	} else {
		data, err = table.AppendTemplateNetworkInterfaces(h, data, templateData.NetworkInterfaces)
		if err != nil {
			return nil, err
		}
	}

	// Append instance type
	instanceType := "not specified"
	if simpleConfig.InstanceType != "" {
		instanceType = simpleConfig.InstanceType
	} else if templateData.InstanceType != nil {
		instanceType = *templateData.InstanceType
	}
	data = append(data, []string{"Instance Type", instanceType})

	// Append image id
	imageId := "not specified"
	if simpleConfig.ImageId != "" {
		imageId = simpleConfig.ImageId
	} else if templateData.ImageId != nil {
		imageId = *templateData.ImageId
		// Give config file an image ID so it can be queried correctly later
		simpleConfig.ImageId = *templateData.ImageId
	}
	data = append(data, []string{"Image ID", imageId})

	// Append all EBS blocks, if applicable
	data = table.AppendTemplateEbs(data, templateData.BlockDeviceMappings)

	answer, err := askConfigTableQuestion(qh, data)

	if err != nil {
		return nil, err
	}

	return &answer, nil
}

// Print confirmation information for instance launch and ask for confirmation
func AskConfirmationWithInput(qh *questionModel.QuestionModelHelper, simpleConfig *config.SimpleInfo,
	detailedConfig *config.DetailedInfo, allowEdit bool) (string, error) {
	// If new subnets will be created, skip formatting the subnet info.
	subnetInfo := "New Subnet"
	subnet := detailedConfig.Subnet
	if simpleConfig.NewVPC {
		/*
			If the subnet id is not empty and is not a real subnet id,
			it will be a placeholder subnet with an availability zone.
		*/
		if simpleConfig.SubnetId != "" && simpleConfig.SubnetId[0:6] != "subnet" {
			subnetInfo += " in " + simpleConfig.SubnetId
		}
	} else {
		subnetInfo = *subnet.SubnetId
		subnetTagName := ec2helper.GetTagName(subnet.Tags)
		if subnetTagName != nil {
			subnetInfo = fmt.Sprintf("%s(%s)", *subnetTagName, *subnet.SubnetId)
		}
	}

	// If a new VPC will be created, skip formatting
	vpcInfo := "New VPC"
	vpc := detailedConfig.Vpc
	if !simpleConfig.NewVPC {
		vpcInfo = *vpc.VpcId
		vpcTagName := ec2helper.GetTagName(vpc.Tags)
		if vpcTagName != nil {
			vpcInfo = fmt.Sprintf("%s(%s)", *vpcTagName, *vpc.VpcId)
		}
	}

	// Get display data ready
	data := [][]string{
		{cli.ResourceRegion, simpleConfig.Region},
		{cli.ResourceVpc, vpcInfo},
		{cli.ResourceSubnet, subnetInfo},
		{cli.ResourceInstanceType, simpleConfig.InstanceType},
		{cli.ResourceCapacityType, simpleConfig.CapacityType},
		{cli.ResourceImage, simpleConfig.ImageId},
	}

	rows := questionModel.CreateSingleLineRows(data)
	indexedOptions := []string{
		"",
		cli.ResourceVpc,
		cli.ResourceSubnet,
		cli.ResourceInstanceType,
		cli.ResourceCapacityType,
		cli.ResourceImage,
	}

	/*
		Append all security groups.
		If security groups were successfully parsed into the detailed config, append them here.
		Otherwise, look for placeholder security groups, such as "all" and "new" in the simple config.
		Also, use the bool value to tell the next block what question option to use for security group
	*/
	if detailedConfig.SecurityGroups != nil {
		_, row := table.AppendSecurityGroups(data, detailedConfig.SecurityGroups)
		if len(row) != 0 {
			rows = append(rows, row)
			indexedOptions = append(indexedOptions, cli.ResourceSecurityGroup)
		}
	} else if simpleConfig.SecurityGroupIds != nil && len(simpleConfig.SecurityGroupIds) >= 1 {
		if simpleConfig.SecurityGroupIds[0] == cli.ResponseNew {
			rows = append(rows, [][]string{{cli.ResourceSecurityGroup, "New security group for SSH"}})
		} else if simpleConfig.SecurityGroupIds[0] == cli.ResponseAll {
			rows = append(rows, [][]string{{cli.ResourceSecurityGroup, "New default security group"}})
		}
	}

	if ec2helper.HasEbsVolume(detailedConfig.Image) {
		rows = append(rows, [][]string{{cli.ResourceKeepEbsVolume,
			strconv.FormatBool(simpleConfig.KeepEbsVolumeAfterTermination)}})
		indexedOptions = append(indexedOptions, cli.ResourceKeepEbsVolume)
	}

	if detailedConfig.Image.PlatformDetails != nil &&
		ec2helper.IsLinux(*detailedConfig.Image.PlatformDetails) {
		if simpleConfig.AutoTerminationTimerMinutes > 0 {
			rows = append(rows, [][]string{{cli.ResourceAutoTerminationTimer,
				strconv.Itoa(simpleConfig.AutoTerminationTimerMinutes)}})
		} else {
			rows = append(rows, [][]string{{cli.ResourceAutoTerminationTimer, "None"}})
		}
		indexedOptions = append(indexedOptions, cli.ResourceAutoTerminationTimer)
	}

	// Append all EBS blocks, if applicable
	blockDeviceMappings := detailedConfig.Image.BlockDeviceMappings
	if len(blockDeviceMappings) != 0 {
		_, row := table.AppendEbs(data, blockDeviceMappings)
		rows = append(rows, row)
		indexedOptions = append(indexedOptions, "")
	}

	// Append instance store, if applicable
	if detailedConfig.InstanceTypeInfo.InstanceStorageInfo != nil {
		rows = append(rows, [][]string{{"Instance Storage", fmt.Sprintf("%d GB",
			*detailedConfig.InstanceTypeInfo.InstanceStorageInfo.TotalSizeInGB)}})
		indexedOptions = append(indexedOptions, "")
	}

	// Append instance profile, if applicable
	if simpleConfig.IamInstanceProfile != "" {
		rows = append(rows, [][]string{{cli.ResourceIamInstanceProfile, simpleConfig.IamInstanceProfile}})
		indexedOptions = append(indexedOptions, cli.ResourceIamInstanceProfile)
	}

	if simpleConfig.BootScriptFilePath != "" {
		rows = append(rows, [][]string{{cli.ResourceBootScriptFilePath, simpleConfig.BootScriptFilePath}})
		indexedOptions = append(indexedOptions, cli.ResourceBootScriptFilePath)
	}
	if len(simpleConfig.UserTags) != 0 {
		var tags [][]string
		index := 0
		for k, v := range simpleConfig.UserTags {
			tag := fmt.Sprintf("%s|%s", k, v)
			if index == 0 {
				tags = append(tags, []string{cli.ResourceUserTags, tag})
			} else {
				tags = append(tags, []string{"", tag})
			}
			index++
		}
		rows = append(rows, tags)
		indexedOptions = append(indexedOptions, cli.ResourceUserTags)
	}

	model := &questionModel.Confirmation{}
	model.SetAllowEdit(allowEdit)
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		IndexedOptions: indexedOptions,
		Rows:           rows,
	})

	if err != nil {
		return "", err
	}

	return model.GetChoice(), nil
}

// Ask if the user wants to save the config as a JSON config file
func AskSaveConfig(qh *questionModel.QuestionModelHelper) (string, error) {
	question := "Do you want to save the configuration above as a JSON file that can be used in non-interactive mode and as question defaults? "
	answer, err := questionModel.AskYesNoQuestion(qh, question, false)

	if err != nil {
		return "", err
	}

	return answer, nil
}

// Ask the instance id to be connected
func AskInstanceId(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper) (*string, error) {
	// Only include running states
	states := []string{
		ec2.InstanceStateNameRunning,
	}

	instances, err := h.GetInstancesByState(states)
	if err != nil {
		return nil, err
	}

	// If no instance is available, simply don't ask
	if len(instances) <= 0 {
		return nil, errors.New("No instance available to connect")
	}

	data := [][]string{}
	indexedOptions := []string{}

	data, indexedOptions, _, rows := table.AppendInstances(data, indexedOptions, instances, nil)

	headers := []string{"Instance", "Tag-Key", "Tag-Value"}
	question := "Select the instance you want to connect to: "

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		Rows:           rows,
		QuestionString: question,
		HeaderStrings:  headers,
		IndexedOptions: indexedOptions,
	})

	answer := model.GetChoice()
	return &answer, err
}

// Ask the instance IDs to be terminated
func AskInstanceIds(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper, addedInstanceIds []string) ([]string, error) {
	// Only include non-terminated states
	states := []string{
		ec2.InstanceStateNamePending,
		ec2.InstanceStateNameRunning,
		ec2.InstanceStateNameStopping,
		ec2.InstanceStateNameStopped,
	}

	instances, err := h.GetInstancesByState(states)
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	data, indexedOptions, _, rows := table.AppendInstances(data, indexedOptions, instances,
		addedInstanceIds)
	_ = rows

	// There are no instances available for termination in selected region
	if len(data) <= 0 && len(addedInstanceIds) == 0 {
		return nil, errors.New("No instance available in selected region for termination")
	}

	// Since no more instance(s) are available for termination, proceed with current selection
	if len(data) == 0 && len(addedInstanceIds) > 0 {
		return nil, nil
	}

	headers := []string{"Instance", "Tag-Key", "Tag-Value"}
	question := "Select the instances you want to terminate: "

	model := &questionModel.MultiSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		HeaderStrings:  headers,
		IndexedOptions: indexedOptions,
		Rows:           rows,
	})

	answer := model.GetSelectedValues()
	return answer, err
}

// AskBootScriptConfirmation confirms if the user should be prompted to enter in a bootscript
func AskBootScriptConfirmation(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultBootScript string) (string, error) {
	question := "Would you like to add a filepath to the instance boot script?"
	answer, err := questionModel.AskYesNoQuestion(qh, question, defaultBootScript != "")

	if err != nil {
		return "", err
	}

	return answer, nil
}

// AskBootScript prompts the user for a filepath to an optional boot script
func AskBootScript(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper, defaultBootScript string) (string, error) {
	question := "Enter a filepath to instance boot script. Enter \"None\" for no bootscript:"

	noEntryValidation := func(h *ec2helper.EC2Helper, filepath string) bool {
		return strings.ToLower(filepath) == strings.ToLower("None")
	}

	model := &questionModel.PlainText{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  defaultBootScript,
		EC2Helper:      h,
		Fns:            []questionModel.CheckInput{ec2helper.ValidateFilepath, noEntryValidation},
	})

	if err != nil {
		return "", err
	}

	return model.GetTextAnswer(), nil
}

// AskUserTagsConfirmation confirms if the user should be prompted to enter in tags
func AskUserTagsConfirmation(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultTags map[string]string) (string, error) {
	question := "Would you like to add tags to instances and persisted volumes?"
	answer, err := questionModel.AskYesNoQuestion(qh, question, len(defaultTags) != 0)

	if err != nil {
		return "", err
	}

	return answer, nil
}

// AskUserTags prompts the user for optional tags
func AskUserTags(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	defaultTags map[string]string) (string, error) {
	question := "Enter Key/Value pairs to add tags to instances and persisted volumes:"
	kvs := make([]string, 0, len(defaultTags))
	for key, value := range defaultTags {
		kvs = append(kvs, fmt.Sprintf("%s|%s", key, value))
	}
	defaultOption := strings.Join(kvs, ",")

	model := &questionModel.KeyValue{}
	err := qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  defaultOption,
	})

	if err != nil {
		return "", err
	}

	return model.TagsToString(), nil
}

// AskTerminationConfirmation confirms if the user wants to terminate the selected instanceIds
func AskTerminationConfirmation(qh *questionModel.QuestionModelHelper, instanceIds []string) (string, error) {
	question := fmt.Sprintf("Are you sure you want to terminate %d instance(s): %s ", len(instanceIds), instanceIds)
	answer, err := questionModel.AskYesNoQuestion(qh, question, false)

	if err != nil {
		return "", err
	}

	return answer, nil
}

/*
AskCapacityType asks the capacity type of the instance, either Spot or On-Demand. The user is informed of the
pricing of each type before selection.
*/
func AskCapacityType(qh *questionModel.QuestionModelHelper, instanceType string,
	region string, defaultCapacityType string) (string, error) {
	ec2Pricing := ec2pricing.New(session.New().Copy(aws.NewConfig().WithRegion(region)))
	onDemandPrice, err := ec2Pricing.GetOnDemandInstanceTypeCost(instanceType)
	formattedOnDemandPrice := "N/A"
	if err == nil {
		onDemandPrice = math.Round(onDemandPrice*10000) / 10000
		formattedOnDemandPrice = fmt.Sprintf("$%s/hr", strconv.FormatFloat(onDemandPrice, 'f', -1, 64))
	}

	spotPrice, err := ec2Pricing.GetSpotInstanceTypeNDayAvgCost(instanceType, []string{}, 1)
	formattedSpotPrice := "N/A"
	if err == nil {
		spotPrice = math.Round(spotPrice*10000) / 10000
		formattedSpotPrice = fmt.Sprintf("$%s/hr", strconv.FormatFloat(spotPrice, 'f', -1, 64))
	}

	question := fmt.Sprintf("Select capacity type. Spot instances are available at up to a 90%% discount compared to On-Demand instances,\n" +
		"but they may get interrupted by EC2 with a 2-minute warning")

	indexedOptions := []string{DefaultCapacityTypeText.OnDemand, DefaultCapacityTypeText.Spot}
	defaultOption := DefaultCapacityTypeText.OnDemand
	if slices.Contains(indexedOptions, defaultCapacityType) {
		defaultOption = defaultCapacityType
	}

	data := [][]string{{DefaultCapacityTypeText.OnDemand, formattedOnDemandPrice}, {DefaultCapacityTypeText.Spot, formattedSpotPrice}}

	headers := []string{"Capacity Type", "Price"}

	model := &questionModel.SingleSelectList{}
	err = qh.Svc.AskQuestion(model, &questionModel.QuestionInput{
		QuestionString: question,
		DefaultOption:  defaultOption,
		IndexedOptions: indexedOptions,
		Rows:           questionModel.CreateSingleLineRows(data),
		HeaderStrings:  headers,
	})

	if err != nil {
		return "", err
	}

	return model.GetChoice(), nil
}

// askConfigTableQuestion asks the user to create an instance based on given configurations
func askConfigTableQuestion(qh *questionModel.QuestionModelHelper, tableData [][]string) (string, error) {
	question := "Please confirm if you would like to launch instance with following options:"
	headers := []string{"Configurations", "Values"}

	configList := questionModel.SingleSelectList{}
	configList.InitializeModel(&questionModel.QuestionInput{
		QuestionString: question,
		HeaderStrings:  headers,
		Rows:           questionModel.CreateSingleLineRows(tableData),
	})

	answer, err := questionModel.AskYesNoQuestion(qh, configList.PrintTable(), false)

	if err != nil {
		return "", err
	}

	return answer, nil
}
