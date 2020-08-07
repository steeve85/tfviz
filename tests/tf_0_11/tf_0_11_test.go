package tf_0_11_test

import (
	"fmt"
	"math/rand"
	"testing"

	//"github.com/awalterschulze/gographviz"
	"github.com/steeve85/tfviz/aws"
	"github.com/steeve85/tfviz/utils"
)

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

	// TODO 
	tfAws := &aws.AwsTemp{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}
	
	tfAws.CreateGraphNodes(tfModule, ctx, graph)
	tfAws.PrepareSecurityGroups(tfModule, ctx)
	tfAws.CreateGraphEdges(tfModule, ctx, graph)
	
	
	// Checking graph export
	outputPath := fmt.Sprintf("/tmp/tfviz_testing_%d", rand.Intn(100000))
	err = utils.ExportGraphToFile(outputPath, "png", graph)
	if err != nil {
		t.Errorf(err.Error())
	}
}