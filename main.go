package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/steeve85/tfviz/utils"
	"github.com/steeve85/tfviz/aws"
)

var exportFormats = []string{"dot", "jpeg", "pdf", "png"}

func main() {
	inputFlag := flag.String("input", ".", "Path to Terraform file or directory ")
	outputFlag := flag.String("output", "tfviz.bin", "Path to the exported file")
	formatFlag := flag.String("format", "png", "Format for the output file: dot, jpeg, pdf, png")
	flag.BoolVar(&utils.Ignorewarnings, "Ignorewarnings", false, "Set to ignore warning messages")
	flag.Parse()

	// checking that export format is supported
	_, found := utils.Find(exportFormats, *formatFlag)
	if !found {
		fmt.Printf("[ERROR] File format %s is not supported. Quitting...\n", *formatFlag)
		os.Exit(1)
	}

	// check that the export path does not already exist
	if _, err := os.Stat(*outputFlag); err == nil {
		fmt.Printf("[ERROR] File %s already exists. Quitting...\n", *outputFlag)
		os.Exit(1)
	}

	tfModule, err := utils.ParseTFfile(*inputFlag)
	if err != nil {
		// invalid input directory/file
		utils.PrintError(err)
		os.Exit(1)
	}
	ctx := aws.InitiateVariablesAndResources(tfModule)
	graph, err := utils.InitiateGraph()
	if err != nil {
		utils.PrintError(err)
		os.Exit(1)
	}

	/*tfAws := &aws.AwsTemp{
		SecurityGroups:		make(map[string][]string),
		Ingress:			make(map[string][]string),
		Egress:				make(map[string][]string),
		CidrVpc:			make(map[string]string),
		CidrSubnet:			make(map[string]string),
	}*/
	//var tfAws aws.AwsData

	tfAws := &aws.AwsData{
		Vpc:				make(map[string]aws.AwsVpc),
		Subnet:				make(map[string]aws.AwsSubnet),
		Instance:			make(map[string]aws.AwsInstance),
		SecurityGroup:		make(map[string]aws.AwsSecurityGroup),
	}
	
	err = tfAws.CreateDefaultNodes(tfModule, graph)
	if err != nil {
		utils.PrintError(err)
	}

	err = tfAws.ParseTfResources(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}

	err = tfAws.CreateGraphNodes(graph)
	if err != nil {
		utils.PrintError(err)
	}

	//fmt.Println(graph.String())

	err = tfAws.CreateGraphEdges(graph)
	if err != nil {
		utils.PrintError(err)
	}

	fmt.Println(graph.String())
	/*
	err = tfAws.CreateGraphNodes(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}
	tfAws.PrepareSecurityGroups(tfModule, ctx)

	err = tfAws.CreateGraphEdges(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}
	
	err = utils.ExportGraphToFile(*outputFlag, *formatFlag, graph)
	if err != nil {
		utils.PrintError(err)
	}
	*/
}