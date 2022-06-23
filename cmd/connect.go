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
	ec2ichelper "simple-ec2/pkg/ec2instanceconnecthelper"
	"simple-ec2/pkg/question"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an Amazon EC2 Instance",
	Long:  `Connect to an Amazon EC2 Instance, given the region and instance id`,
	Run:   connect,
}

// Add flags
func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().StringVarP(&regionFlag, "region", "r", "",
		"The region in which the instance you want to connect locates")
	connectCmd.Flags().StringVarP(&instanceIdConnectFlag, "instance-id", "n", "",
		"The instance id of the instance you want to connect to")
	connectCmd.Flags().BoolVarP(&isInteractive, "interactive", "i", false, "Interactive mode")
}

// The main function
func connect(cmd *cobra.Command, args []string) {
	if !ValidateConnectFlags() {
		return
	}

	// Start a new session, with the default credentials and config loading
	sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
	ec2helper.GetDefaultRegion(sess)
	h := ec2helper.New(sess)

	if isInteractive {
		connectInteractive(h)
	} else {
		connectNonInteractive(h)
	}
}

// Connect instances interactively
func connectInteractive(h *ec2helper.EC2Helper) {
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

	// Ask instance ID
	instanceId, err := question.AskInstanceId(h)
	if cli.ShowError(err, "Asking instance ID failed") {
		return
	}

	err = GetInstanceAndConnect(h, *instanceId)
	if cli.ShowError(err, "Connecting to instance failed") {
		return
	}
}

// Connect instances non-interactively
func connectNonInteractive(h *ec2helper.EC2Helper) {
	// Override region if specified
	if regionFlag != "" {
		h.ChangeRegion(regionFlag)
	}

	// Trim leading and trailing whitespace of the instance id
	instanceIdConnectFlag = strings.TrimSpace(instanceIdConnectFlag)

	err := GetInstanceAndConnect(h, instanceIdConnectFlag)
	if cli.ShowError(err, "Connecting to instance failed") {
		return
	}
}

// Validate flags using simple rules. Return true if the flags are validated, false otherwise
func ValidateConnectFlags() bool {
	if !isInteractive {
		if instanceIdConnectFlag == "" && regionFlag == "" {
			fmt.Println("Not in interactive mode and no flag is specified")
			return false
		}
		if instanceIdConnectFlag == "" {
			fmt.Println("Not in interactive mode and instance id is not specified")
			return false
		}
	}

	return true
}

// Get the information of the instance and connect to it
func GetInstanceAndConnect(h *ec2helper.EC2Helper, instanceId string) error {
	instance, err := h.GetInstanceById(instanceId)
	if err != nil {
		return err
	}

	err = ec2ichelper.ConnectInstance(h.Sess, instance, false)
	if err != nil {
		return err
	}

	return nil
}
