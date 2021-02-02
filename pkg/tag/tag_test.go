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

	"simple-ec2/pkg/tag"
)

const createTimeRegex = "^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2} [0-9]{1,2}:[0-9]{1,2}:[0-9]{1,2} [A-Z]{3}"

func TestGetTags(t *testing.T) {
	tags := tag.GetSimpleEc2Tags()

	createdBy, found := (*tags)["CreatedBy"]
	if !found || createdBy != "simple-ec2" {
		t.Error("CreatedBy tag is not created correctly")
	}

	createdTime, found := (*tags)["CreatedTime"]
	matched, _ := regexp.MatchString(createTimeRegex, createdTime)
	if !matched {
		t.Error("CreatedTime tag was not created/formatted correctly")
	}
	if !found {
		t.Error("CreatedTime tag is not created correctly")
	}
}

func TestGetTagAsFilter(t *testing.T) {
	tags := tag.GetSimpleEc2Tags()
	actualTagFilter, err := tag.GetTagAsFilter(*tags)
	if err != nil {
		t.Error("GetTagAsFilter encountered an error: " + err.Error())
	}

	if len(actualTagFilter) != 2 {
		t.Error("TagFilters length should be 2 but is " + fmt.Sprint(len(actualTagFilter)))
	}
	for _, tfilter := range actualTagFilter {
		if *tfilter.Name != "tag:CreatedBy" && *tfilter.Name != "tag:CreatedTime" {
			t.Error("TagFilters does not contain tag:CreatedBy or tag:CreatedTime")
		}
		if *tfilter.Name == "tag:CreatedBy" && *tfilter.Values[0] != "simple-ec2" {
			t.Error("CreatedBy tag was not converted to filter correctly")
		}
		if *tfilter.Name == "tag:CreatedTime" {
			matched, _ := regexp.MatchString(createTimeRegex, *tfilter.Values[0])
			if !matched {
				t.Error("CreatedTime tag was not converted to filter correctly")
			}
		}
	}
}
