<h1>AWS Simple EC2 CLI</h1>

<h4>A CLI tool that simplifies the process of launching, connecting and terminating an EC2 instance.</h4>

<p>
	<a href="https://golang.org/doc/go1.18">
	<img src="https://img.shields.io/github/go-mod/go-version/aws/amazon-ec2-metadata-mock?color=blueviolet" alt="go-version">
	</a>
	<a href="https://opensource.org/licenses/Apache-2.0">
	<img src="https://img.shields.io/badge/License-Apache%202.0-ff69b4.svg?color=orange" alt="license">
	</a>
</p>




<div>
  <hr>
</div>

## Summary

In order to launch a new EC2 instance, customers need to specify a lot of options, and it can be a slow and overwhelming task. It requires users to have an initial network stack (VPC-Id/Subnet-Id/Security-Groups), remote login, and many more. Often times, we require EC2 instance for adhoc testing for a short period of time without requiring complex networking infrastructure in place. AWS Simple EC2 CLI aims to solve this issue and make it easier for users to launch, connect and terminate EC2 instances with a single command

## Major Features

- Launch an instance using single command
- Connect to an instance using single command
- Terminate an instance using single command
- Interactive mode that help users to decide parameters to use
- Config file for more convenient launch

## Installation and Configuration
### Install AWS CLI

