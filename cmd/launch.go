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

package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/iamhelper"
	"simple-ec2/pkg/question"
	"simple-ec2/pkg/questionModel"

	"github.com/aws/amazon-ec2-instance-selector/v2/pkg/selector"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch an Amazon EC2 instance",
	Long: "Launch an Amazon EC2 instance with the default configurations. " +
		"All configurations can be overridden by configurations provided by configuration files or user input.",
	Run: launch,
}

// Add flags
func init() {
	rootCmd.AddCommand(launchCmd)
	launchCmd.Flags().BoolVarP(&isInteractive, "interactive", "i", false, "Interactive mode")
	launchCmd.Flags().StringVarP(&flagConfig.Region, "region", "r", "",
		"The region where the instance will be launched")
	launchCmd.Flags().StringVarP(&flagConfig.InstanceType, "instance-type", "t", "",
		"The instance type of the instance")
	launchCmd.Flags().StringVarP(&flagConfig.ImageId, "image-id", "m", "",
		"The image id of the AMI used to launch the instance")
	launchCmd.Flags().StringVarP(&flagConfig.SubnetId, "subnet-id", "s", "",
		"The subnet id in which the instance will be launched")
	launchCmd.Flags().StringVarP(&flagConfig.LaunchTemplateId, "launch-template-id", "l", "",
		"The launch template id with which the instance will be launched")
	launchCmd.Flags().StringVarP(&flagConfig.LaunchTemplateVersion, "launch-template-version", "v", "",
		"The launch template version with which the instance will be launched")
	launchCmd.Flags().StringSliceVarP(&flagConfig.SecurityGroupIds, "security-group-ids", "g", nil,
		"The security groups with which the instance will be launched")
	launchCmd.Flags().BoolVarP(&isSaveConfig, "save-config", "c", false, "Save config as a JSON config file")
	launchCmd.Flags().BoolVarP(&flagConfig.KeepEbsVolumeAfterTermination, "keep-ebs", "k", false,
		"Keep EBS volumes after instance termination")
	launchCmd.Flags().IntVarP(&flagConfig.AutoTerminationTimerMinutes, "auto-termination-timer", "a", 0,
		"The auto-termination timer for the instance in minutes")
	launchCmd.Flags().StringVarP(&flagConfig.IamInstanceProfile, "iam-instance-profile", "p", "",
		"The profile containing an IAM role to attach to the instance")
	launchCmd.Flags().StringVarP(&flagConfig.BootScriptFilePath, "boot-script", "b", "",
		"The absolute filepath to a bash script passed to the instance and executed after the instance starts (user data)")
	launchCmd.Flags().StringToStringVar(&flagConfig.UserTags, "tags", nil,
		"The tags applied to instances and volumes at launch (Example: tag1=val1,tag2=val2)")
	launchCmd.Flags().StringVar(&flagConfig.CapacityType, "capacity-type", "",
		fmt.Sprintf("Launch instance as \"%s\" (the default) or \"%s\"", question.DefaultCapacityTypeText.OnDemand, question.DefaultCapacityTypeText.Spot))
}

// The main function
func launch(cmd *cobra.Command, args []string) {
	if !ValidateLaunchFlags(flagConfig) {
		return
	}

	// Start a new session, with the default credentials and config loading
	sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
	ec2helper.GetDefaultRegion(sess)
	h := ec2helper.New(sess)
	qh := questionModel.NewQuestionModelHelper()

	if isInteractive {
		launchInteractive(h, qh)
	} else {
		launchNonInteractive(h, qh)
	}
}

