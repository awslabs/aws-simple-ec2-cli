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
	"strings"

	"simple-ec2/pkg/cli"
	"simple-ec2/pkg/config"
	"simple-ec2/pkg/ec2helper"
	"simple-ec2/pkg/question"
	"simple-ec2/pkg/tag"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
)

// terminateCmd represents the terminate command
var terminateCmd = &cobra.Command{
	Use:   "terminate",
	Short: "Terminate Amazon EC2 Instances",
	Long:  `Terminate Amazon EC2 Instances, given the region and instance ids or tag values`,
	Run:   terminate,
}

// Add flags
func init() {
	rootCmd.AddCommand(terminateCmd)

	terminateCmd.Flags().StringVarP(&regionFlag, "region", "r", "",
		"The region in which the instances you want to terminate locates")
	terminateCmd.Flags().StringSliceVarP(&instanceIdFlag, "instance-ids", "n", nil,
		"The instance ids of the instances you want to terminate")
	terminateCmd.Flags().BoolVarP(&isInteractive, "interactive", "i", false, "Interactive mode")
	terminateCmd.Flags().StringToStringVar(&flagConfig.UserTags, "tags", nil,
		"Terminate instances containing EXACT tag key-pair (Example: CreatedBy=simple-ec2)")
}

// The main function
func terminate(cmd *cobra.Command, args []string) {
	if !ValidateTerminateFlags() {
		return
	}

	// Start a new session, with the default credentials and config loading
	sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
	ec2helper.GetDefaultRegion(sess)
	h := ec2helper.New(sess)

	if isInteractive {
		terminateInteractive(h)
	} else {
		terminateNonInteractive(h)
	}
}

// Terminate instances interactively
func terminateInteractive(h *ec2helper.EC2Helper) {
	// If region is not specified in flags, ask region
	var region *string
	var err error
	if regionFlag == "" {
		defaultsConfig := config.NewSimpleInfo()
		err = config.ReadConfig(defaultsConfig, nil)
		if err != nil {
			defaultsConfig = config.NewSimpleInfo()
		}
		region, err = question.AskRegion(h, defaultsConfig.Region)
		if cli.ShowError(err, "Asking region failed") {
			return
		}
	} else {
		region = &regionFlag
	}

	h.ChangeRegion(*region)

	// Keep asking for instance ids for termination
	instanceIds := []string{}
	for {
		// Ask instance ID
		instanceIdAnswer, err := question.AskInstanceIds(h, instanceIds)
		if cli.ShowError(err, "Terminate Error") {
			return
		}

		if instanceIdAnswer == nil || *instanceIdAnswer == cli.ResponseNo {
			break
		} else {
			instanceIds = append(instanceIds, *instanceIdAnswer)
		}
	}

	confirmationAnswer, err := question.AskTerminationConfirmation(instanceIds)
	if cli.ShowError(err, "Asking termination confirmation failed") {
		return
	}

	if confirmationAnswer == cli.ResponseYes {
		cli.ShowError(h.TerminateInstances(instanceIds), "Terminating instances failed")
	}
}

// Terminate instances non-interactively
func terminateNonInteractive(h *ec2helper.EC2Helper) {
	// Override region if specified
	if regionFlag != "" {
		h.ChangeRegion(regionFlag)
	}

	// Trim leading and trailing whitespace of the instance ids
	for i := 0; i < len(instanceIdFlag); i++ {
		instanceIdFlag[i] = strings.TrimSpace(instanceIdFlag[i])
	}

	instFilters, err := tag.GetTagAsFilter(flagConfig.UserTags)
	instancesToTerm, err := h.GetInstancesByFilter(instanceIdFlag, instFilters)
	if err != nil {
		cli.ShowError(err, "Finding instances with filters failed")
		return
	}

	err = h.TerminateInstances(instancesToTerm)
	if err != nil {
		cli.ShowError(err, "Terminating instances failed")
	}
}

// Validate flags using some simple rules. Return true if the flags are validated, false otherwise
func ValidateTerminateFlags() bool {
	if !isInteractive && instanceIdFlag == nil && len(flagConfig.UserTags) == 0 {
		fmt.Println("Specify instanceIds, tags, or use interactive mode")
		return false
	}
	return true
}
