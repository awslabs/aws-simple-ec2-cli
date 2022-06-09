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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"simple-ec2/pkg/cfn"
	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/iamhelper"
	"simple-ec2/pkg/table"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/briandowns/spinner"
)

const yesNoOption = "[ yes / no ]"

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

// Ask a question on CLI, with a default input and a list of valid inputs.
func AskQuestion(input *AskQuestionInput) string {
	fmt.Println()
	if input.OptionsString != nil {
		fmt.Print(*input.OptionsString)
	}

	// Keep asking for user input until one valid input in entered
	for {
		// GetQuestion displays question with default values
		GetQuestion(input)

		// Read input from the user and convert CRLF to LF
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.Replace(answer, "\n", "", -1)

		// If no input is entered, simply return the default value, if there is one
		if answer == "" && input.DefaultOption != nil {
			return *input.DefaultOption
		}

		// Check if the answer is a valid index in the indexed options. If so, return the option value
		if input.IndexedOptions != nil {
			index, err := strconv.Atoi(answer)
			if err == nil && index >= 1 && index <= len(input.IndexedOptions) {
				return input.IndexedOptions[index-1]
			}
		}

		// Check if the input matches any string option. If so, return it immediately
		if input.StringOptions != nil {
			for _, input := range input.StringOptions {
				if input == answer {
					return answer
				}
			}
		}

		// Check if any CheckInput function validates the input. If so, return it immediately
		if input.EC2Helper != nil && input.Fns != nil {
			for _, fn := range input.Fns {
				if fn(input.EC2Helper, answer) {
					return answer
				}
			}
		}

		// If an arbitrary integer is allowed, try to parse the input as an integer
		if input.AcceptAnyInteger {
			_, err := strconv.Atoi(answer)
			if err == nil {
				return answer
			}
		}

		// If an arbitrary string is allowed, return the answer anyway
		if input.AcceptAnyString {
			return answer
		}

		// No match at all
		fmt.Println("Input invalid. Please try again.")
	}
}

// GetQuestion displays question with default values
func GetQuestion(input *AskQuestionInput) {
	if input.DefaultOptionRepr != nil {
		fmt.Printf("%s [%s]:  ", input.QuestionString, *input.DefaultOptionRepr)
	} else if input.DefaultOption != nil {
		fmt.Printf("%s [%s]:  ", input.QuestionString, *input.DefaultOption)
	} else {
		fmt.Printf(input.QuestionString + ": ")
	}
}

// Ask for the region to use
func AskRegion(h *ec2helper.EC2Helper) (*string, error) {
	defaultRegion := h.Sess.Config.Region
	regionDescription := getRegionDescriptions()
	const regionPerRow = 1
	const elementPerRegion = 3

	// Get all enabled regions and make sure no error
	regions, err := h.GetEnabledRegions()
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Fill the data used for drawing a table and the options map
	var row []string
	for index, region := range regions {
		indexedOptions = append(indexedOptions, *region.RegionName)

		if index%regionPerRow == 0 {
			row = []string{}
		}

		row = append(row, fmt.Sprintf("%d.", index+1))
		row = append(row, fmt.Sprintf("%s", *region.RegionName))
		desc, found := (*regionDescription)[*region.RegionName]
		if found {
			row = append(row, desc)
		}

		// Append the row to the data when the row is filled with 4 elements
		if len(row) == regionPerRow*elementPerRegion {
			data = append(data, row)
		}
	}

	optionsText := table.BuildTable(data, []string{"Option", "Region", "Description"})
	question := "Region"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  defaultRegion,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

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
func AskLaunchTemplate(h *ec2helper.EC2Helper) *string {
	// Get all launch templates. If no launch template is available, skip this question
	launchTemplates, err := h.GetLaunchTemplatesInRegion()
	if err != nil || len(launchTemplates) <= 0 {
		return aws.String(cli.ResponseNo)
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Fill the data used for drawing a table and the options map
	for index, launchTemplate := range launchTemplates {
		indexedOptions = append(indexedOptions, *launchTemplate.LaunchTemplateId)

		launchTemplateName := fmt.Sprintf("%s(%s)", *launchTemplate.LaunchTemplateName,
			*launchTemplate.LaunchTemplateId)
		data = append(data, []string{fmt.Sprintf("%d.", index+1), launchTemplateName,
			strconv.FormatInt(*launchTemplate.LatestVersionNumber, 10)})
	}

	// Add the do not use launch template option at the end
	defaultOptionRepr, defaultOptionValue := "Do not use launch template", cli.ResponseNo
	indexedOptions = append(indexedOptions, defaultOptionValue)
	data = append(data, []string{fmt.Sprintf("%d.", len(data)+1), defaultOptionRepr})

	optionsText := table.BuildTable(data, []string{"Option", "Launch Template", "Latest Version"})
	question := "Launch Template"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: &defaultOptionRepr,
		DefaultOption:     &defaultOptionValue,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
	})

	return &answer
}

