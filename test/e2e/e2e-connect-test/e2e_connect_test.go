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

package connect_e2e

import (
	"testing"
	"time"

	"ez-ec2/pkg/cfn"
	"ez-ec2/pkg/ec2helper"
	ec2ichelper "ez-ec2/pkg/ec2instanceconnecthelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

const testStackName = "ez-ec2-e2e-connect-test"
const correctRegion = "us-east-2"
const availabilityZone = "us-east-2a"

var sess = session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
var c = cfn.New(sess)
var instanceId *string

// This is not a test function. it just sets up the testing environment. The tests are tailored for us-east-2 region only
func TestSetupEnvironment(t *testing.T) {
	// Parse CloudFormation templates
	err := cfn.DecodeTemplateVariables()
	if err != nil {
		t.Fatal(err)
	}

	// The tests only work in us-east-2, so change the region if the region is not correct
	if *sess.Config.Region != correctRegion {
		sess.Config.Region = aws.String(correctRegion)
		c.Svc = cloudformation.New(sess)
	}

	_, _, instanceId, _, err = c.CreateStackAndGetResources(nil, aws.String(testStackName),
		cfn.E2eConnectTestCloudformationTemplate)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for a while after stack creation so that the instance can properly initialize
	time.Sleep(cfn.PostCreationWait)
}

var publicKey, privateKey *string

func TestGenerateSSHKeyPair(t *testing.T) {
	var err error
	publicKey, privateKey, err = ec2ichelper.GenerateSSHKeyPair()
	if err != nil {
		t.Error(err)
	}
}

func TestSendSSHPublicKey(t *testing.T) {
	if instanceId == nil {
		t.Fatal("Instance ID not found")
	}

	if publicKey == nil {
		t.Fatal("Public key not found")
	}

	// Incorrect availability zone
	err := ec2ichelper.SendSSHPublicKey(sess, "us-east-1a", *instanceId, *publicKey)
	if err == nil {
		t.Error("Wrong availability zone is used but no error")
	}

	// Incorrect instance ID
	err = ec2ichelper.SendSSHPublicKey(sess, availabilityZone, "i-1234567890", *publicKey)
	if err == nil {
		t.Error("Wrong instance ID is used but no error")
	}

	// Incorrect public key
	err = ec2ichelper.SendSSHPublicKey(sess, availabilityZone, *instanceId, "123")
	if err == nil {
		t.Error("Wrong public key is used but no error")
	}

	// Correct call
	err = ec2ichelper.SendSSHPublicKey(sess, availabilityZone, *instanceId, *publicKey)
	if err != nil {
		t.Error(err)
	}
}

func TestEstablishSSHConnection(t *testing.T) {
	if instanceId == nil {
		t.Fatal("Instance ID not found")
	}

	if privateKey == nil {
		t.Fatal("Private key not found")
	}

	h := ec2helper.New(sess)
	instance, err := h.GetInstanceById(*instanceId)
	if err != nil {
		t.Fatal(err)
	}

	// Check if the instance has a public DNS name
	instanceDnsName, err := ec2ichelper.GetInstancePublicDnsName(instance)
	if err != nil {
		t.Fatal(err)
	}

	// Correct call
	err = ec2ichelper.EstablishSSHConnection(*privateKey, *instanceDnsName, false, true)
	if err != nil {
		t.Error(err)
	}

	// Incorrect private key
	err = ec2ichelper.EstablishSSHConnection("fake-private-key", *instanceDnsName, false, true)
	if err == nil {
		t.Error("Wrong private key is used but no error")
	}

	// Incorrect instance DNS name
	err = ec2ichelper.EstablishSSHConnection(*privateKey, "fake-instance-DNS-name", false, true)
	if err == nil {
		t.Error("Wrong instance IP is used but no error")
	}
}

func TestCleanupEnvironment(t *testing.T) {
	err := c.DeleteStack(testStackName)
	if err != nil {
		t.Error(err)
	}
}
