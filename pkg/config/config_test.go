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
const testAutoTerminationTimerMinutes = 37
const testKeepEBSVolume = true
const testIamProfile = "iam-profile"
const testBootScriptFilePath = "some/path/to/bootscript"
const testCapacityType = "On-Spot-Demand"

var testTags = map[string]string{"testedBy": "BRYAN", "brokenBy": "CBASKIN"}
var testSecurityGroup = []string{"sg-12345", "sg-67890"}

// This JSON must match the above values used for testing
const expectedJson = `{"Region":"us-somewhere","ImageId":"ami-12345","InstanceType":"t2.micro","SubnetId":"s-12345","LaunchTemplateId":"lt-12345","LaunchTemplateVersion":"1","SecurityGroupIds":["sg-12345","sg-67890"],"NewVPC":true,"AutoTerminationTimerMinutes":37,"KeepEbsVolumeAfterTermination":true,"IamInstanceProfile":"iam-profile","BootScriptFilePath":"some/path/to/bootscript","UserTags":{"brokenBy":"CBASKIN","testedBy":"BRYAN"},"CapacityType":"On-Spot-Demand"}`

// This JSON must NOT match the above values, to verify overriding with flags
const overridableJson = `{"Region":"us-nowhere","ImageId":"ami-67890","InstanceType":"t2.nano","SubnetId":"s-67890","LaunchTemplateId":"lt-67890","LaunchTemplateVersion":"2","SecurityGroupIds":["sg-98765","sg-43210"],"NewVPC":false,"AutoTerminationTimerMinutes":0,"KeepEbsVolumeAfterTermination":false,"IamInstanceProfile":"you-are-profile","BootScriptFilePath":"some/other/path/to/bootscript","UserTags":{"brokenBy":"JFINLAY","testedBy":"BRYAN"},"CapacityType":"On-Demand"}`

// TestSaveConfig writes a config to a temporary file and verifies that the resulting JSON is correct
func TestSaveConfig(t *testing.T) {
	testConfig := &config.SimpleInfo{
		Region:                        testRegion,
		ImageId:                       testImageId,
		InstanceType:                  testInstanceType,
		SubnetId:                      testSubnetId,
		LaunchTemplateId:              testLaunchTemplateId,
		LaunchTemplateVersion:         testLaunchTemplateVersion,
		SecurityGroupIds:              testSecurityGroup,
		NewVPC:                        testNewVPC,
		AutoTerminationTimerMinutes:   testAutoTerminationTimerMinutes,
		KeepEbsVolumeAfterTermination: testKeepEBSVolume,
		IamInstanceProfile:            testIamProfile,
		BootScriptFilePath:            testBootScriptFilePath,
		UserTags:                      testTags,
		CapacityType:                  testCapacityType,
	}

	err := config.SaveConfig(testConfig, aws.String(testConfigFileName))
	defer os.Remove(testConfigFilePath)
	th.Ok(t, err)

	// Check the content of the file is correct
	readData, err := ioutil.ReadFile(testConfigFilePath)
	th.Ok(t, err)
	th.Equals(t, expectedJson, string(readData))
}

// TestOverrideConfigWithFlags reads a config from JSON (via a temporary file), overrides it with different values,
// and verifies that the overrides take precedence over the original JSON
func TestOverrideConfigWithFlags(t *testing.T) {
	actualConfig, err := readConfigFromFile(overridableJson)
	th.Ok(t, err)
	expectedConfig := &config.SimpleInfo{
		Region:                        testRegion,
		ImageId:                       testImageId,
		InstanceType:                  testInstanceType,
		SubnetId:                      testSubnetId,
		LaunchTemplateId:              testLaunchTemplateId,
		LaunchTemplateVersion:         testLaunchTemplateVersion,
		SecurityGroupIds:              testSecurityGroup,
		NewVPC:                        testNewVPC,
		AutoTerminationTimerMinutes:   testAutoTerminationTimerMinutes,
		KeepEbsVolumeAfterTermination: testKeepEBSVolume,
		IamInstanceProfile:            testIamProfile,
		BootScriptFilePath:            testBootScriptFilePath,
		UserTags:                      testTags,
		CapacityType:                  testCapacityType,
	}
	config.OverrideConfigWithFlags(actualConfig, expectedConfig)
	th.Equals(t, expectedConfig, actualConfig)
}

// readConfigFromFile writes the given JSON string to a temporary file and unmarshals it into a SimpleInfo object
func readConfigFromFile(configJson string) (*config.SimpleInfo, error) {
	err := ioutil.WriteFile(testConfigFilePath, []byte(configJson), 0644)
	defer os.Remove(testConfigFilePath)
	if err != nil {
		return nil, err
	}

	configFromFile := config.NewSimpleInfo()
	err = config.ReadConfig(configFromFile, aws.String(testConfigFileName))
	if err != nil {
		return nil, err
	}
	return configFromFile, nil
}

// TestReadConfig reads a config from JSON (via a temporary file) and verifies that the resulting SimpleInfo object has the correct values
func TestReadConfig(t *testing.T) {
	actualConfig, err := readConfigFromFile(expectedJson)
	th.Ok(t, err)

	// Check if the config is read correctly
	expectedConfig := &config.SimpleInfo{
		Region:                        testRegion,
		ImageId:                       testImageId,
		InstanceType:                  testInstanceType,
		SubnetId:                      testSubnetId,
		LaunchTemplateId:              testLaunchTemplateId,
		LaunchTemplateVersion:         testLaunchTemplateVersion,
		SecurityGroupIds:              testSecurityGroup,
		NewVPC:                        testNewVPC,
		AutoTerminationTimerMinutes:   testAutoTerminationTimerMinutes,
		KeepEbsVolumeAfterTermination: testKeepEBSVolume,
		IamInstanceProfile:            testIamProfile,
		BootScriptFilePath:            testBootScriptFilePath,
		UserTags:                      testTags,
		CapacityType:                  testCapacityType,
	}
	th.Equals(t, expectedConfig, actualConfig)
}