// Ask for the launch template version to use. The result will be a launch template version
func AskLaunchTemplateVersion(h *ec2helper.EC2Helper, launchTemplateId string) (*string, error) {
	launchTemplateVersions, err := h.GetLaunchTemplateVersions(launchTemplateId, nil)
	if err != nil || launchTemplateVersions == nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Fill the data used for drawing a table and the options map
	var defaultVersion string
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

		if *launchTemplateVersion.DefaultVersion {
			defaultVersion = versionString
		}
	}

	optionsText := table.BuildTable(data, []string{"Option(Version Number)", "Description"})
	question := "Launch Template Version"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  &defaultVersion,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return &answer, nil
}

// Ask whether the users want to enter instance type themselves or seek advice
func AskIfEnterInstanceType(h *ec2helper.EC2Helper) (*string, error) {
	// Find the default free instance type. If no default instance type available, simply don't give default option
	var defaultInstanceTypeText *string
	defaultInstanceType, err := h.GetDefaultFreeTierInstanceType()
	if err != nil {
		return nil, err
	}
	if defaultInstanceType != nil {
		defaultInstanceTypeText = defaultInstanceType.InstanceType
	}

	indexedOptions := []string{cli.ResponseYes, cli.ResponseNo}

	optionsText := "1. I will enter the instance type\n2. I need advice given vCPUs and memory\n"
	question := "Instance Select Method"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  defaultInstanceTypeText,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return &answer, nil
}

// Ask the users to enter instace type
func AskInstanceType(h *ec2helper.EC2Helper) (*string, error) {
	instanceTypes, err := h.GetInstanceTypesInRegion()
	if err != nil {
		return nil, err
	}

	// Find the default free instance type. If no default instance type available, simply don't give default option
	var defaultInstanceTypeText *string
	defaultInstanceType, err := h.GetDefaultFreeTierInstanceType()
	if err != nil {
		return nil, err
	}
	if defaultInstanceType != nil {
		defaultInstanceTypeText = defaultInstanceType.InstanceType
	}

	stringOptions := []string{}

	// Add all queried instance types to options
	for _, instanceTypeInfo := range instanceTypes {
		stringOptions = append(stringOptions, *instanceTypeInfo.InstanceType)
	}

	question := "Instance Type (eg. m5.xlarge, c5.xlarge)"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  defaultInstanceTypeText,
		StringOptions:  stringOptions,
	})

	return &answer, nil
}

// Ask the users to enter instance type vCPUs
func AskInstanceTypeVCpu() string {
	question := "vCPUs (integer)"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:   question,
		AcceptAnyInteger: true,
	})

	return answer
}

// Ask the users to enter instace type memory
func AskInstanceTypeMemory() string {
	question := "memory in GiB (integer)"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:   question,
		AcceptAnyInteger: true,
	})

	return answer
}

