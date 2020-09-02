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

package tag

import (
	"fmt"
	"time"
)

// Get the tags for resources created by simple-ec2
func GetTags() *map[string]string {
	now := time.Now()
	zone, _ := now.Zone()
	nowString := fmt.Sprintf("%d-%d-%d %d:%d:%d %s", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(),
		now.Second(), zone)

	tags := map[string]string{
		"CreatedBy":   "simple-ec2",
		"CreatedTime": nowString,
	}

	return &tags
}