To execute the CLI, you will need AWS credentials configured. Take a look at the [AWS CLI configuration documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html#config-settings-and-precedence) for details on the various ways to configure credentials. Alternatively, you can try out the AWS Simple EC2 CLI by populating following environment variables:

```
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
# Set default Region (optional)
export AWS_REGION="us-east-1" 
```
### Install w/ Homebrew

```
brew tap aws/tap
brew install aws-simple-ec2-cli
```
### Install w/ Curl

#### MacOS/Linux
```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.8.2/simple-ec2-`uname | tr '[:upper:]' '[:lower:]'`-amd64
chmod +x simple-ec2
```

#### ARM Linux
```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.8.2/simple-ec2-linux-arm
```

```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.8.2/simple-ec2-linux-arm64
```

#### Windows
```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.8.2/simple-ec2-windows-amd64.exe
```

## Examples

### Version

**Single Command Launch**
```
$ simple-ec2 version
Prints the version of this tool
```
### Launch

**All CLI Options**

```
$ simple-ec2 launch -h
Launch an Amazon EC2 instance with the default configurations. All configurations can be overridden by configurations provided by configuration files or user input.

Usage:
  simple-ec2 launch [flags]

Flags:
  -a, --auto-termination-timer int       The auto-termination timer for the instance in minutes
  -b, --boot-script string               The absolute filepath to a bash script passed to the instance and executed after the instance starts (user data)
  -h, --help                             help for launch
  -p, --iam-instance-profile string      The profile containing an IAM role to attach to the instance
  -m, --image-id string                  The image id of the AMI used to launch the instance
  -t, --instance-type string             The instance type of the instance
  -i, --interactive                      Interactive mode
  -k, --keep-ebs                         Keep EBS volumes after instance termination
  -l, --launch-template-id string        The launch template id with which the instance will be launched
  -v, --launch-template-version string   The launch template version with which the instance will be launched
  -r, --region string                    The region where the instance will be launched
  -c, --save-config                      Save config as a JSON config file
  -g, --security-group-ids strings       The security groups with which the instance will be launched
  -s, --subnet-id string                 The subnet id in which the instance will be launched
      --tags stringToString              The tags applied to instances and volumes at launch (Example: tag1=val1,tag2=val2) (default [])
```

**Single Command Launch**
```
$ simple-ec2 launch

+--------------------------------------+--------------------------+
| Region                               | us-east-1                |
+--------------------------------------+--------------------------+
| VPC                                  | vpc-example              |
+--------------------------------------+--------------------------+
| Subnet                               | subnet-example 	  |
+--------------------------------------+--------------------------+
| Instance Type                        | t1.micro                 |
+--------------------------------------+--------------------------+
| Image                                | ami-047a51fa27710816e    |
+--------------------------------------+--------------------------+
| Security Group                       | sg-example               |
+--------------------------------------+--------------------------+
| Keep EBS Volume(s) After Termination | false                    |
+--------------------------------------+--------------------------+
| Auto Termination Timer in Minutes    | None                     |
+--------------------------------------+--------------------------+
| EBS Volumes                          | /dev/xvda(gp2): 8 GiB    |
+--------------------------------------+--------------------------+
[ yes / no ]
Please confirm if you would like to launch instance with following options: yes

Options confirmed! Launching instance...
Launch Instance Success!
Instance ID: i-123example
```

**Single Command Launch With Flags**

```
$ simple-ec2 launch -r us-east-2 -m ami-123example -t t2.micro -s subnet-123example -g sg-123example

+--------------------------------------+--------------------------+
| Region                               | us-east-2                |
+--------------------------------------+--------------------------+
| VPC                                  | vpc-example              |
+--------------------------------------+--------------------------+
| Subnet                               | subnet-123example	  |
+--------------------------------------+--------------------------+
| Instance Type                        | t1.micro                 |
+--------------------------------------+--------------------------+
| Image                                | ami-123example	          |
+--------------------------------------+--------------------------+
| Security Group                       | sg-123example            |
+--------------------------------------+--------------------------+
| Keep EBS Volume(s) After Termination | false                    |
+--------------------------------------+--------------------------+
| Auto Termination Timer in Minutes    | None                     |
+--------------------------------------+--------------------------+
| EBS Volumes                          | /dev/xvda(gp2): 8 GiB    |
+--------------------------------------+--------------------------+
[ yes / no ]
Please confirm if you would like to launch instance with following options: yes
Options confirmed! Launching instance...
Launch Instance Success!
Instance ID: i-123example
```

**Interactive Mode Launch**

```
$ simple-ec2 launch -i

+--------+----------------+---------------------------+
| OPTION |     REGION     |        DESCRIPTION        |
+--------+----------------+---------------------------+
| 1.     | ap-northeast-1 | Asia Pacific (Tokyo)      |
+--------+----------------+---------------------------+
| 2.     | ap-northeast-2 | Asia Pacific (Seoul)      |
+--------+----------------+---------------------------+
| 3.     | ap-northeast-3 | Asia Pacific (Osaka)      |
+--------+----------------+---------------------------+
| 4.     | ap-south-1     | Asia Pacific (Mumbai)     |
+--------+----------------+---------------------------+
| 5.     | ap-southeast-1 | Asia Pacific (Singapore)  |
+--------+----------------+---------------------------+
| 6.     | ap-southeast-2 | Asia Pacific (Sydney)     |
+--------+----------------+---------------------------+
| 7.     | ca-central-1   | Canada (Central)          |
+--------+----------------+---------------------------+
| 8.     | eu-central-1   | Europe (Frankfurt)        |
+--------+----------------+---------------------------+
| 9.     | eu-north-1     | Europe (Stockholm)        |
+--------+----------------+---------------------------+
| 10.    | eu-west-1      | Europe (Ireland)          |
+--------+----------------+---------------------------+
| 11.    | eu-west-2      | Europe (London)           |
+--------+----------------+---------------------------+
| 12.    | eu-west-3      | Europe (Paris)            |
+--------+----------------+---------------------------+
| 13.    | sa-east-1      | South America (Sao Paulo) |
+--------+----------------+---------------------------+
| 14.    | us-east-1      | US East (N. Virginia)     |
+--------+----------------+---------------------------+
| 15.    | us-east-2      | US East (Ohio)            |
+--------+----------------+---------------------------+
| 16.    | us-west-1      | US West (N. California)   |
+--------+----------------+---------------------------+
| 17.    | us-west-2      | US West (Oregon)          |
+--------+----------------+---------------------------+
Region [us-east-1]:  14


+--------+----------------------------------------+----------------+
| OPTION |            LAUNCH TEMPLATE             | LATEST VERSION |
+--------+----------------------------------------+----------------+
| 1.     | Dev-dsk_Template(lt-05f448f1797b94c28) | 1              |
+--------+----------------------------------------+----------------+
| 2.     | ExampleTemplate1(lt-0f000ff5a94cf8088) | 3              |
+--------+----------------------------------------+----------------+
| 3.     | Do not use launch template             |		   |
+--------+----------------------------------------+----------------+
Launch Template [Do not use launch template]:  3


1. I will enter the instance type
2. I need advice given vCPUs and memory
Instance Select Method [t1.micro]:  1


Please enter the instance type (eg. m5.xlarge, c5.xlarge) [t1.micro]: t2.micro

⣯ fetching images

+--------+------------------+-----------------------+--------------------------+
| OPTION | OPERATING SYSTEM |       IMAGE ID        |      CREATION DATE       |
+--------+------------------+-----------------------+--------------------------+
| 1.     | Amazon Linux 2   | ami-047a51fa27710816e | 2021-01-26T07:39:02.000Z |
+--------+------------------+-----------------------+--------------------------+
| 2.     | Ubuntu           | ami-02fe94dee086c0c37 | 2021-01-28T19:54:39.000Z |
+--------+------------------+-----------------------+--------------------------+
| 3.     | Amazon Linux     | ami-0d08a21fc010da680 | 2021-01-26T16:39:01.000Z |
+--------+------------------+-----------------------+--------------------------+
| 4.     | Red Hat          | ami-096fda3c22c1c990a | 2020-11-02T11:01:38.000Z |
+--------+------------------+-----------------------+--------------------------+
| 5.     | SUSE Linux       | ami-0a16c2295ef80ff63 | 2020-12-12T05:12:10.000Z |
+--------+------------------+-----------------------+--------------------------+
| 6.     | Windows          | ami-032e26fff3bb717f3 | 2021-01-13T23:34:55.000Z |
+--------+------------------+-----------------------+--------------------------+
[ any image id ]: Select the image id
AMI [Latest Amazon Linux 2 image]:  1

[ yes / no ]
Persist EBS volume(s) after the instance is terminated? [no]: no

[ integer ] Auto-termination timer in minutes
[ no ] No auto-termination
Auto-termination timer [no]: 25

+--------+------------------------------------------------+---------------+
| OPTION |                      VPC                       |  CIDR BLOCK   |
+--------+------------------------------------------------+---------------+
| 1.     | vpc-123example                                 | 172.31.0.0/16 |
+--------+------------------------------------------------+---------------+
| 2.     | Create new VPC with default CIDR and 3 subnets |
+--------+------------------------------------------------+---------------+
VPC [vpc-123example]: 1

+--------+------------------------------------------+-------------------+----------------+
| OPTION |                  SUBNET                  | AVAILABILITY ZONE |   CIDR BLOCK   |
+--------+------------------------------------------+-------------------+----------------+
| 1.     | subnet-123example	                    | us-east-1c        | 172.31.48.0/20 |
+--------+------------------------------------------+-------------------+----------------+
| 2.     | subnet-456example                 	    | us-east-1b        | 172.31.16.0/20 |
+--------+------------------------------------------+-------------------+----------------+
| 3.     | subnet-789example	                    | us-east-1d        | 172.31.64.0/20 |
+--------+------------------------------------------+-------------------+----------------+
Subnet [subnet-123example]: 1

+--------+-----------------------------------------------------+-------------------------------------------------------+
| OPTION |                   SECURITY GROUP                    |                      DESCRIPTION                      |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 1.     | sg-123example	                               | launch-wizard-1 created 2020-03-02T14:36:06.327-06:00 |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 2.     | simple-ec2 SSH Security Group(sg-456example)	       | Created by simple-ec2 for SSH connection to instances |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 3.     | sg-789example                                       | default VPC security group                            |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 4.     | Add all available security groups                   |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 5.     | Create a new security group that enables SSH        |
+--------+-----------------------------------------------------+-------------------------------------------------------+
Security Group(s) [sg-789example]: 3

+--------+-----------------------------------------------------+-------------------------------------------------------+
| OPTION |                   SECURITY GROUP                    |                      DESCRIPTION                      |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 1.     | sg-123example	                               | launch-wizard-1 created 2020-03-02T14:36:06.327-06:00 |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 2.     | simple-ec2 SSH Security Group(sg-456example)	       | Created by simple-ec2 for SSH connection to instances |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 3.     | Add all available security groups                   | 
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 4.     | Create a new security group that enables SSH        |
+--------+-----------------------------------------------------+-------------------------------------------------------+
| 5.     | Don't add any more security groups        	       |
+--------+-----------------------------------------------------+-------------------------------------------------------+
If you wish to add additional security group(s), add from the following:
Security Group(s) already selected: [sg-789example]: 5

+--------+-----------------------------------------------------------------------------------+-----------------------+-------------------------------+
| OPTION |                                   PROFILE NAME                                    |      PROFILE ID       |         CREATION DATE         |
+--------+-----------------------------------------------------------------------------------+-----------------------+-------------------------------+
| 1.     | Instance-Profile-1                                                                | AIPAXP7DUN6CORG253IFG | 2021-01-20 14:31:28 +0000 UTC |
+--------+-----------------------------------------------------------------------------------+-----------------------+-------------------------------+
| 2.     | Instance-Profile-2                                                                | AIPAXP7DUN6CJLXGLI2M5 | 2021-01-20 14:31:51 +0000 UTC |
+--------+-----------------------------------------------------------------------------------+-----------------------+-------------------------------+
| 3.     | Instance-Profile-3                                                                | AIPAXP7DUN6CFUJT5Q6VR | 2021-01-20 14:32:14 +0000 UTC |
+--------+-----------------------------------------------------------------------------------+-----------------------+-------------------------------+
| 4.     | Do not attach IAM profile                                                         |                       |                               |
+--------+-----------------------------------------------------------------------------------+-----------------------+-------------------------------+
IAM Profile [Do not attach IAM profile]: 4

Add filepath to instance boot script?
format: absolute file path [no]: no

Add tags to instances and persisted volumes?
format: tag1|val1,tag2|val2 [no]: no

+--------------------------------------+--------------------------+
| Region                               | us-east-1                |
+--------------------------------------+--------------------------+
| VPC                                  | vpc-123example           |
+--------------------------------------+--------------------------+
| Subnet                               | subnet-123example 	  |
+--------------------------------------+--------------------------+
| Instance Type                        | t2.micro                 |
+--------------------------------------+--------------------------+
| Image                                | ami-047a51fa27710816e    |
+--------------------------------------+--------------------------+
| Security Group                       | sg-789example            |
+--------------------------------------+--------------------------+
| Keep EBS Volume(s) After Termination | false                    |
+--------------------------------------+--------------------------+
| Auto Termination Timer in Minutes    | 25                       |
+--------------------------------------+--------------------------+
| EBS Volumes                          | /dev/xvda(gp2): 8 GiB    |
+--------------------------------------+--------------------------+
[ yes / no ]
Please confirm if you would like to launch instance with following options: yes
Options confirmed! Launching instance...
Launch Instance Success!
Instance ID: i-123example

[ yes / no ]
Do you want to save the configuration above as a JSON file that can be used in non-interactive mode? [no]: yes
Saving config...
Config successfully saved: /Users/$USER/.simple-ec2/simple-ec2.json
```

### Connect

**All CLI Options**

```
$ simple-ec2 connect -h
Connect to an Amazon EC2 Instance, given the region and instance id

Usage:
  simple-ec2 connect [flags]

Flags:
  -h, --help                 help for connect
  -n, --instance-id string   The instance id of the instance you want to connect to
  -i, --interactive          Interactive mode
  -r, --region string        The region in which the instance you want to connect locates

```

**Single Command Connect**

```
$ simple-ec2 connect -r us-east-2 -n i-123example
Last login: Wed Jul 29 21:01:45 2020 from 52.95.4.1

       __|  __|_  )
       _|  (     /   Amazon Linux 2 AMI
      ___|\___|___|

https://aws.amazon.com/amazon-linux-2/
14 package(s) needed for security, out of 31 available
Run "sudo yum update" to apply all updates.
[ec2-user@ip-example ~]$ exit
logout

```

**Interactive Mode Connect**

```
$ simple-ec2 connect -i

+--------+----------------+---------------------------+
| OPTION |     REGION     |        DESCRIPTION        |
+--------+----------------+---------------------------+
| 1.     | ap-northeast-1 | Asia Pacific (Tokyo)      |
+--------+----------------+---------------------------+
| 2.     | ap-northeast-2 | Asia Pacific (Seoul)      |
+--------+----------------+---------------------------+
| 3.     | ap-northeast-3 | Asia Pacific (Osaka)      |
+--------+----------------+---------------------------+
| 4.     | ap-south-1     | Asia Pacific (Mumbai)     |
+--------+----------------+---------------------------+
| 5.     | ap-southeast-1 | Asia Pacific (Singapore)  |
+--------+----------------+---------------------------+
| 6.     | ap-southeast-2 | Asia Pacific (Sydney)     |
+--------+----------------+---------------------------+
| 7.     | ca-central-1   | Canada (Central)          |
+--------+----------------+---------------------------+
| 8.     | eu-central-1   | Europe (Frankfurt)        |
+--------+----------------+---------------------------+
| 9.     | eu-north-1     | Europe (Stockholm)        |
+--------+----------------+---------------------------+
| 10.    | eu-west-1      | Europe (Ireland)          |
+--------+----------------+---------------------------+
| 11.    | eu-west-2      | Europe (London)           |
+--------+----------------+---------------------------+
| 12.    | eu-west-3      | Europe (Paris)            |
+--------+----------------+---------------------------+
| 13.    | sa-east-1      | South America (Sao Paulo) |
+--------+----------------+---------------------------+
| 14.    | us-east-1      | US East (N. Virginia)     |
+--------+----------------+---------------------------+
| 15.    | us-east-2      | US East (Ohio)            |
+--------+----------------+---------------------------+
| 16.    | us-west-1      | US West (N. California)   |
+--------+----------------+---------------------------+
| 17.    | us-west-2      | US West (Oregon)          |
+--------+----------------+---------------------------+
Region [us-east-1]: 14

+--------+---------------------+-------------+-----------------------+
| OPTION |      INSTANCE       |   TAG-KEY   |       TAG-VALUE       |
+--------+---------------------+-------------+-----------------------+
| 1.     | i-123example	       | CreatedTime | 2021-2-8 13:7:59 CST  |
+--------+---------------------+-------------+-----------------------+
|        |                     | CreatedBy   | simple-ec2            |
+--------+---------------------+-------------+-----------------------+
| 2.     | i-456example        | CreatedBy   | simple-ec2            |
+--------+---------------------+-------------+-----------------------+
|        |                     | CreatedTime | 2021-2-8 13:27:55 CST |
+--------+---------------------+-------------+-----------------------+
Select the instance you want to connect to: 1


       __|  __|_  )
       _|  (     /   Amazon Linux 2 AMI
      ___|\___|___|

https://aws.amazon.com/amazon-linux-2/
14 package(s) needed for security, out of 31 available
Run "sudo yum update" to apply all updates.
[ec2-user@ip-example ~]$ exit
logout
```

### Terminate

**All CLI Options**

```
$ simple-ec2 terminate -h
Terminate Amazon EC2 Instances, given the region and instance ids or tag values

Usage:
  simple-ec2 terminate [flags]

Flags:
  -h, --help                   help for terminate
  -n, --instance-ids strings   The instance ids of the instances you want to terminate
  -i, --interactive            Interactive mode
  -r, --region string          The region in which the instances you want to terminate locates
        --tags stringToString    Terminate instances containing EXACT tag key-pair (Example: CreatedBy=simple-ec2) (default [])
```

**One Command Terminate**

```
$ simple-ec2 terminate -r us-east-2 -n i-123example
Terminating instances
Instances [i-123example] terminated successfully
```

**One Command Terminate using tags**

```
$ simple-ec2 terminate -r us-east-1 --tags CreatedBy=simple-ec2
Terminating instances
Instances [i-123example i-456example] terminated successfully
```

**Interactive Terminate**

```
$ simple-ec2 terminate -i

+--------+----------------+---------------------------+
| OPTION |     REGION     |        DESCRIPTION        |
+--------+----------------+---------------------------+
| 1.     | ap-northeast-1 | Asia Pacific (Tokyo)      |
+--------+----------------+---------------------------+
| 2.     | ap-northeast-2 | Asia Pacific (Seoul)      |
+--------+----------------+---------------------------+
| 3.     | ap-northeast-3 | Asia Pacific (Osaka)      |
+--------+----------------+---------------------------+
| 4.     | ap-south-1     | Asia Pacific (Mumbai)     |
+--------+----------------+---------------------------+
| 5.     | ap-southeast-1 | Asia Pacific (Singapore)  |
+--------+----------------+---------------------------+
| 6.     | ap-southeast-2 | Asia Pacific (Sydney)     |
+--------+----------------+---------------------------+
| 7.     | ca-central-1   | Canada (Central)          |
+--------+----------------+---------------------------+
| 8.     | eu-central-1   | Europe (Frankfurt)        |
+--------+----------------+---------------------------+
| 9.     | eu-north-1     | Europe (Stockholm)        |
+--------+----------------+---------------------------+
| 10.    | eu-west-1      | Europe (Ireland)          |
+--------+----------------+---------------------------+
| 11.    | eu-west-2      | Europe (London)           |
+--------+----------------+---------------------------+
| 12.    | eu-west-3      | Europe (Paris)            |
+--------+----------------+---------------------------+
| 13.    | sa-east-1      | South America (Sao Paulo) |
+--------+----------------+---------------------------+
| 14.    | us-east-1      | US East (N. Virginia)     |
+--------+----------------+---------------------------+
| 15.    | us-east-2      | US East (Ohio)            |
+--------+----------------+---------------------------+
| 16.    | us-west-1      | US West (N. California)   |
+--------+----------------+---------------------------+
| 17.    | us-west-2      | US West (Oregon)          |
+--------+----------------+---------------------------+
Region [us-east-1]:  14

+--------+---------------------+-------------+-----------------------+
| OPTION |      INSTANCE       |   TAG-KEY   |       TAG-VALUE       |
+--------+---------------------+-------------+-----------------------+
| 1.     | i-123example	       | CreatedTime | 2021-2-8 13:7:59 CST  |
+--------+---------------------+-------------+-----------------------+
|        |                     | CreatedBy   | simple-ec2            |
+--------+---------------------+-------------+-----------------------+
Select the instance you want to terminate: 1

[ yes / no ]
Are you sure you want to terminate 1 instance(s): [i-123example]  [no]: yes

Terminating instances
Instances [i-123example] terminated successfully
```

## Building
For build instructions please consult [BUILD.md](./BUILD.md).

## Communication
If you've run into a bug or have a new feature request, please open an issue.

##  Contributing
Contributions are welcome! Please read our [guidelines](./CONTRIBUTING.md) and our [Code of Conduct](./CODE_OF_CONDUCT.md)

## License
This project is licensed under the [Apache-2.0](./LICENSE) License.