// Ask the users to select an instance type given the options from Instance Selector
func AskInstanceTypeInstanceSelector(h *ec2helper.EC2Helper, instanceSelector ec2helper.InstanceSelector,
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

	var optionsText string
	if len(instanceTypes) > 0 {
		for index, instanceType := range instanceTypes {
			// Fill the data with properties
			data = append(data, []string{
				fmt.Sprintf("%d.", index+1),
				*instanceType.InstanceType,
				strconv.FormatInt(*instanceType.VCpuInfo.DefaultVCpus, 10),
				strconv.FormatFloat(float64(*instanceType.MemoryInfo.SizeInMiB)/1024, 'f', 2, 64) + " GiB",
				strconv.FormatBool(*instanceType.InstanceStorageSupported),
			})

			indexedOptions = append(indexedOptions, *instanceType.InstanceType)
		}

		optionsText = table.BuildTable(data, []string{"Option", "Instance Type", "vCPUs",
			"Memory", "Instance Storage"})
	} else {
		return nil, errors.New("No suggested instance types available. Please enter vCPUs and memory again. ")
	}

	question := "Instance Type"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return &answer, nil
}

/*
Ask the users to select an image. This function is different from other question-asking functions.
It returns not a string but an ec2.Image object
*/
func AskImage(h *ec2helper.EC2Helper, instanceType string) (*ec2.Image, error) {
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

	var defaultImageRepr, defaultImageId, optionsText string
	if defaultImages != nil && len(*defaultImages) > 0 {
		// Pick the available image with the highest priority as the default choice
		priority := ec2helper.GetImagePriority()
		for _, osName := range priority {
			image, found := (*defaultImages)[osName]
			if found {
				defaultImageRepr = fmt.Sprintf("Latest %s image", osName)
				defaultImageId = *image.ImageId
				break
			}
		}

		// Add all default images to indexed options, with priority
		counter := 0
		for _, osName := range priority {
			image, found := (*defaultImages)[osName]
			if found {
				indexedOptions = append(indexedOptions, *image.ImageId)
				data = append(data, []string{fmt.Sprintf("%d.", counter+1), osName, *image.ImageId,
					*image.CreationDate})
				counter++
			}
		}

		optionsText = table.BuildTable(data, []string{"Option", "Operating System", "Image ID",
			"Creation Date"})
	} else {
		optionsText = "No default images available\n"
	}

	// Add the option to enter an image id
	optionsText += "[ any image id ]: Select the image id\n"

	question := "AMI"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: &defaultImageRepr,
		DefaultOption:     &defaultImageId,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
		EC2Helper:         h,
		Fns:               []CheckInput{ec2helper.ValidateImageId},
	})

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
func AskKeepEbsVolume() string {
	stringOptions := []string{cli.ResponseYes, cli.ResponseNo}
	optionsText := yesNoOption + "\n"
	question := "Persist EBS volume(s) after the instance is terminated?"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  aws.String(cli.ResponseNo),
		OptionsString:  &optionsText,
		StringOptions:  stringOptions,
	})

	return answer
}

// Ask if the users want to attach IAM profile to instance
func AskIamProfile(i *iamhelper.IAMHelper) (string, error) {
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

	data := [][]string{}
	indexedOptions := []string{}
	var optionsText string
	if len(instanceProfiles) > 0 {
		counter := 0
		for _, profile := range instanceProfiles {
			indexedOptions = append(indexedOptions, *profile.InstanceProfileName)
			data = append(data, []string{fmt.Sprintf("%d.", counter+1), *profile.InstanceProfileName, *profile.InstanceProfileId,
				profile.CreateDate.String()})
			counter++
		}
	} else {
		optionsText = "No IAM Profiles available\n"
	}

	// Add the do not attach IAM profile option at the end
	defaultOptionRepr, defaultOptionValue := "Do not attach IAM profile", cli.ResponseNo
	indexedOptions = append(indexedOptions, defaultOptionValue)
	data = append(data, []string{fmt.Sprintf("%d.", len(data)+1), defaultOptionRepr, "", ""})
	optionsText = table.BuildTable(data, []string{"Option", "PROFILE NAME", "PROFILE ID",
		"Creation Date"})

	question := "IAM Profile"
	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: &defaultOptionRepr,
		DefaultOption:     &defaultOptionValue,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
	})

	return answer, nil
}

