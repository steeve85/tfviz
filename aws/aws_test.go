package aws

import (
	"testing"
	"github.com/awalterschulze/gographviz"
	"github.com/steeve85/tfviz/utils"
)

func TestInitiateVariablesAndResources(t *testing.T) {
	// Checking TF file parsing
	tfModule, err := utils.ParseTFfile("../utils/testdata/main.tf")
	if err != nil {
		t.Errorf(err.Error())
	}
	
	// Checking TF variables
	ctx := InitiateVariablesAndResources(tfModule)
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
}

func TestDefaultVpcSubnet(t *testing.T) {
	// Checking TF file parsing
	tfModule, err := utils.ParseTFfile("../utils/testdata/main.tf")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Graph initialization
	graph := gographviz.NewEscape()
	graph.SetName("G")
	graph.SetDir(false)

	tfAws := &AwsTemp{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}

	err = tfAws.DefaultVpcSubnet(tfModule, graph)
	if err != nil {
		t.Errorf(err.Error())
	}

	if len(graph.Nodes.Nodes) != 0 {
		t.Errorf("DefaultVpcSubnet: Incorrect number of nodes")
	}
}

func TestCreateGraphNodes(t *testing.T) {
	// Checking TF file parsing
	tfModule, err := utils.ParseTFfile("../utils/testdata/main.tf")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Graph initialization
	graph := gographviz.NewEscape()
	graph.SetName("G")
	graph.SetDir(false)

	tfAws := &AwsTemp{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}

	err = tfAws.DefaultVpcSubnet(tfModule, graph)
	if err != nil {
		t.Errorf(err.Error())
	}

	// Checking TF variables
	ctx := InitiateVariablesAndResources(tfModule)

	err = tfAws.CreateGraphNodes(tfModule, ctx, graph)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(graph.Nodes.Nodes) != 9 {
		t.Errorf("CreateGraphNodes: Incorrect number of nodes")
	}
}