// Ref: https://gist.github.com/ankyit/d180cdc2843a21204f27473a6c7eeb2c
// Variable Definition
variable "aws_region" {}
variable "aws_vpc_cidr_block" {}
variable "aws_subnet_cidr_block" {}
variable "aws_private_ip_fe" {}
variable "aws_Name" {}
variable "aws_Application" {}
variable "aws_ami" {}
variable "aws_instance_type" {}

// Provider Definition
provider "aws" {
  version = "~> 2.40"
  region  = var.aws_region
}

// Adds a VPC
resource "aws_vpc" "aws_ec2_deployment_test-vpc" {
  cidr_block = var.aws_vpc_cidr_block

  tags = {
    Name        = join("-", [var.aws_Name, "vpc"])
    Application = var.aws_Application
  }
}

//Adds a subnet
resource "aws_subnet" "aws_ec2_deployment_test-subnet" {
  vpc_id            = aws_vpc.aws_ec2_deployment_test-vpc.id
  cidr_block        = var.aws_subnet_cidr_block
  availability_zone = join("", [var.aws_region, "a"])

  tags = {
    Name        = join("-", [var.aws_Name, "subnet"])
    Application = var.aws_Application
  }
}

//Adds a Network Interface
resource "aws_network_interface" "aws_ec2_deployment_test-fe" {
    subnet_id = aws_subnet.aws_ec2_deployment_test-subnet.id
    private_ips = [ var.aws_private_ip_fe ]

    tags = {
    Name        = join("-", [var.aws_Name, "network-interface-fe"])
    Application = var.aws_Application
  }

}
//Adds an EC2 Instance 
resource "aws_instance" "aws_ec2_deployment_test-fe"{
    ami = var.aws_ami
    instance_type = var.aws_instance_type

    network_interface {
        network_interface_id = aws_network_interface.aws_ec2_deployment_test-fe.id
        device_index = 0
    }

    tags = {
    Name        = join("-", [var.aws_Name, "fe-ec2"])
    Application = var.aws_Application
  }
}


// Print Output Values
output "aws_ec2_deployment_test-vpc" {
  description = "CIDR Block for the VPC: "
  value       = aws_vpc.aws_ec2_deployment_test-vpc.cidr_block
}

output "aws_ec2_deployment_test-subnet" {
  description = "Subnet Block: "
  value       = aws_subnet.aws_ec2_deployment_test-subnet.cidr_block
}

output "aws_ec2_deployment_test-private-ip" {
  description = "System Private IP: "
  value       = aws_network_interface.aws_ec2_deployment_test-fe.private_ip
}

output "aws_ec2_deployment_test-EC2-Details" {
  description = "EC2 Details: "
  value       = aws_instance.aws_ec2_deployment_test-fe.public_ip
}