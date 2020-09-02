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
	"bytes"
	"io/ioutil"
	"os"
	"strconv"
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
	if err != nil {
		t.Fatal(err)
	}

	// Check if the path is correct
	if *path != testConfigFilePath {
		os.Remove(*path)
		t.Fatal("The config file path is incorrect")
	}

	// Check the content of the file is correct
	readData, err := ioutil.ReadFile(testConfigFilePath)
	if err != nil {
		os.Remove(*path)
		t.Fatal(err)
	}
	if bytes.Compare(testData, readData) != 0 {
		t.Errorf("Config file content incorrect, expect: \"%s\" got: \"%s\"",
			testString, string(readData))
	}

	os.Remove(*path)
}

// This is merely a test config to test if functions work. It won't work in the real environment.
const testRegion = "us-somewhere"
const testImageId = "ami-12345"
const testInstanceType = "t2.micro"
const testSubnetId = "s-12345"
const testLaunchTemplateId = "lt-12345"
const testLaunchTemplateVersion = "1"
const testNewVPC = true

const expectedJson = `{"Region":"us-somewhere","ImageId":"ami-12345","InstanceType":"t2.micro","SubnetId":"s-12345","LaunchTemplateId":"lt-12345","LaunchTemplateVersion":"1","SecurityGroupIds":["sg-12345","sg-67890"],"NewVPC":true,"AutoTerminationTimerMinutes":0,"KeepEbsVolumeAfterTermination":false}`

var testSecurityGroup = []string{
	"sg-12345",
	"sg-67890",
}

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
	}

	err := config.SaveConfig(testConfig, aws.String(testConfigFileName))
	if err != nil {
		os.Remove(testConfigFilePath)
		t.Fatal(err)
	}

	// Check the content of the file is correct
	readData, err := ioutil.ReadFile(testConfigFilePath)
	if err != nil {
		os.Remove(testConfigFilePath)
		t.Fatal(err)
	}
	if expectedJson != string(readData) {
		t.Errorf("Config file content incorrect, expect: \"%s\" got: \"%s\"",
			expectedJson, string(readData))
	}

	os.Remove(testConfigFilePath)
}

func TestOverrideConfigWithFlags(t *testing.T) {
	simpleConfig := &config.SimpleInfo{}
	flagConfig := &config.SimpleInfo{
		Region:                testRegion,
		ImageId:               testImageId,
		InstanceType:          testInstanceType,
		SubnetId:              testSubnetId,
		LaunchTemplateId:      testLaunchTemplateId,
		LaunchTemplateVersion: testLaunchTemplateVersion,
		SecurityGroupIds:      testSecurityGroup,
		NewVPC:                testNewVPC,
	}

	config.OverrideConfigWithFlags(simpleConfig, flagConfig)

	// Check if the fields are correct
	compareConfig(flagConfig, simpleConfig, t)
}

func compareConfig(correctConfig, otherConfig *config.SimpleInfo, t *testing.T) {
	if correctConfig.Region != otherConfig.Region {
		t.Errorf("Region is not correct, expect: %s got %s",
			correctConfig.Region, otherConfig.Region)
	}
	if correctConfig.InstanceType != otherConfig.InstanceType {
		t.Errorf("InstanceType is not correct, expect: %s got %s",
			correctConfig.InstanceType, otherConfig.InstanceType)
	}
	if correctConfig.ImageId != otherConfig.ImageId {
		t.Errorf("ImageId is not correct, expect: %s got %s",
			correctConfig.ImageId, otherConfig.ImageId)
	}
	if correctConfig.SubnetId != otherConfig.SubnetId {
		t.Errorf("SubnetId is not correct, expect: %s got %s",
			correctConfig.SubnetId, otherConfig.SubnetId)
	}
	if correctConfig.LaunchTemplateId != otherConfig.LaunchTemplateId {
		t.Errorf("LaunchTemplateId is not correct, expect: %s got %s",
			correctConfig.LaunchTemplateId, otherConfig.LaunchTemplateId)
	}
	if correctConfig.LaunchTemplateVersion != otherConfig.LaunchTemplateVersion {
		t.Errorf("LaunchTemplateVersion is not correct, expect: %s got %s",
			correctConfig.LaunchTemplateVersion, otherConfig.LaunchTemplateVersion)
	}
	if !th.StringSliceEqual(correctConfig.SecurityGroupIds, otherConfig.SecurityGroupIds) {
		t.Errorf("SecurityGroupIds is not correct, expect: %s got %s",
			correctConfig.SecurityGroupIds, otherConfig.SecurityGroupIds)
	}
	if correctConfig.NewVPC != otherConfig.NewVPC {
		t.Errorf("NewVPC is not correct, expect: %s got %s",
			strconv.FormatBool(correctConfig.NewVPC), strconv.FormatBool(otherConfig.NewVPC))
	}
	if correctConfig.AutoTerminationTimerMinutes != otherConfig.AutoTerminationTimerMinutes {
		t.Errorf("AutoTerminationTimerMinutes is not correct, expect: %d got %d",
			correctConfig.AutoTerminationTimerMinutes, otherConfig.AutoTerminationTimerMinutes)
	}
	if correctConfig.KeepEbsVolumeAfterTermination != otherConfig.KeepEbsVolumeAfterTermination {
		t.Errorf("KeepEbsVolumeAfterTermination is not correct, expect: %s got %s",
			strconv.FormatBool(correctConfig.KeepEbsVolumeAfterTermination), strconv.FormatBool(otherConfig.KeepEbsVolumeAfterTermination))
	}
}

func TestReadConfig(t *testing.T) {
	err := ioutil.WriteFile(testConfigFilePath, []byte(expectedJson), 0644)
	if err != nil {
		os.Remove(testConfigFilePath)
		t.Fatal(err)
	}

	simpleConfig := &config.SimpleInfo{}
	err = config.ReadConfig(simpleConfig, aws.String(testConfigFileName))
	if err != nil {
		os.Remove(testConfigFilePath)
		t.Fatal(err)
	}

	// Check if the config is read correctly
	correctConfig := &config.SimpleInfo{
		Region:                testRegion,
		ImageId:               testImageId,
		InstanceType:          testInstanceType,
		SubnetId:              testSubnetId,
		LaunchTemplateId:      testLaunchTemplateId,
		LaunchTemplateVersion: testLaunchTemplateVersion,
		SecurityGroupIds:      testSecurityGroup,
		NewVPC:                testNewVPC,
	}

	compareConfig(correctConfig, simpleConfig, t)

	os.Remove(testConfigFilePath)
}