// Launch the instance interactively
func launchInteractive(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper) {
	simpleConfig := config.NewSimpleInfo()

	// Override config with flags if applicable
	config.OverrideConfigWithFlags(simpleConfig, flagConfig)

	simpleDefaultsConfig := config.NewSimpleInfo()
	err := config.ReadConfig(simpleDefaultsConfig, nil)
	if cli.ShowError(err, "Default config file not loaded; using system defaults instead") {
		simpleDefaultsConfig = config.NewSimpleInfo()
	}

	if simpleConfig.Region == "" {
		// Ask Region
		region, err := question.AskRegion(h, qh, simpleDefaultsConfig.Region)
		if cli.ShowError(err, "Asking region failed") {
			return
		}
		simpleConfig.Region = *region
	}

	h.ChangeRegion(simpleConfig.Region)

	detailedDefaultsConfig, err := h.ParseConfig(simpleDefaultsConfig)

	// Ask Launch Template
	launchTemplateId := &simpleConfig.LaunchTemplateId
	if simpleConfig.LaunchTemplateId == "" {
		launchTemplateId, err = question.AskLaunchTemplate(h, qh, simpleDefaultsConfig.LaunchTemplateId)
		if err != nil {
			return
		}
	}

	if *launchTemplateId != cli.ResponseNo {
		// Use a launch template in this case.
		simpleConfig.LaunchTemplateId = *launchTemplateId
		UseLaunchTemplate(h, qh, simpleConfig, simpleDefaultsConfig)
		return
	}

	// Not using a launch template if the program is not terminated at the point
	if simpleConfig.InstanceType == "" && !ReadInstanceType(h, qh, simpleConfig, simpleDefaultsConfig.InstanceType) {
		return
	}

	// Ask for image ID, auto-termination timer, and keeping EBS volumes after instance termination
	if simpleConfig.ImageId == "" && !ReadImageId(h, qh, simpleConfig, simpleDefaultsConfig) {
		return
	}

	// Ask for network configuration
	if (simpleConfig.SubnetId == "" || simpleConfig.SecurityGroupIds == nil) &&
		!ReadNetworkConfiguration(h, qh, simpleConfig, detailedDefaultsConfig) {
		return
	}

	// Ask for IAM profile
	if simpleConfig.IamInstanceProfile == "" && !ReadIamProfile(h, qh, simpleConfig, simpleDefaultsConfig.IamInstanceProfile) {
		return
	}

	// Ask for user boot data
	if simpleConfig.BootScriptFilePath == "" {
		err := ReadBootScript(h, qh, simpleConfig, simpleDefaultsConfig.BootScriptFilePath)
		if err != nil {
			return
		}
	}

	// Ask for tags
	if len(simpleConfig.UserTags) == 0 {
		err := ReadUserTags(h, qh, simpleConfig, simpleDefaultsConfig.UserTags)
		if err != nil {
			return
		}
	}
	// Ask for and set the capacity type
	simpleConfig.CapacityType, err = question.AskCapacityType(qh, simpleConfig.InstanceType, simpleConfig.Region, simpleDefaultsConfig.CapacityType)
	if cli.ShowError(err, "Asking capacity type failed") {
		return
	}

	// Ask for confirmation or modification. Keep asking until the config is confirmed or denied
	var detailedConfig *config.DetailedInfo
	var confirmation string
	for {
		// Parse config first
		detailedConfig, err = h.ParseConfig(simpleConfig)
		if cli.ShowError(err, "Parsing config failed") {
			return
		}

		// Ask for confirmation or modification
		confirmation, err = question.AskConfirmationWithInput(qh, simpleConfig, detailedConfig, true)
		if cli.ShowError(err, "Asking configuration confirmation failed") {
			return
		}

		// The users have confirmed or denied the config
		if confirmation == cli.ResponseYes || confirmation == cli.ResponseNo {
			break
		}

		simpleDefaultsConfig = simpleConfig
		detailedDefaultsConfig, err = h.ParseConfig(simpleDefaultsConfig)

		switch confirmation {
		// Ask questions to modify the config
		case cli.ResourceVpc:
			if !ReadNetworkConfiguration(h, qh, simpleConfig, detailedDefaultsConfig) {
				return
			}
		case cli.ResourceSubnet:
			if !ReadSubnet(h, qh, simpleConfig, *detailedConfig.Subnet.VpcId, simpleDefaultsConfig.SubnetId) {
				return
			}
		case cli.ResourceSecurityGroup:
			if !ReadSecurityGroups(h, qh, simpleConfig, *detailedConfig.Subnet.VpcId, detailedDefaultsConfig.SecurityGroups) {
				return
			}
		case cli.ResourceInstanceType:
			if !ReadInstanceType(h, qh, simpleConfig, simpleDefaultsConfig.InstanceType) {
				return
			}
			if !ReadImageId(h, qh, simpleConfig, simpleDefaultsConfig) {
				return
			}
		case cli.ResourceImage:
			if !ReadImageId(h, qh, simpleConfig, simpleDefaultsConfig) {
				return
			}
		case cli.ResourceKeepEbsVolume:
			ebsVolumeAnswer, err := question.AskKeepEbsVolume(qh, simpleDefaultsConfig.KeepEbsVolumeAfterTermination)
			if cli.ShowError(err, "Asking EBS volume persistence failed") {
				return
			}
			ReadKeepEbsVolume(simpleConfig, ebsVolumeAnswer == cli.ResponseYes)
		case cli.ResourceAutoTerminationTimer:
			if !ReadAutoTerminationTimer(h, qh, simpleConfig, simpleDefaultsConfig.AutoTerminationTimerMinutes) {
				return
			}
		case cli.ResourceIamInstanceProfile:
			if !ReadIamProfile(h, qh, simpleConfig, simpleDefaultsConfig.IamInstanceProfile) {
				return
			}
		case cli.ResourceCapacityType:
			simpleConfig.CapacityType, err = question.AskCapacityType(qh, simpleConfig.InstanceType, simpleConfig.Region, simpleDefaultsConfig.CapacityType)
			if cli.ShowError(err, "Asking capacity type failed") {
				return
			}
		case cli.ResourceUserTags:
			err := ReadUserTags(h, qh, simpleConfig, simpleDefaultsConfig.UserTags)
			if err != nil {
				return
			}
		case cli.ResourceBootScriptFilePath:
			err := ReadBootScript(h, qh, simpleConfig, simpleDefaultsConfig.BootScriptFilePath)
			if err != nil {
				return
			}
		}
	}

	// Launch On-Demand or Spot instance based on capacity type
	err = LaunchCapacityInstance(h, simpleConfig, detailedConfig, confirmation)

	if cli.ShowError(err, "Launching instance failed") {
		return
	}
	ReadSaveConfig(qh, simpleConfig)
}

