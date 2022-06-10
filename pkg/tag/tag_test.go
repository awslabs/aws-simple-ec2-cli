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

package tag_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"simple-ec2/pkg/tag"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var expectedTagAsFilter = []*ec2.Filter{
	{
		Name:   aws.String("tag:CreatedBy"),
		Values: aws.StringSlice([]string{"simple-ec2"}),
	},
	{
		Name:   aws.String("tag:CreatedTime"),
		Values: aws.StringSlice([]string{""}),
	},
}

const createTimeRegex = "^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2} [0-9]{1,2}:[0-9]{1,2}:[0-9]{1,2} [A-Z]{3}"

func TestGetTags(t *testing.T) {
	tags := tag.GetSimpleEc2Tags()
	createdBy, createdByFound := (*tags)["CreatedBy"]
	createdTime, createdTimeFound := (*tags)["CreatedTime"]
	matched, _ := regexp.MatchString(createTimeRegex, createdTime)

	th.Assert(t, createdByFound, "CreatedBy tag is not created correctly")
	th.Assert(t, createdBy == "simple-ec2", "CreatedBy tag is not created correctly")
	th.Assert(t, createdTimeFound, "CreatedTime tag is not created correctly")
	th.Assert(t, matched, "CreatedTime tag was not created/formatted correctly")
}

func TestGetTagAsFilter(t *testing.T) {
	tags := tag.GetSimpleEc2Tags()
	actualTagFilter, err := tag.GetTagAsFilter(*tags)
	now := time.Now()
	zone, _ := now.Zone()
	currTimeStr := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d %s", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(),
		now.Second(), zone)
	// assign current time to avoid manually traversing map and checking time format
	expectedTagAsFilter[1].Values = aws.StringSlice([]string{currTimeStr})

	th.Ok(t, err)
	th.Assert(t, len(actualTagFilter) == 2, "TagFilters length should be 2")
	th.Equals(t, expectedTagAsFilter, actualTagFilter)
}