// Ask if the users want to set an auto-termination timer for the instance
func AskAutoTerminationTimerMinutes() string {
	stringOptions := []string{cli.ResponseNo}
	optionsText := "[ integer ] Auto-termination timer in minutes\n" + "[ no ] No auto-termination" + "\n"
	question := "Auto-termination timer"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:   question,
		DefaultOption:    aws.String(cli.ResponseNo),
		OptionsString:    &optionsText,
		StringOptions:    stringOptions,
		AcceptAnyInteger: true,
	})

	return answer
}

// Ask the users to select a VPC
func AskVpc(h *ec2helper.EC2Helper) (*string, error) {
	vpcs, err := h.GetAllVpcs()
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}
	defaultOptionRepr, defaultOptionValue := "Create new VPC", cli.ResponseNew

	// Add VPCs to the data for table
	if vpcs != nil {
		for index, vpc := range vpcs {
			indexedOptions = append(indexedOptions, *vpc.VpcId)

			vpcName := *vpc.VpcId
			vpcTagName := ec2helper.GetTagName(vpc.Tags)
			if vpcTagName != nil {
				vpcName = fmt.Sprintf("%s(%s)", *vpcTagName, *vpc.VpcId)
			}

			if *vpc.IsDefault {
				defaultOptionRepr, defaultOptionValue = vpcName, *vpc.VpcId
			}

			data = append(data, []string{fmt.Sprintf("%d.", index+1), vpcName, *vpc.CidrBlock})
		}
	}

	indexedOptions = append(indexedOptions, cli.ResponseNew)
	data = append(data, []string{fmt.Sprintf("%d.", len(data)+1),
		fmt.Sprintf("Create new VPC with default CIDR and %d subnets", cfn.RequiredAvailabilityZones)})

	question := "VPC"
	optionsText := table.BuildTable(data, []string{"Option", "VPC", "CIDR Block"})

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: &defaultOptionRepr,
		DefaultOption:     &defaultOptionValue,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
	})

	return &answer, nil
}

// Ask the users to select a subnet
func AskSubnet(h *ec2helper.EC2Helper, vpcId string) (*string, error) {
	subnets, err := h.GetSubnetsByVpc(vpcId)
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}
	var defaultOptionRepr, defaultOptionValue *string = nil, nil

	// Add security groups to the data for table
	for index, subnet := range subnets {
		indexedOptions = append(indexedOptions, *subnet.SubnetId)

		subnetName := *subnet.SubnetId
		subnetTagName := ec2helper.GetTagName(subnet.Tags)
		if subnetTagName != nil {
			subnetName = fmt.Sprintf("%s(%s)", *subnetTagName, *subnet.SubnetId)
		}

		data = append(data, []string{fmt.Sprintf("%d.", index+1), subnetName, *subnet.AvailabilityZone,
			*subnet.CidrBlock})
	}

	defaultOptionRepr, defaultOptionValue = &data[0][1], subnets[0].SubnetId
	question := "Subnet"
	optionsText := table.BuildTable(data, []string{"Option", "Subnet", "Availability Zone", "CIDR Block"})

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: defaultOptionRepr,
		DefaultOption:     defaultOptionValue,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
	})

	return &answer, nil
}

// Ask the users to select a subnet placeholder
func AskSubnetPlaceholder(h *ec2helper.EC2Helper) (*string, error) {
	availabilityZones, err := h.GetAvailableAvailabilityZones()
	if err != nil {
		return nil, err
	}

	data := [][]string{}
	indexedOptions := []string{}

	// Add availability zones to the data for table
	for index, zone := range availabilityZones {
		indexedOptions = append(indexedOptions, *zone.ZoneName)

		data = append(data, []string{fmt.Sprintf("%d.", index+1), *zone.ZoneName, *zone.ZoneId})
	}
	defaultOptionRepr, defaultOptionValue := &data[0][1], &data[0][1]

	question := "Availability Zone"
	optionsText := table.BuildTable(data, []string{"Option", "Zone Name", "Zone ID"})

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: defaultOptionRepr,
		DefaultOption:     defaultOptionValue,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
	})

	return &answer, nil
}