// Launch the instance non-interactively
func launchNonInteractive(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper) {
	simpleConfig := config.NewSimpleInfo()
	if flagConfig.Region != "" {
		simpleConfig.Region = flagConfig.Region
		h.ChangeRegion(simpleConfig.Region)
	}

	// Try to get config from the config file
	err := config.ReadConfig(simpleConfig, nil)
	if cli.ShowError(err, "Default config file not loaded; using system defaults instead") {
		// If getting config file fails, go for default values
		simpleConfig, err = h.GetDefaultSimpleConfig()
		if cli.ShowError(err, "Generating config failed") {
			return
		}
	}

	h.ChangeRegion(simpleConfig.Region)

	// Override config with flags if applicable
	config.OverrideConfigWithFlags(simpleConfig, flagConfig)

	// When the flags specify a launch template
	if flagConfig.LaunchTemplateId != "" {
		// If using a launch template, ignore the config file. Only read from the flags
		UseLaunchTemplateWithConfig(h, qh, flagConfig, simpleConfig.CapacityType)
		return
	}

	// When the config file specifies a launch template
	if simpleConfig.LaunchTemplateId != "" {
		UseLaunchTemplateWithConfig(h, qh, simpleConfig, simpleConfig.CapacityType)
		return
	}

	// Parse the simple string config to the detailed config with data structures for later use
	detailedConfig, err := h.ParseConfig(simpleConfig)
	if cli.ShowError(err, "Parsing config failed") {
		return
	}

	confirmation, err := question.AskConfirmationWithInput(qh, simpleConfig, detailedConfig, false)
	if cli.ShowError(err, "Asking configuration confirmation failed") {
		return
	}

	LaunchCapacityInstance(h, simpleConfig, detailedConfig, confirmation)

	if cli.ShowError(err, "Launching instance failed") {
		return
	}
	ReadSaveConfig(qh, simpleConfig)
}

