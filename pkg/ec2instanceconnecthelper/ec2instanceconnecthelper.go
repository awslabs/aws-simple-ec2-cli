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

package ec2instanceconnecthelper

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"ez-ec2/pkg/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2instanceconnect"
	"golang.org/x/crypto/ssh"
)

const userName = "ec2-user"
const passPhrase = "ez-ec2"

// Push an SSH key to an EC2 instance
func SendSSHPublicKey(sess *session.Session, availabilityZone, instanceId,
	publicKey string) error {
	svc := ec2instanceconnect.New(sess)
	input := &ec2instanceconnect.SendSSHPublicKeyInput{
		AvailabilityZone: aws.String(availabilityZone),
		InstanceId:       aws.String(instanceId),
		InstanceOSUser:   aws.String("ec2-user"),
		SSHPublicKey:     aws.String(publicKey),
	}

	result, err := svc.SendSSHPublicKey(input)
	if err != nil {
		return err
	}
	if !*result.Success {
		return errors.New("Sending public key failed")
	}

	return nil
}

// Generate a key pair for SSH. Returns the public and private keys
func GenerateSSHKeyPair() (publicKeyString, privateKeyString *string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	privateKeyPEM, err := x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(passPhrase), x509.PEMCipherAES256)
	if err != nil {
		return nil, nil, err
	}

	var private bytes.Buffer
	if err := pem.Encode(&private, privateKeyPEM); err != nil {
		return nil, nil, err
	}

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	public := ssh.MarshalAuthorizedKey(pub)
	return aws.String(string(public)), aws.String(private.String()), nil
}

// Establish an SSH connection to the instance
func EstablishSSHConnection(privateKey, instanceDnsName string, exitAtOnce bool) error {
	// Create the folder if it doesn't exist
	ezec2Dir := os.Getenv("HOME") + "/.ez-ec2"
	if _, err := os.Stat(ezec2Dir); os.IsNotExist(err) {
		err = os.MkdirAll(ezec2Dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Save the private key as a .pem file
	keyPath, err := config.SaveInConfigFolder("instance_connect.pem", []byte(privateKey), 0600)
	if err != nil {
		return err
	}

	/*
		This is an ugly workaround for ssh to work on TravisCI.
		Compare to the normal version, the workaround does the following additional steps:
		1. Add a passphrase to the key. This is not for security but just to make the key have a passphrase.
		2. Use sshpass to wrap ssh. Note that the version of sshpass has to be 1.06+ for flag -P to work.
		3. Always use -oStrictHostKeyChecking=no for ssh, otherwise sshpass won't work.
	*/

	// Arguments for the ssh command
	args := []string{
		"-Ppassphrase",
		fmt.Sprintf("-p%s", passPhrase),
		"ssh",
		fmt.Sprintf("-i%s", *keyPath),
		fmt.Sprintf("%s@%s", userName, instanceDnsName),
		"-oStrictHostKeyChecking=no",
	}

	// Decide whether to include additional arguments or not.
	if exitAtOnce {
		args = append(args, "exit")
	}

	cmd := exec.Command("sshpass", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	var errb bytes.Buffer
	cmd.Stderr = &errb

	/*
		The error thrown here, which only tells if the function is unsuccessful, isn't really useful.
		Therefore, return an error containing the message from cmd.Stderr.
	*/
	err = cmd.Run()
	if err != nil {
		return errors.New(err.Error() + errb.String())
	}

	return nil
}

// Connect to an instance
func ConnectInstance(sess *session.Session, instance *ec2.Instance, exitAtOnce bool) error {
	instanceDnsName, err := GetInstancePublicDnsName(instance)
	if err != nil {
		return err
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if err != nil {
		return err
	}

	availabilityZone := instance.Placement.AvailabilityZone
	instanceId := instance.InstanceId

	err = SendSSHPublicKey(sess, *availabilityZone, *instanceId, *publicKey)
	if err != nil {
		return err
	}

	err = EstablishSSHConnection(*privateKey, *instanceDnsName, exitAtOnce)
	if err != nil {
		return err
	}

	return nil
}

// Check if the instance has a public DNS name. If so, return it. Return an error otherwise.
func GetInstancePublicDnsName(instance *ec2.Instance) (*string, error) {
	if instance == nil || instance.NetworkInterfaces == nil || len(instance.NetworkInterfaces) <= 0 {
		return nil, errors.New("No network interfaces available for the instance")
	} else {
		for _, ni := range instance.NetworkInterfaces {
			if ni.Association != nil && ni.Association.PublicDnsName != nil {
				return ni.Association.PublicDnsName, nil
			}
		}
	}

	return nil, errors.New("No public DNS name available")
}