// Ask the users to select security groups
func AskSecurityGroups(groups []*ec2.SecurityGroup, addedGroups []string) string {
	question := "Security Group(s)"
	data := [][]string{}
	indexedOptions := []string{}
	var defaultOptionRepr, defaultOptionValue *string = nil, nil

	// Add security groups to the data for table
	if groups != nil {
		counter := 0
		for _, group := range groups {
			// If this security group is already added, just don't display it here
			isFound := false
			for _, addedGroupId := range addedGroups {
				if addedGroupId == *group.GroupId {
					isFound = true
					break
				}
			}
			if isFound {
				continue
			}

			indexedOptions = append(indexedOptions, *group.GroupId)

			groupName := *group.GroupId
			groupTagName := ec2helper.GetTagName(group.Tags)
			if groupTagName != nil {
				groupName = fmt.Sprintf("%s(%s)", *groupTagName, *group.GroupId)
			}

			if *group.GroupName == "default" {
				defaultOptionRepr, defaultOptionValue = &groupName, group.GroupId
			}

			data = append(data, []string{fmt.Sprintf("%d.", counter+1), groupName,
				*group.Description})
			counter++
		}
	}

	// If no security group is available, simply don't ask
	if len(data) <= 0 {
		return cli.ResponseNo
	}

	// Add "add all" option
	indexedOptions = append(indexedOptions, cli.ResponseAll)
	data = append(data, []string{fmt.Sprintf("%d.", len(data)+1),
		"Add all available security groups"})

	// Add "new" option
	indexedOptions = append(indexedOptions, cli.ResponseNew)
	data = append(data, []string{fmt.Sprintf("%d.", len(data)+1),
		"Create a new security group that enables SSH"})

	// Add "done" option, if the added security group slice is not empty
	if len(addedGroups) > 0 {
		question = fmt.Sprintf("If you wish to add additional security group(s), add from the following:\nSecurity Group(s) already selected: %s", addedGroups)
		indexedOptions = append(indexedOptions, cli.ResponseNo)
		data = append(data, []string{fmt.Sprintf("%d.", len(data)+1),
			"Don't add any more security group"})
	}

	optionsText := table.BuildTable(data, []string{"Option", "Security Group", "Description"})

	answer := AskQuestion(&AskQuestionInput{
		QuestionString:    question,
		DefaultOptionRepr: defaultOptionRepr,
		DefaultOption:     defaultOptionValue,
		OptionsString:     &optionsText,
		IndexedOptions:    indexedOptions,
	})

	return answer
}

// Ask the users to select a security group placeholder
func AskSecurityGroupPlaceholder() string {
	data := [][]string{}
	indexedOptions := []string{}

	// Add the options
	indexedOptions = append(indexedOptions, cli.ResponseAll)
	indexedOptions = append(indexedOptions, cli.ResponseNew)
	data = append(data, []string{fmt.Sprintf("%d.", 1), "Use the default security group"})
	data = append(data, []string{fmt.Sprintf("%d.", 2), "Create and use a new security group for SSH"})

	question := "Security Group(s)"
	optionsText := table.BuildTable(data, []string{"Option", ""})

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return answer
}

// Print confirmation information for instance launch and ask for confirmation
func AskConfirmationWithTemplate(h *ec2helper.EC2Helper,
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

	stringOptions := []string{cli.ResponseYes, cli.ResponseNo}

	configText := table.BuildTable(data, nil)
	optionsText := yesNoOption + "\n"
	question := "Please confirm if you would like to launch instance with following options: \n" +
		configText

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		OptionsString:  &optionsText,
		StringOptions:  stringOptions,
	})

	return &answer, nil
}