// Launch On-Demand or Spot instance based on capacity type
func LaunchCapacityInstance(h *ec2helper.EC2Helper, simpleConfig *config.SimpleInfo, detailedConfig *config.DetailedInfo,
	confirmation string) error {
	var err error
	if simpleConfig.CapacityType == question.DefaultCapacityTypeText.OnDemand {
		_, err = h.LaunchInstance(simpleConfig, detailedConfig, confirmation == cli.ResponseYes)
	} else {
		err = h.LaunchSpotInstance(simpleConfig, detailedConfig, confirmation == cli.ResponseYes)
	}
	return err
}

// Validate flags using some simple rules. Return true if the flags are validated, false otherwise
func ValidateLaunchFlags(flags *config.SimpleInfo) bool {
	if flags.LaunchTemplateVersion != "" && flags.LaunchTemplateId == "" {
		fmt.Println("Error: You can't define the version without launch template")
		return false
	}
	if flags.BootScriptFilePath != "" {
		_, err := os.Stat(flags.BootScriptFilePath)
		if err != nil {
			fmt.Println("Error: Boot script file path invalid or does not exist")
			return false
		}
	}

	if flags.CapacityType != "" {
		if strings.ToLower(flags.CapacityType) == strings.ToLower(question.DefaultCapacityTypeText.OnDemand) {
			flags.CapacityType = question.DefaultCapacityTypeText.OnDemand
		} else if strings.ToLower(flags.CapacityType) == strings.ToLower(question.DefaultCapacityTypeText.Spot) {
			flags.CapacityType = question.DefaultCapacityTypeText.Spot
		} else {
			fmt.Printf("Error: Capacity type must be \"%s\" or \"%s\"\n", question.DefaultCapacityTypeText.OnDemand, question.DefaultCapacityTypeText.Spot)
			return false
		}
	}

	return true
}

// Ask for version and launch with the launch template.
func UseLaunchTemplate(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultsConfig *config.SimpleInfo) {
	// Ask Launch Template version, if not specified already
	if simpleConfig.LaunchTemplateVersion == "" {
		launchTemplateVersion, err := question.AskLaunchTemplateVersion(h, qh, simpleConfig.LaunchTemplateId, defaultsConfig.LaunchTemplateVersion)
		if cli.ShowError(err, "Asking launch template version failed") {
			return
		}
		simpleConfig.LaunchTemplateVersion = *launchTemplateVersion
	}

	LaunchWithLaunchTemplate(h, qh, simpleConfig, defaultsConfig.CapacityType)
}

// Use a launch template with config
func UseLaunchTemplateWithConfig(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultCapacityType string) {
	/*
		Deciding the version of the launch template. If no version is specified,
		use the default version.
	*/
	var launchTemplateVersion string
	if simpleConfig.LaunchTemplateVersion == "" {
		launchTemplate, err := h.GetLaunchTemplateById(simpleConfig.LaunchTemplateId)
		if cli.ShowError(err, "The specified launch template is not available") {
			return
		}
		launchTemplateVersion = strconv.FormatInt(*launchTemplate.DefaultVersionNumber, 10)
	} else {
		launchTemplateVersion = simpleConfig.LaunchTemplateVersion
	}
	simpleConfig.LaunchTemplateVersion = launchTemplateVersion

	LaunchWithLaunchTemplate(h, qh, simpleConfig, defaultCapacityType)
}

// Launch an instance with a launch template
func LaunchWithLaunchTemplate(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultCapacityType string) {
	versions, err := h.GetLaunchTemplateVersions(simpleConfig.LaunchTemplateId,
		&simpleConfig.LaunchTemplateVersion)
	templateData := versions[0].LaunchTemplateData
	simpleConfig.CapacityType, err = question.AskCapacityType(qh, *templateData.InstanceType, simpleConfig.Region, defaultCapacityType)
	if cli.ShowError(err, "Asking capacity type failed") {
		return
	}

	confirmation, err := question.AskConfirmationWithTemplate(h, qh, simpleConfig)
	if cli.ShowError(err, "Asking confirmation with launch template failed") {
		return
	}

	// Launch the instance.
	err = LaunchCapacityInstance(h, simpleConfig, nil, *confirmation)
	if cli.ShowError(err, "Launching instance failed") {
		return
	}
	ReadSaveConfig(qh, simpleConfig)
}

