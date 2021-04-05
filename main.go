package main

import (
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
	"github.com/pulumi/pulumi-aws/sdk/v3/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v3/go/aws"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Create VPC
		itgixVpc, vpcCreateErr := ec2.NewVpc(ctx, "pulumi-test-vpc", &ec2.VpcArgs{
			CidrBlock:	pulumi.String("172.16.0.0/16"),
			Tags:		pulumi.StringMap{
							"Name": pulumi.String("pulumi-test-vpc"),
							},
		})

		if vpcCreateErr != nil {
			return vpcCreateErr
		}

		// Create Subnet
		itgixSubnet, subnetCreateErr := ec2.NewSubnet(ctx, "pulumi-test-subnet", &ec2.SubnetArgs{
			VpcId:				itgixVpc.ID(),
			CidrBlock:			pulumi.String("172.16.10.0/24"),
			AvailabilityZone:	pulumi.String("eu-central-1a"),
			Tags:				pulumi.StringMap{
									"Name": pulumi.String("pulumi-test-subnet"),
									},
		})

		if subnetCreateErr != nil {
			return subnetCreateErr
		}

		// Create Security Group
		itgixSecurityGroup, createSgErr := ec2.NewSecurityGroup(ctx, "pulumi-test-sg", &ec2.SecurityGroupArgs{
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol: pulumi.String("tcp"),
					FromPort: pulumi.Int(80),
					ToPort: pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		})

		if createSgErr != nil {
			return createSgErr
		}

		// Create NetworkInterface
		itgixNetworkInterface, createNwIfaceErr := ec2.NewNetworkInterface(ctx, "pulumi-test-nw-iface", &ec2.NetworkInterfaceArgs{
			SubnetId:		itgixSubnet.ID(),
			PrivateIps:		pulumi.StringArray{
				pulumi.String("172.16.10.100"),
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("primary_network_Interface"),
			},
			SecurityGroups: pulumi.StringArray{itgixSecurityGroup.ID()},
		})

		if createNwIfaceErr != nil {
			return createNwIfaceErr
		}

		// Get ID of latest Amazon Linux AMI
		mostRecent := true
		//ami, amiLookupErr := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
		ami, amiLookupErr := aws.GetAmi(ctx, &aws.GetAmiArgs{
			Filters: []aws.GetAmiFilter{
				{
					Name: "name",
					Values: []string{"amzn-ami-hvm-*-x86_64-ebs"},
				},
			},
			Owners: []string{"137112412989"},
			MostRecent: &mostRecent,
		})

		if amiLookupErr != nil {
			return amiLookupErr
		}

		// Create EC2 instance using - AMI, SG, NetworkInterface, Subnet, VPC
		srv, createEc2InstanceErr := ec2.NewInstance(ctx, "pulumi-test-ec2", &ec2.InstanceArgs{
			Tags:					pulumi.StringMap{"Name": pulumi.String("pulumi-itgix-test"),},
			InstanceType:			pulumi.String("t2.micro"),
			//SubnetId:				itgixSubnet.ID(),
			NetworkInterfaces:		ec2.InstanceNetworkInterfaceArray{
										&ec2.InstanceNetworkInterfaceArgs{
											NetworkInterfaceId: itgixNetworkInterface.ID(),
											DeviceIndex:		pulumi.Int(0),
										},
									},
			VpcSecurityGroupIds:	pulumi.StringArray{itgixSecurityGroup.ID()}, //take sg ID from output
			Ami:					pulumi.String(ami.Id), // take ami ID from lookup
			UserData:				pulumi.String(`#!/bin/bash
												echo "ITgix Pulumi!" > index.html
												nohup python -m SimpleHTTPServer 80 &`),
		})

		if createEc2InstanceErr != nil {
			return createEc2InstanceErr
		}

		ctx.Export("publicIp", srv.PublicIp)
		ctx.Export("publicHostName", srv.PublicDns)

		return nil
	})
}