// Print confirmation information for instance launch and ask for confirmation
func AskConfirmationWithInput(simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo,
	allowEdit bool) string {
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

	indexedOptions := []string{}
	stringOptions := []string{cli.ResponseYes, cli.ResponseNo}

	/*
		Append all security groups.
		If security groups were successfully parsed into the detailed config, append them here.
		Otherwise, look for placeholder security groups, such as "all" and "new" in the simple config.
		Also, use the bool value to tell the next block what question option to use for security group
	*/
	if detailedConfig.SecurityGroups != nil {
		data = table.AppendSecurityGroups(data, detailedConfig.SecurityGroups)
	} else if simpleConfig.SecurityGroupIds != nil && len(simpleConfig.SecurityGroupIds) >= 1 {
		if simpleConfig.SecurityGroupIds[0] == cli.ResponseNew {
			data = append(data, []string{cli.ResourceSecurityGroup, "New security group for SSH"})
		} else if simpleConfig.SecurityGroupIds[0] == cli.ResponseAll {
			data = append(data, []string{cli.ResourceSecurityGroup, "New default security group"})
		}
	}

	if ec2helper.HasEbsVolume(detailedConfig.Image) {
		data = append(data, []string{cli.ResourceKeepEbsVolume,
			strconv.FormatBool(simpleConfig.KeepEbsVolumeAfterTermination)})
	}

	if detailedConfig.Image.PlatformDetails != nil &&
		ec2helper.IsLinux(*detailedConfig.Image.PlatformDetails) {
		if simpleConfig.AutoTerminationTimerMinutes > 0 {
			data = append(data, []string{cli.ResourceAutoTerminationTimer,
				strconv.Itoa(simpleConfig.AutoTerminationTimerMinutes)})
		} else {
			data = append(data, []string{cli.ResourceAutoTerminationTimer, "None"})
		}
	}

	// If edit is allowed, give all items a number and fill the indexed options
	if allowEdit {
		for i := 0; i < len(data); i++ {
			// Skip region
			if data[i][0] == cli.ResourceRegion {
				continue
			}

			/*
				Only add an option number for rows that has a value in the first column,
				because some rows are subrows
			*/
			if data[i][0] != "" {
				/*
					If the row is for placeholder security group or placeholder subnet,
					append a placeholder option.
					Otherwise, append the first column of the row as option
				*/
				if simpleConfig.NewVPC {
					if data[i][0] == cli.ResourceSecurityGroup {
						indexedOptions = append(indexedOptions, cli.ResourceSecurityGroupPlaceholder)
					} else if data[i][0] == cli.ResourceSubnet {
						indexedOptions = append(indexedOptions, cli.ResourceSubnetPlaceholder)
					} else {
						indexedOptions = append(indexedOptions, data[i][0])
					}
				} else {
					indexedOptions = append(indexedOptions, data[i][0])
				}
				data[i][0] = fmt.Sprintf("%s", data[i][0])
			}
		}
	}

	// Append all EBS blocks, if applicable
	blockDeviceMappings := detailedConfig.Image.BlockDeviceMappings
	data = table.AppendEbs(data, blockDeviceMappings)

	// Append instance store, if applicable
	if detailedConfig.InstanceTypeInfo.InstanceStorageInfo != nil {
		data = append(data, []string{"Instance Storage", fmt.Sprintf("%d GB",
			*detailedConfig.InstanceTypeInfo.InstanceStorageInfo.TotalSizeInGB)})
	}

	// Append instance profile, if applicable
	if simpleConfig.IamInstanceProfile != "" {
		data = append(data, []string{cli.ResourceIamInstanceProfile, simpleConfig.IamInstanceProfile})
	}

	if simpleConfig.BootScriptFilePath != "" {
		data = append(data, []string{cli.ResourceBootScriptFilePath, simpleConfig.BootScriptFilePath})
	}
	if len(simpleConfig.UserTags) != 0 {
		var tags []string
		for k, v := range simpleConfig.UserTags {
			tags = append(tags, fmt.Sprintf("%s|%s", k, v))
		}
		data = append(data, []string{cli.ResourceUserTags, strings.Join(tags, "\n")})
	}

	configText := table.BuildTable(data, nil)

	optionsText := configText + yesNoOption + "\n"
	question := "Please confirm if you would like to launch instance with following options"

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
		StringOptions:  stringOptions,
	})

	return answer
}