/*
Ask user input for an instance type, resource definition (using instance selector) or fall back to using default.
Return true if the function is executed successfully, false otherwise.
*/
func ReadInstanceType(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultInstanceType string) bool {
	// Ask if the users want to enter an instance type
	instanceTypeResponse, err := question.AskIfEnterInstanceType(h, qh, defaultInstanceType)
	if cli.ShowError(err, "Asking instance type failed") {
		return false
	}

	/*
		The users can input yes, which brings them to another question asking instance type
		The users can input no, which brings them to instance selector
		Otherwise, the default instance type must be the response and the value is taken
	*/
	var instanceType *string
	if *instanceTypeResponse == cli.ResponseYes {
		instanceType, err = question.AskInstanceType(h, qh, defaultInstanceType)
		if cli.ShowError(err, "Asking instance type failed") {
			return false
		}
	} else if *instanceTypeResponse == cli.ResponseNo {
		// Instantiate a new instance of a selector with the AWS session
		instanceSelector := selector.New(h.Sess)

		vcpus, err := question.AskInstanceTypeVCpu(h, qh)
		if cli.ShowError(err, "Asking vCPUs failed") {
			return false
		}

		memoryGib, err := question.AskInstanceTypeMemory(h, qh)
		if cli.ShowError(err, "Asking memory failed") {
			return false
		}

		instanceType, err = question.AskInstanceTypeInstanceSelector(h, qh, instanceSelector, vcpus, memoryGib)
		if cli.ShowError(err, "Asking instance type failed") {
			return false
		}
	} else {
		// The default instance type is used in this case
		instanceType = instanceTypeResponse
	}

	simpleConfig.InstanceType = *instanceType

	return true
}

/*
Ask user input for an image id. The user can select from provided options orenter a valid image id.
Return true if the function is executed successfully, false otherwise
*/
func ReadImageId(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultsConfig *config.SimpleInfo) bool {
	// Get the image ID
	image, err := question.AskImage(h, qh, simpleConfig.InstanceType, defaultsConfig.ImageId)
	if cli.ShowError(err, "Asking image failed") {
		return false
	}

	simpleConfig.ImageId = *image.ImageId

	if !simpleConfig.KeepEbsVolumeAfterTermination && ec2helper.HasEbsVolume(image) {
		ebsVolumeAnswer, err := question.AskKeepEbsVolume(qh, defaultsConfig.KeepEbsVolumeAfterTermination)
		if cli.ShowError(err, "Asking EBS volume persistence failed") {
			return false
		}
		ReadKeepEbsVolume(simpleConfig, ebsVolumeAnswer == cli.ResponseYes)
	}

	// Auto-termination only supports Linux for now
	if simpleConfig.AutoTerminationTimerMinutes == 0 && image.PlatformDetails != nil &&
		ec2helper.IsLinux(*image.PlatformDetails) {
		return ReadAutoTerminationTimer(h, qh, simpleConfig, defaultsConfig.AutoTerminationTimerMinutes)
	}

	return true
}

/*
Ask user input for the auto-termination timer.
Return true if the function is executed successfully, false otherwise
*/
func ReadAutoTerminationTimer(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultTimer int) bool {
	// Ask for auto-termination timer
	var timer int
	timerResponse, err := question.AskAutoTerminationTimerMinutes(h, qh, defaultTimer)
	if err == nil {
		timer, err = strconv.Atoi(timerResponse)
	}
	if cli.ShowError(err, "Asking auto-termination timer failed") {
		return false
	}
	simpleConfig.AutoTerminationTimerMinutes = timer
	return true
}

