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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	homedir "github.com/mitchellh/go-homedir"
)

const defaultConfigFileName = "simple-ec2.json"

var simpleEc2Dir = getHomeDir() + "/.simple-ec2"

/*
A simple config for reading config files or flags into primitive type information.
The config will later be used to parse into a detailed config and to launch an instance.
*/
type SimpleInfo struct {
	Region                        string
	ImageId                       string
	InstanceType                  string
	SubnetId                      string
	LaunchTemplateId              string
	LaunchTemplateVersion         string
	SecurityGroupIds              []string
	NewVPC                        bool
	AutoTerminationTimerMinutes   int
	KeepEbsVolumeAfterTermination bool
	IamInstanceProfile            string
	BootScriptFilePath            string
	UserTags                      map[string]string
	CapacityType                  string
}

/*
A detailed config for storing detailed information about resources that will be used to
launch an instance. This config is usually derived from a simple config.
*/
type DetailedInfo struct {
	Image            *ec2.Image
	Vpc              *ec2.Vpc
	Subnet           *ec2.Subnet
	InstanceTypeInfo *ec2.InstanceTypeInfo
	SecurityGroups   []*ec2.SecurityGroup
	TagSpecs         []*ec2.TagSpecification
}

type RequestInstanceInfo struct {
	ImageId                           *string
	InstanceType                      *string
	SubnetId                          *string
	SecurityGroupIds                  []*string
	IamInstanceProfile                *ec2.IamInstanceProfileSpecification
	LaunchTemplate                    *ec2.LaunchTemplateSpecification
	BlockDeviceMappings               []*ec2.BlockDeviceMapping
	LaunchTemplateBlockMappings       []*ec2.LaunchTemplateBlockDeviceMappingRequest
	InstanceInitiatedShutdownBehavior *string
	UserData                          *string
	LaunchTemplateTagSpecs            []*ec2.LaunchTemplateTagSpecificationRequest
}

func NewSimpleInfo() *SimpleInfo {
	var s SimpleInfo
	s.UserTags = make(map[string]string)
	return &s
}

// Get the home directory
func getHomeDir() string {
	var err error
	var h string
	if h, err = homedir.Dir(); err != nil {
		log.Printf("Warning: Failed to find home directory due to error: %s\n", err)
	}
	return h
}

// Read from a json file to parse config
func ReadConfig(simpleConfig *SimpleInfo, configFileName *string) error {
	if configFileName == nil {
		configFileName = aws.String(defaultConfigFileName)
	}

	path := simpleEc2Dir + "/" + *configFileName

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(data), simpleConfig)
	if err != nil {
		return err
	}

	return nil
}

// Override config fields, if they are specified in flags
func OverrideConfigWithFlags(simpleConfig *SimpleInfo, flagConfig *SimpleInfo) {
	if flagConfig.Region != "" {
		simpleConfig.Region = flagConfig.Region
	}
	if flagConfig.InstanceType != "" {
		simpleConfig.InstanceType = flagConfig.InstanceType
	}
	if flagConfig.ImageId != "" {
		simpleConfig.ImageId = flagConfig.ImageId
	}
	if flagConfig.SubnetId != "" {
		simpleConfig.SubnetId = flagConfig.SubnetId
	}
	if flagConfig.LaunchTemplateId != "" {
		simpleConfig.LaunchTemplateId = flagConfig.LaunchTemplateId
	}
	if flagConfig.LaunchTemplateVersion != "" {
		simpleConfig.LaunchTemplateVersion = flagConfig.LaunchTemplateVersion
	}
	if flagConfig.SecurityGroupIds != nil {
		simpleConfig.SecurityGroupIds = flagConfig.SecurityGroupIds
	}
	if flagConfig.NewVPC != false {
		simpleConfig.NewVPC = flagConfig.NewVPC
	}
	if flagConfig.AutoTerminationTimerMinutes != 0 {
		simpleConfig.AutoTerminationTimerMinutes = flagConfig.AutoTerminationTimerMinutes
	}
	if flagConfig.KeepEbsVolumeAfterTermination != false {
		simpleConfig.KeepEbsVolumeAfterTermination = flagConfig.KeepEbsVolumeAfterTermination
	}
	if flagConfig.IamInstanceProfile != "" {
		simpleConfig.IamInstanceProfile = flagConfig.IamInstanceProfile
	}
	if flagConfig.BootScriptFilePath != "" {
		simpleConfig.BootScriptFilePath = flagConfig.BootScriptFilePath
	}
	if flagConfig.UserTags != nil {
		simpleConfig.UserTags = flagConfig.UserTags
	}
}

// Save the config as a JSON config file
func SaveConfig(simpleConfig *SimpleInfo, configFileName *string) error {
	fmt.Println("Saving config...")
	if configFileName == nil {
		configFileName = aws.String(defaultConfigFileName)
	}

	jsonString, err := json.Marshal(simpleConfig)
	if err != nil {
		return err
	}

	data := []byte(jsonString)
	path, err := SaveInConfigFolder(*configFileName, data, 0644)
	if err != nil {
		return err
	}

	fmt.Println("Config successfully saved:", *path)

	return nil
}

// Save a file in the config folder
func SaveInConfigFolder(fileName string, data []byte, perm os.FileMode) (*string, error) {
	// Create the folder if it doesn't exist
	if _, err := os.Stat(simpleEc2Dir); os.IsNotExist(err) {
		err = os.MkdirAll(simpleEc2Dir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	// Save the file
	path := simpleEc2Dir + "/" + fileName
	err := ioutil.WriteFile(path, data, perm)
	if err != nil {
		return nil, err
	}

	return &path, nil
}
