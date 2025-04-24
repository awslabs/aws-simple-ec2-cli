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

<h2>⚠️ REPOSITORY ARCHIVE NOTICE</h2>

<h3> Status: The repository will be archived on 06/02/2025 </h3>

AWS is no longer actively developing this project, and will archive this repository on 06/02/2025. If you have questions about this change, please raise a customer issue.

## Important Notices

* ⛔ No new feature requests will be accepted
* 🚫 No bug fixes will be implemented
* 📝 No documentation updates will be made
* ❌ Issues and Pull Requests will not be reviewed or merged
* 🔒 Repository will be set to read-only during archival period

## Migration Guide

### Recommended Alternative: AWS CLI
The AWS Command Line Interface (AWS CLI) is the recommended and officially supported tool for managing EC2 instances. It provides all the functionality of Simple EC2 CLI with additional features and ongoing support.

### Key Benefits of AWS CLI:
* Official AWS support and regular updates
* Comprehensive EC2 instance management capabilities
* Consistent interface across all AWS services
* Enhanced security features

### Common Commands:
* Launch an instance
    ```
     aws ec2 run-instances —image-id <ami-xxxxx> —instance-type t2.micro —key-name <MyKeyPair> 
     ```
* List instances
    ```
     aws ec2 describe-instances
     ```
* Delete an instance
    ```
    aws ec2 terminate-instances --instance-ids <i-xxxxxx>
    ```

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
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.12.0/simple-ec2-`uname | tr '[:upper:]' '[:lower:]'`-amd64
chmod +x simple-ec2
```

#### ARM Linux
```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.12.0/simple-ec2-linux-arm
```

```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.12.0/simple-ec2-linux-arm64
```

#### Windows
```
curl -Lo simple-ec2 https://github.com/awslabs/aws-simple-ec2-cli/releases/download/v0.12.0/simple-ec2-windows-amd64.exe
```

## Examples

### Version

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
      --capacity-type string             Launch instance as "On-Demand" (the default) or "Spot"
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

Please confirm if you would like to launch instance with following options:

       CONFIGURATION                        │ VALUE                                                
     ───────────────────────────────────────┼──────────────────────────────────────────────────────
       Region                               │ us-east-1                                            
       VPC                                  │ vpc-example                
       Subnet                               │ subnet-example          
       Instance Type                        │ t1.micro                                             
       Capacity Type                        │ On-Demand                                                 
       Image                                │ ami-047a51fa27710816e                                
       Security Group                       │ sg-example  
       Keep EBS Volume(s) After Termination │ false                                                
       Auto Termination Timer in Minutes    │ None                                                   
       EBS Volumes                          │ /dev/xvda(gp2): 8 GiB                                
                                                                                                   
   >   Yes  
       No   
Options confirmed! Launching instance...
Launch Instance Success!
Instance ID: i-123example
```

**Single Command Launch With Flags**

```
$ simple-ec2 launch -r us-east-2 -m ami-123example -t t2.micro -s subnet-123example -g sg-123example

Please confirm if you would like to launch instance with following options:

       CONFIGURATION                        │ VALUE                                                
     ───────────────────────────────────────┼──────────────────────────────────────────────────────
       Region                               │ us-east-2                                            
       VPC                                  │ vpc-example                
       Subnet                               │ subnet-123example          
       Instance Type                        │ t2.micro                                             
       Capacity Type                        │ On-Demand                                                 
       Image                                │ ami-123example
       Security Group                       │ sg-123example  
       Keep EBS Volume(s) After Termination │ false                                                
       Auto Termination Timer in Minutes    │ None                                                   
       EBS Volumes                          │ /dev/xvda(gp2): 8 GiB                                
                                                                                                   
   >   Yes  
       No   
Options confirmed! Launching instance...
Launch Instance Success!
Instance ID: i-123example
```

**Interactive Mode Launch**