/*
Ask user input for keeping EBS volumes after instance termination.
Return true if the function is executed successfully, false otherwise
*/
func ReadKeepEbsVolume(simpleConfig *config.SimpleInfo, isKeepVolume bool) {
	simpleConfig.KeepEbsVolumeAfterTermination = isKeepVolume
}

/*
Ask user input for a network interface, including VPC, subnet and security groups.
The user can select from provided options or create new resources.
Return true if the function is executed successfully, false otherwise
*/
func ReadNetworkConfiguration(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultsConfig *config.DetailedInfo) bool {
	var defaultAzId, defaultSubnetId, defaultVpcId string
	defaultSecurityGroups := []*ec2.SecurityGroup{}
	if defaultsConfig != nil {
		if defaultsConfig.Subnet != nil {
			defaultAzId = *defaultsConfig.Subnet.AvailabilityZoneId
			defaultSubnetId = *defaultsConfig.Subnet.SubnetId
		}
		if defaultsConfig.Vpc != nil {
			defaultVpcId = *defaultsConfig.Vpc.VpcId
		}
		if defaultsConfig.SecurityGroups != nil {
			defaultSecurityGroups = defaultsConfig.SecurityGroups
		}
	}

	vpcId, err := question.AskVpc(h, qh, defaultVpcId)
	if cli.ShowError(err, "Asking VPC failed") {
		return false
	}

	/*
		When a new VPC will be created, ask for subnet and security group placeholders.
		Otherwise, proceed to subnet and security group selection
	*/
	if *vpcId == cli.ResponseNew {
		simpleConfig.NewVPC = true
		return ReadSubnetPlaceholder(h, qh, simpleConfig, defaultAzId) && ReadSecurityGroupPlaceholder(h, qh, simpleConfig)
	} else {
		// If the resources are not specified in the config, ask for them
		if (flagConfig.SubnetId == "" && !ReadSubnet(h, qh, simpleConfig, *vpcId, defaultSubnetId)) ||
			(flagConfig.SecurityGroupIds == nil && !ReadSecurityGroups(h, qh, simpleConfig, *vpcId, defaultSecurityGroups)) {
			return false
		}

		return true
	}
}

/*
Ask user input for subnet. The user can select from provided options.
Return true if the function is executed successfully, false otherwise
*/
func ReadSubnet(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, vpcId string, defaultSubnetId string) bool {
	// Ask for subnet
	subnetIdAnswer, err := question.AskSubnet(h, qh, vpcId, defaultSubnetId)
	if cli.ShowError(err, "Asking subnet failed") {
		return false
	}

	// the answer is a subnet id in this case
	simpleConfig.SubnetId = *subnetIdAnswer

	return true
}

/*
Ask user input for subnet placeholder. The user can select from provided options.
Return true if the function is executed successfully, false otherwise
*/
func ReadSubnetPlaceholder(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultAz string) bool {
	subnetPlaceholder, err := question.AskSubnetPlaceholder(h, qh, defaultAz)
	if cli.ShowError(err, "Asking availability zone failed") {
		return false
	}

	simpleConfig.SubnetId = *subnetPlaceholder

	return true
}

/*
Ask user input for security groups. The user can select from provided options or create new resources.
Return true if the function is executed successfully, false otherwise
*/
func ReadSecurityGroups(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, vpcId string, defaultSecurityGroups []*ec2.SecurityGroup) bool {
	retrievedGroups, err := h.GetSecurityGroupsByVpc(vpcId)
	if cli.ShowError(err, "Getting security groups in VPC failed") {
		return false
	}

	securityGroupAnswer, err := question.AskSecurityGroups(qh, retrievedGroups, defaultSecurityGroups)
	if cli.ShowError(err, "Asking Security Groups failed") {
		return false
	}

	// Create a new security group for SSH if the users selects "new"
	if slices.Contains(securityGroupAnswer, cli.ResponseNew) {
		newSecurityGroupId, err := h.CreateSecurityGroupForSsh(vpcId)
		if cli.ShowError(err, "Creating new security group for SSH failed") {
			return false
		}

		// Replace the "New" with the new security group Id
		for index, group := range securityGroupAnswer {
			if group == cli.ResponseNew {
				securityGroupAnswer[index] = *newSecurityGroupId
				break
			}
		}
	}

	simpleConfig.SecurityGroupIds = securityGroupAnswer
	return true
}

