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
	"testing"

	"simple-ec2/pkg/tag"
)

func TestGetTags(t *testing.T) {
	tags := tag.GetSimpleEc2Tags()

	createdBy, found := (*tags)["CreatedBy"]
	if !found || createdBy != "simple-ec2" {
		t.Error("CreatedBy tag is not created correctly")
	}

	_, found = (*tags)["CreatedTime"]
	if !found {
		t.Error("CreatedTime tag is not created correctly")
	}
}
