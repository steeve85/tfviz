package tf_0_11_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/steeve85/tfviz/aws"
	"github.com/steeve85/tfviz/utils"
)

func TestAwsEc2Instance(t *testing.T) {
	// Checking TF file parsing
	tfModule, err := utils.ParseTFfile("aws-ec2-instance/")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Checking TF variables
	ctx := aws.InitiateVariablesAndResources(tfModule)
	if len(ctx.Variables["var"].AsValueMap()) != 4 {
		t.Errorf("Incorrect number of TF variables")
	}
	if ctx.Variables["var"].AsValueMap()["aws_region"].AsString() != "us-east-1" {
		t.Errorf("Incorrect value for var.aws_region")
	}
	if ctx.Variables["var"].AsValueMap()["ami_id"].AsString() != "ami-2e1ef954" {
		t.Errorf("Incorrect value for var.ami_id")
	}
	if ctx.Variables["var"].AsValueMap()["instance_type"].AsString() != "t2.micro" {
		t.Errorf("Incorrect value for var.instance_type")
	}
	if len(ctx.Variables["aws_instance"].AsValueMap()) != 1 {
		t.Errorf("Incorrect number of aws_instance")
	}

	// Checking graph initialization
	graph, err := utils.InitiateGraph()
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(graph.Nodes.Nodes) != 1 {
		t.Errorf("Incorrect number of nodes")
	}

	tfAws := &aws.AwsTemp{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}

	if len(tfModule.ManagedResources) != 1 {
		t.Errorf("Incorrect number of TF resources")
	}

	tfAws.DefaultVpcSubnet(tfModule, graph)
	tfAws.CreateGraphNodes(tfModule, ctx, graph)
	
	if len(graph.Nodes.Nodes) != 4 {
		t.Errorf("CreateGraphNodes: Incorrect number of nodes")
	}

	// Checking graph export
	outputPath := fmt.Sprintf("/tmp/tfviz_TestAwsEc2Instance_%d.png", rand.Intn(100000))
	err = utils.ExportGraphToFile(outputPath, "png", graph)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestVpcSubnetEc2(t *testing.T) {
	// Checking TF file parsing
	tfModule, err := utils.ParseTFfile("vpc-subnet-ec2/main.tf")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Checking TF variables
	ctx := aws.InitiateVariablesAndResources(tfModule)
	if len(ctx.Variables["var"].AsValueMap()) != 1 {
		t.Errorf("Incorrect number of TF variables")
	}
	if ctx.Variables["var"].AsValueMap()["aws_region"].AsString() != "us-east-1" {
		t.Errorf("Incorrect value for var.aws_region")
	}
	if len(ctx.Variables["aws_vpc"].AsValueMap()) != 2 {
		t.Errorf("Incorrect number of aws_vpc")
	}
	if len(ctx.Variables["aws_subnet"].AsValueMap()) != 4 {
		t.Errorf("Incorrect number of aws_subnet")
	}
	if len(ctx.Variables["aws_instance"].AsValueMap()) != 3 {
		t.Errorf("Incorrect number of aws_instance")
	}

	// Checking graph initialization
	graph, err := utils.InitiateGraph()
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(graph.Nodes.Nodes) != 1 {
		t.Errorf("Incorrect number of nodes")
	}

	tfAws := &aws.AwsTemp{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}

	if len(tfModule.ManagedResources) != 9 {
		t.Errorf("Incorrect number of TF resources")
	}

	tfAws.CreateGraphNodes(tfModule, ctx, graph)
	if len(graph.Nodes.Nodes) != 10 {
		t.Errorf("CreateGraphNodes: Incorrect number of nodes")
	}

	// Checking graph export
	outputPath := fmt.Sprintf("/tmp/tfviz_TestVpcSubnetEc2_%d.png", rand.Intn(100000))
	err = utils.ExportGraphToFile(outputPath, "png", graph)
	if err != nil {
		t.Errorf(err.Error())
	}
}