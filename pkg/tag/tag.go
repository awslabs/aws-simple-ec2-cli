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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Get the tags for resources created by simple-ec2
func GetSimpleEc2Tags() *map[string]string {
	now := time.Now()
	zone, _ := now.Zone()
	nowString := fmt.Sprintf("%02d-%02d-%02d %02d:%02d:%02d %s", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(),
		now.Second(), zone)

	tags := map[string]string{
		"CreatedBy":   "simple-ec2",
		"CreatedTime": nowString,
	}
	return &tags
}

// Convert tag map to Filter
func GetTagAsFilter(userTags map[string]string) (filters []*ec2.Filter, err error) {
	for k, v := range userTags {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("tag:" + k), //prepend tag: for exact matching
			Values: aws.StringSlice([]string{v}),
		})
	}
	return filters, nil
}
