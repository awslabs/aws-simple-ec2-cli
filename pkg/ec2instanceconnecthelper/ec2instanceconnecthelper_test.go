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

package ec2instanceconnecthelper_test

import (
	"encoding/base64"
	"testing"

	ec2ichelper "simple-ec2/pkg/ec2instanceconnecthelper"
	th "simple-ec2/test/testhelper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func TestGenerateSSHKeyPair(t *testing.T) {
	const expectedPublicKeyHeader = "ssh-rsa"
	const publicKeyHeaderRightIndex = 7
	const publicKeyContentLeftIndex = 8
	const expectedPrivateKeyHeader = "-----BEGIN RSA PRIVATE KEY-----"
	const expectedPrivateKeyTail = "-----END RSA PRIVATE KEY-----"
	const privateKeyHeaderRightIndex = 31
	const privateKeyTailOffset = len(expectedPrivateKeyTail)

	publicKey, privateKey, err := ec2ichelper.GenerateSSHKeyPair()
	th.Ok(t, err)
	th.Equals(t, expectedPublicKeyHeader, (*publicKey)[:7])
	th.Assert(t, isBase64((*publicKey)[publicKeyContentLeftIndex:]), "Public key is not Base64 encoded")
	th.Equals(t, expectedPrivateKeyHeader, (*privateKey)[:privateKeyHeaderRightIndex])
	th.Equals(t, expectedPrivateKeyTail, (*privateKey)[len(*privateKey)-privateKeyTailOffset-1:len(*privateKey)-1])
}

func TestGetInstancePublicDnsName_Success(t *testing.T) {
	const testDnsName = "test dns name"
	instance := &ec2.Instance{
		NetworkInterfaces: []*ec2.InstanceNetworkInterface{
			{
				Association: &ec2.InstanceNetworkInterfaceAssociation{
					PublicDnsName: aws.String(testDnsName),
				},
			},
		},
	}

	name, err := ec2ichelper.GetInstancePublicDnsName(instance)
	th.Ok(t, err)
	th.Equals(t, testDnsName, *name)
}

func TestGetInstancePublicDnsName_NoNetworkInterface(t *testing.T) {
	instance := &ec2.Instance{
		NetworkInterfaces: []*ec2.InstanceNetworkInterface{},
	}

	_, err := ec2ichelper.GetInstancePublicDnsName(instance)
	th.Nok(t, err)
}

func TestGetInstancePublicDnsName_NoDnsNameInNetworkInterface(t *testing.T) {
	instance := &ec2.Instance{
		NetworkInterfaces: []*ec2.InstanceNetworkInterface{
			{
				Association: &ec2.InstanceNetworkInterfaceAssociation{},
			},
		},
	}

	_, err := ec2ichelper.GetInstancePublicDnsName(instance)
	th.Nok(t, err)
}

// A helper function to decide whether a string is Base64 encoded or not
func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
