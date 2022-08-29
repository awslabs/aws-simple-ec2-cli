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

package cli

import (
	"fmt"
)

// Enum values for response messages
const (
	ResponseYes = "Yes"
	ResponseNo  = "No"
	ResponseNew = "New"
	ResponseAll = "All"
)

// Enum values for displaying resource types in CLI
const (
	ResourceRegion                   = "Region"
	ResourceVpc                      = "VPC"
	ResourceSubnet                   = "Subnet"
	ResourceSubnetPlaceholder        = "Subnet Placeholder"
	ResourceInstanceType             = "Instance Type"
	ResourceImage                    = "Image"
	ResourceAutoTerminationTimer     = "Auto Termination Timer in Minutes"
	ResourceKeepEbsVolume            = "Keep EBS Volume(s) After Termination"
	ResourceSecurityGroup            = "Security Group"
	ResourceSecurityGroupPlaceholder = "Security Group Placeholder"
	ResourceIamInstanceProfile       = "IAM Instance Profile"
	ResourceBootScriptFilePath       = "Boot Script Filepath"
	ResourceUserTags                 = "Tag Specification(key|value)"
	ResourceCapacityType             = "Capacity Type"
)

// Show errors if there are any. Return true when there are errors, and false when there is none
func ShowError(err error, message string) bool {
	if err != nil {
		fmt.Println(message+":", err)
		return true
	}
	return false
}