/*
Ask user input for security group placeholder.
The user can select from provided options or create new resources.
Return true if the function is executed successfully, false otherwise
*/
func ReadSecurityGroupPlaceholder(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo) bool {
	securityGroupPlaceholder, err := question.AskSecurityGroupPlaceholder(qh)
	if cli.ShowError(err, "Asking Security Groups failed") {
		return false
	}

	simpleConfig.SecurityGroupIds = []string{
		securityGroupPlaceholder,
	}
	return true
}

/*
Ask user input for IAM profile. The user can select from provided options.
Return true if the function is executed successfully, false otherwise
*/
func ReadIamProfile(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultIamProfile string) bool {
	// Ask for iam profile
	iam := iamhelper.New(h.Sess)
	iamAnswer, err := question.AskIamProfile(qh, iam, defaultIamProfile)
	if cli.ShowError(err, "Asking IAM failed") {
		return false
	}
	if iamAnswer != cli.ResponseNo {
		simpleConfig.IamInstanceProfile = iamAnswer
	} else {
		simpleConfig.IamInstanceProfile = ""
	}
	return true
}

/*
Ask user input for filepath containing boot script.
Return true if the function is executed successfully, false otherwise
*/
func ReadBootScript(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultBootScript string) error {
	confirmationAnswer, err := question.AskBootScriptConfirmation(h, qh, defaultBootScript)
	if cli.ShowError(err, "Asking boot script confirmation failed") {
		return err
	}

	if confirmationAnswer == cli.ResponseNo {
		return nil
	}

	bootScriptAnswer, err := question.AskBootScript(h, qh, defaultBootScript)
	if cli.ShowError(err, "Asking boot script failed") {
		return err
	}

	if bootScriptAnswer == "" || strings.ToLower(bootScriptAnswer) == strings.ToLower("None") {
		return nil
	}

	simpleConfig.BootScriptFilePath = bootScriptAnswer
	return nil
}

/*
Ask user input for tags applied to launched instances and volumes.
Return true if the function is executed successfully, false otherwise
*/
func ReadUserTags(h *ec2helper.EC2Helper, qh *questionModel.QuestionModelHelper,
	simpleConfig *config.SimpleInfo, defaultTags map[string]string) error {
	simpleConfig.UserTags = make(map[string]string)
	confirmationAnswer, err := question.AskUserTagsConfirmation(h, qh, defaultTags)
	if cli.ShowError(err, "Asking user tags confirmation failed") {
		return err
	}

	if confirmationAnswer == cli.ResponseNo {
		return nil
	}

	userTagsAnswer, err := question.AskUserTags(h, qh, defaultTags)
	if cli.ShowError(err, "Asking user tags failed") {
		return err
	}

	if userTagsAnswer == "" {
		return nil
	}

	//convert user input tag1|val1,tag2|val2 to map
	tags := strings.Split(userTagsAnswer, ",") //[tag1|val1, tag2|val2]
	for _, tag := range tags {
		kv := strings.Split(tag, "|") //[tag1, val1]
		simpleConfig.UserTags[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return nil
}

/*
Ask user input for config saving.
If the user chooses to save the config, save the config as a JSON config file.
*/
func ReadSaveConfig(qh *questionModel.QuestionModelHelper, simpleConfig *config.SimpleInfo) {
	isSaveRequired := isSaveConfig
	if !isSaveRequired && isInteractive {
		// Ask if the user wants to save the config. If so, save the config
		answer, err := question.AskSaveConfig(qh)
		if cli.ShowError(err, "Asking save configurations failed") {
			return
		}
		isSaveRequired = answer == cli.ResponseYes
	}

	if isSaveRequired {
		err := config.SaveConfig(simpleConfig, nil)
		cli.ShowError(err, "Saving config file failed")
	}
}
