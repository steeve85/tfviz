//Replace with preferred AWS Region
aws_region            = "ap-southeast-1" 

// Replace with preferred CIDR/ Subnet and IP
aws_vpc_cidr_block    = "172.16.0.0/16"
aws_subnet_cidr_block = "172.16.10.0/24"
aws_private_ip_fe     = "172.16.10.100"

// Replace with preferred name 
aws_Name              = "aws-ec2-test-ground-application"
aws_Application       = "aws-ec2-test-ground"

//Replace with AMI id from your region
aws_ami               = "ami-07539a31f72d244e7"

//Replace with instance type
aws_instance_type     = "t2.micro"