```
$ simple-ec2 launch -i


Select a region for the instance:

       REGION         │ DESCRIPTION                
     ─────────────────┼────────────────────────────
       ap-northeast-1 │ Asia Pacific (Tokyo)       
       ap-northeast-2 │ Asia Pacific (Seoul)       
       ap-northeast-3 │ Asia Pacific (Osaka)       
       ap-south-1     │ Asia Pacific (Mumbai)      
       ap-southeast-1 │ Asia Pacific (Singapore)   
       ap-southeast-2 │ Asia Pacific (Sydney)      
       ca-central-1   │ Canada (Central)           
       eu-central-1   │ Europe (Frankfurt)         
       eu-north-1     │ Europe (Stockholm)         
       eu-west-1      │ Europe (Ireland)           
       eu-west-2      │ Europe (London)            
       eu-west-3      │ Europe (Paris)             
       sa-east-1      │ South America (Sao Paulo)  
       us-east-1      │ US East (N. Virginia)      
   >   us-east-2      │ US East (Ohio)             
       us-west-1      │ US West (N. California)    
       us-west-2      │ US West (Oregon)           

How do you want to choose the instance type?

       Enter the instance type                          
       Provide vCPUs and memory information for advice  
   >   Use the default instance type, [t3.micro]        

Select an AMI for the instance:

       OPERATING SYSTEM │ IMAGE ID              │ CREATION DATE             
     ───────────────────┼───────────────────────┼───────────────────────────
   >   Amazon Linux 2   │ ami-017a73c6475f1cefe │ 2022-07-22T22:59:04.000Z  
       Ubuntu           │ ami-0c1efade7e2a5a12e │ 2022-08-10T12:06:14.000Z  
       Amazon Linux     │ ami-02a1b876e6016a354 │ 2022-07-16T02:38:59.000Z  
       Red Hat          │ ami-078cbc4c2d057c244 │ 2022-05-13T11:53:05.000Z  
       SUSE Linux       │ ami-0535d9b70179f9734 │ 2022-07-23T07:01:55.000Z  
       Windows          │ ami-04d1c6a7290ee815a │ 2022-08-10T07:21:08.000Z  

Persist EBS Volume(s) after the instance is terminated?

       Yes  
   >   No   

After how many minutes should the instance terminate? (0 for no auto-termination)

   > 25 

Select the VPC for the instance:

       VPC                                            │ CIDR BLOCK     
     ─────────────────────────────────────────────────┼────────────────
   >   vpc-123example                                 │ 172.31.0.0/16  
       vpc-example                                    │ 172.31.0.0/16  
       Create new VPC with default CIDR and 3 subnets │                

Select the subnet for the instance:

       SUBNET            │ AVAILABILITY ZONE │ CIDR BLOCK      
     ────────────────────┼───────────────────┼─────────────────
   >   subnet-123example │ us-east-2a        │ 172.31.0.0/24   
       subnet-456example │ us-east-2b        │ 172.31.16.0/24  
       subnet-789example │ us-east-2c        │ 172.31.32.0/24  

Select the security groups for the instance:

           SECURITY GROUP                               │ DESCRIPTION                             
         ───────────────────────────────────────────────┼─────────────────────────────────────────
     [x]   sg-123example                                │ My Favorite Security Group
     [ ]   sg-456example                                │ default VPC security group              
     [ ]   Create a new security group that enables SSH │                                         
                                                                                                                            
         [ SUBMIT ]                                                                                                         

Select an IAM Profile:

       PROFILE NAME              │ PROFILE ID            │ CREATION DATE                  
     ────────────────────────────┼───────────────────────┼────────────────────────────────
       Instance-Profile-1        │ AIPAXP7DUN6CORG253IFG │ 2021-01-20 14:31:28 +0000 UTC  
       Instance-Profile-2        │ AIPAXP7DUN6CJLXGLI2M5 │ 2021-01-20 14:31:51 +0000 UTC  
   >   Do not attach IAM profile │                       │                                

Would you like to add a filepath to the instance boot script?

       Yes  
   >   No   

Would you like to add tags to instances and persisted volumes?

       Yes  
   >   No   

Select capacity type. Spot instances are available at up to a 90% discount compared to On-Demand instances,
but they may get interrupted by EC2 with a 2-minute warning

       CAPACITY TYPE │ PRICE       
     ────────────────┼─────────────
   >   On-Demand     │ $0.0104/hr  
       Spot          │ $0.0031/hr  

Please confirm if you would like to launch instance with following options:
(Or select a configuration to repeat a question)

       CONFIGURATION                        │ VALUE                                                
     ───────────────────────────────────────┼──────────────────────────────────────────────────────
       Region                               │ us-east-2                                            
       VPC                                  │ vpc-123example                
       Subnet                               │ subnet-123example          
       Instance Type                        │ t3.micro                                             
       Capacity Type                        │ On-Demand                                            
       Image                                │ ami-017a73c6475f1cefe                                
       Security Group                       │ sg-123example  
       Keep EBS Volume(s) After Termination │ false                                                
       Auto Termination Timer in Minutes    │ 25                                                   
       EBS Volumes                          │ /dev/xvda(gp2): 8 GiB                                
                                                                                                   
   >   Yes  
       No   
Options confirmed! Launching instance...
Launch Instance Success!
Instance ID: i-123example

Do you want to save the configuration above as a JSON file that can be used in non-interactive mode and as question defaults

   >   Yes  
       No   
Saving config...
Config successfully saved: /Users/${USER}/.simple-ec2/simple-ec2.json
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

Select a region for the instance:

       REGION         │ DESCRIPTION                
     ─────────────────┼────────────────────────────
       ap-northeast-1 │ Asia Pacific (Tokyo)       
       ap-northeast-2 │ Asia Pacific (Seoul)       
       ap-northeast-3 │ Asia Pacific (Osaka)       
       ap-south-1     │ Asia Pacific (Mumbai)      
       ap-southeast-1 │ Asia Pacific (Singapore)   
       ap-southeast-2 │ Asia Pacific (Sydney)      
       ca-central-1   │ Canada (Central)           
       eu-central-1   │ Europe (Frankfurt)         
       eu-north-1     │ Europe (Stockholm)         
       eu-west-1      │ Europe (Ireland)           
       eu-west-2      │ Europe (London)            
       eu-west-3      │ Europe (Paris)             
       sa-east-1      │ South America (Sao Paulo)  
       us-east-1      │ US East (N. Virginia)      
   >   us-east-2      │ US East (Ohio)             
       us-west-1      │ US West (N. California)    
       us-west-2      │ US West (Oregon)           

Select the instance you want to connect to: 

       INSTANCE            │ TAG-KEY                       │ TAG-VALUE                                   
     ──────────────────────┼───────────────────────────────┼─────────────────────────────────────────────
   >   i-123example        │ CreatedBy                     │ simple-ec2                                  
                           │ CreatedTime                   │ 2022-08-19 14:04:08 CDT                     
       i-456example        │ CreatedBy                     │ simple-ec2                                  
                           │ CreatedTime                   │ 2022-08-19 13:58:33 CDT                     

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
Select a region for the instance:

       REGION         │ DESCRIPTION                
     ─────────────────┼────────────────────────────
       ap-northeast-1 │ Asia Pacific (Tokyo)       
       ap-northeast-2 │ Asia Pacific (Seoul)       
       ap-northeast-3 │ Asia Pacific (Osaka)       
       ap-south-1     │ Asia Pacific (Mumbai)      
       ap-southeast-1 │ Asia Pacific (Singapore)   
       ap-southeast-2 │ Asia Pacific (Sydney)      
       ca-central-1   │ Canada (Central)           
       eu-central-1   │ Europe (Frankfurt)         
       eu-north-1     │ Europe (Stockholm)         
       eu-west-1      │ Europe (Ireland)           
       eu-west-2      │ Europe (London)            
       eu-west-3      │ Europe (Paris)             
       sa-east-1      │ South America (Sao Paulo)  
       us-east-1      │ US East (N. Virginia)      
   >   us-east-2      │ US East (Ohio)             
       us-west-1      │ US West (N. California)    
       us-west-2      │ US West (Oregon)           

Select the instances you want to terminate: 

           INSTANCE            │ TAG-KEY                       │ TAG-VALUE                                   
         ──────────────────────┼───────────────────────────────┼─────────────────────────────────────────────
     [x]   i-123example        │ CreatedBy                     │ simple-ec2                                  
                               │ CreatedTime                   │ 2022-08-19 14:05:29 CDT                     
     [x]   i-456example        │ CreatedTime                   │ 2022-08-19 14:18:10 CDT                     
                               │ CreatedBy                     │ simple-ec2                                  
                                                                                                             
         [ SUBMIT ]                                                                                          

Are you sure you want to terminate 2 instance(s): [i-123example i-456example] 

   >   Yes  
       No   
Terminating instances
Instances [i-123example i-456example] terminated successfully
```

## Building
For build instructions please consult [BUILD.md](./BUILD.md).

## Communication
If you've run into a bug or have a new feature request, please open an issue.

##  Contributing
Contributions are welcome! Please read our [guidelines](./CONTRIBUTING.md) and our [Code of Conduct](./CODE_OF_CONDUCT.md)

## License
This project is licensed under the [Apache-2.0](./LICENSE) License.
