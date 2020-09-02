package cfn_e2e

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"testing"

	"simple-ec2/pkg/cfn"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

const testStackName = "simple-ec2-e2e-cfn-test"
const correctRegion = "us-east-2"

var sess = session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
var c = cfn.New(sess)
var testAvailabilityZones = []*ec2.AvailabilityZone{
	&ec2.AvailabilityZone{
		ZoneName: aws.String("us-east-2a"),
	},
	&ec2.AvailabilityZone{
		ZoneName: aws.String("us-east-2b"),
	},
	&ec2.AvailabilityZone{
		ZoneName: aws.String("us-east-2c"),
	},
}

func TestCreateStackAndGetResources(t *testing.T) {
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

	vpcId, subnetIds, instanceId, _, err := c.CreateStackAndGetResources(testAvailabilityZones,
		aws.String(testStackName), cfn.E2eCfnTestCloudformationTemplate)
	if err != nil {
		t.Fatal(err)
	}
	if vpcId == nil {
		t.Error("Expect a VPC ID but got none")
	}
	if subnetIds == nil || len(subnetIds) <= 0 {
		t.Error("Expect subnet IDs but got none")
	}
	if instanceId == nil {
		t.Error("Expect an instance ID but got none")
	}
}

func TestDeleteStack(t *testing.T) {
	err := c.DeleteStack(testStackName)
	if err != nil {
		t.Error(err)
	}
}