// Ask if the user wants to save the config as a JSON config file
func AskSaveConfig() string {
	stringOptions := []string{cli.ResponseYes, cli.ResponseNo}

	optionsText := yesNoOption + "\n"
	question := "Do you want to save the configuration above as a JSON file that can be used in non-interactive mode? "

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  aws.String(cli.ResponseNo),
		OptionsString:  &optionsText,
		StringOptions:  stringOptions,
	})

	return answer
}

// Ask the instance id to be connected
func AskInstanceId(h *ec2helper.EC2Helper) (*string, error) {
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

	data, indexedOptions, _ = table.AppendInstances(data, indexedOptions, instances, nil)

	optionsText := table.BuildTable(data, []string{"Option", "Instance", "Tag-Key", "Tag-Value"})
	question := "Select the instance you want to connect to: "

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return &answer, nil
}

// Ask the instance IDs to be terminated
func AskInstanceIds(h *ec2helper.EC2Helper, addedInstanceIds []string) (*string, error) {
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

	data, indexedOptions, finalCounter := table.AppendInstances(data, indexedOptions, instances,
		addedInstanceIds)

	// There are no instances available for termination in selected region
	if len(data) <= 0 && len(addedInstanceIds) == 0 {
		return nil, errors.New("No instance available in selected region for termination")
	}

	// Since no more instance(s) are available for termination, proceed with current selection
	if len(data) == 0 && len(addedInstanceIds) > 0 {
		return nil, nil
	}

	// Add "done" option, if instance(s) are already selected
	if len(addedInstanceIds) > 0 {
		indexedOptions = append(indexedOptions, cli.ResponseNo)
		data = append(data, []string{fmt.Sprintf("%d.", finalCounter+1),
			"Don't add any more instance id"})
	}

	optionsText := table.BuildTable(data, []string{"Option", "Instance", "Tag-Key", "Tag-Value"})
	question := "Select the instance you want to terminate: "
	if len(addedInstanceIds) > 0 {
		question = "If you wish to terminate multiple instance(s), add from the following: "
	}

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return &answer, err
}

// AskBootScript prompts the user for a filepath to an optional boot script
func AskBootScript(h *ec2helper.EC2Helper) string {
	question := "Add filepath to instance boot script? " + "\n" + "format: absolute file path"
	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  aws.String(cli.ResponseNo),
		EC2Helper:      h,
		Fns:            []CheckInput{ec2helper.ValidateFilepath},
	})
	return answer
}

// AskUserTags prompts the user for optional tags
func AskUserTags(h *ec2helper.EC2Helper) string {
	question := "Add tags to instances and persisted volumes? " + "\n" + "format: tag1|val1,tag2|val2"
	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  aws.String(cli.ResponseNo),
		EC2Helper:      h,
		Fns:            []CheckInput{ec2helper.ValidateTags},
	})
	return answer
}

// AskTerminationConfirmation confirms if the user wants to terminate the selected instanceIds
func AskTerminationConfirmation(instanceIds []string) string {
	stringOptions := []string{cli.ResponseYes, cli.ResponseNo}

	optionsText := yesNoOption + "\n"
	question := fmt.Sprintf("Are you sure you want to terminate %d instance(s): %s ", len(instanceIds), instanceIds)

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  aws.String(cli.ResponseNo),
		OptionsString:  &optionsText,
		StringOptions:  stringOptions,
	})

	return answer
}

func AskCapacityType() string {
	question := fmt.Sprintf("Select capacity type. Spot instances are available at up to a 90%% discount compared to On-Demand instances,\nbut they may get interrupted by EC2 with a 2-minute warning")
	defaultInstanceTypeText := "On-Demand"
	optionsText := "1. On-Demand\n2. Spot\n"
	indexedOptions := []string{"On-Demand", "Spot"}

	answer := AskQuestion(&AskQuestionInput{
		QuestionString: question,
		DefaultOption:  &defaultInstanceTypeText,
		OptionsString:  &optionsText,
		IndexedOptions: indexedOptions,
	})

	return answer
}
