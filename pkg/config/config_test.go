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

package config_test

import (
	"io/ioutil"
	"os"
	"testing"

	"simple-ec2/pkg/config"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
)

const testConfigFileName = "unit_test_config_temp.json"

var testConfigFilePath = os.Getenv("HOME") + "/.simple-ec2/" + testConfigFileName

func TestSaveInConfigFolder(t *testing.T) {
	testString := "unit test config"
	testData := []byte(testString)

	path, err := config.SaveInConfigFolder(testConfigFileName, testData, 0644)
	defer os.Remove(*path)
	th.Ok(t, err)
	th.Assert(t, testConfigFilePath == *path, "The config file path is incorrect")

	// Check the content of the file is correct
	readData, err := ioutil.ReadFile(testConfigFilePath)
	th.Ok(t, err)
	th.Equals(t, testData, readData)
}

// This is merely a test config to test if functions work. It won't work in the real environment.
const testRegion = "us-somewhere"
const testImageId = "ami-12345"
const testInstanceType = "t2.micro"
const testSubnetId = "s-12345"
const testLaunchTemplateId = "lt-12345"
const testLaunchTemplateVersion = "1"
const testNewVPC = true
const testIamProfile = "iam-profile"
const testBootScriptFilePath = "some/path/to/bootscript"
const testCapacityType = "On-Spot-Demand"

var testTags = map[string]string{"testedBy": "BRYAN", "brokenBy": "CBASKIN"}
var testSecurityGroup = []string{"sg-12345", "sg-67890"}

const expectedJson = `{"Region":"us-somewhere","ImageId":"ami-12345","InstanceType":"t2.micro","SubnetId":"s-12345","LaunchTemplateId":"lt-12345","LaunchTemplateVersion":"1","SecurityGroupIds":["sg-12345","sg-67890"],"NewVPC":true,"AutoTerminationTimerMinutes":0,"KeepEbsVolumeAfterTermination":false,"IamInstanceProfile":"iam-profile","BootScriptFilePath":"some/path/to/bootscript","UserTags":{"brokenBy":"CBASKIN","testedBy":"BRYAN"},"CapacityType":"On-Spot-Demand"}`

func TestSaveConfig(t *testing.T) {
	testConfig := &config.SimpleInfo{
		Region:                testRegion,
		ImageId:               testImageId,
		InstanceType:          testInstanceType,
		SubnetId:              testSubnetId,
		LaunchTemplateId:      testLaunchTemplateId,
		LaunchTemplateVersion: testLaunchTemplateVersion,
		SecurityGroupIds:      testSecurityGroup,
		NewVPC:                testNewVPC,
		IamInstanceProfile:    testIamProfile,
		BootScriptFilePath:    testBootScriptFilePath,
		UserTags:              testTags,
		CapacityType:          testCapacityType,
	}

	err := config.SaveConfig(testConfig, aws.String(testConfigFileName))
	defer os.Remove(testConfigFilePath)
	th.Ok(t, err)

	// Check the content of the file is correct
	readData, err := ioutil.ReadFile(testConfigFilePath)
	th.Ok(t, err)
	th.Equals(t, expectedJson, string(readData))
}

func TestOverrideConfigWithFlags(t *testing.T) {
	actualConfig := config.NewSimpleInfo()
	expectedConfig := &config.SimpleInfo{
		Region:                testRegion,
		ImageId:               testImageId,
		InstanceType:          testInstanceType,
		SubnetId:              testSubnetId,
		LaunchTemplateId:      testLaunchTemplateId,
		LaunchTemplateVersion: testLaunchTemplateVersion,
		SecurityGroupIds:      testSecurityGroup,
		NewVPC:                testNewVPC,
		IamInstanceProfile:    testIamProfile,
		BootScriptFilePath:    testBootScriptFilePath,
		UserTags:              testTags,
	}
	config.OverrideConfigWithFlags(actualConfig, expectedConfig)
	th.Equals(t, expectedConfig, actualConfig)
}

func TestReadConfig(t *testing.T) {
	err := ioutil.WriteFile(testConfigFilePath, []byte(expectedJson), 0644)
	defer os.Remove(testConfigFilePath)

	th.Ok(t, err)

	actualConfig := config.NewSimpleInfo()
	err = config.ReadConfig(actualConfig, aws.String(testConfigFileName))
	th.Ok(t, err)

	// Check if the config is read correctly
	expectedConfig := &config.SimpleInfo{
		Region:                testRegion,
		ImageId:               testImageId,
		InstanceType:          testInstanceType,
		SubnetId:              testSubnetId,
		LaunchTemplateId:      testLaunchTemplateId,
		LaunchTemplateVersion: testLaunchTemplateVersion,
		SecurityGroupIds:      testSecurityGroup,
		NewVPC:                testNewVPC,
		IamInstanceProfile:    testIamProfile,
		BootScriptFilePath:    testBootScriptFilePath,
		UserTags:              testTags,
		CapacityType:          testCapacityType,
	}
	th.Equals(t, expectedConfig, actualConfig)
}
