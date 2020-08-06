variable "aws_region" {
  default = "us-east-1"
}

provider "aws" {
  region = "${var.aws_region}"
}

# Create a VPC to launch our instances into
resource "aws_vpc" "default" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_vpc" "second_VPC" {
  cidr_block = "20.0.0.0/16"
}

resource "aws_subnet" "default1" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
}

resource "aws_subnet" "default2" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.2.0/24"
  map_public_ip_on_launch = true
}

resource "aws_subnet" "second_subnet1" {
  vpc_id                  = "${aws_vpc.second_VPC.id}"
  cidr_block              = "20.0.1.0/24"
  map_public_ip_on_launch = true
}

resource "aws_subnet" "second_subnet2" {
  vpc_id                  = "${aws_vpc.second_VPC.id}"
  cidr_block              = "20.0.2.0/24"
  map_public_ip_on_launch = true
}

resource "aws_instance" "sub_def_1" {
    subnet_id = "${aws_subnet.default1.id}"
}

resource "aws_instance" "sub_def_2" {
    subnet_id = "${aws_subnet.default2.id}"
}

resource "aws_instance" "sub_sec_1" {
    subnet_id = "${aws_subnet.second_subnet1.id}"
}