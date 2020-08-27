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
	flag.BoolVar(&utils.Ignorewarnings, "ignorewarnings", false, "Set to ignore warning messages")
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

	fmt.Print("[1/7] ")
	tfModule, err := utils.ParseTFfile(*inputFlag)
	if err != nil {
		// invalid input directory/file
		utils.PrintError(err)
		os.Exit(1)
	}

	fmt.Println("[2/7] Initiating variables and Terraform references")
	ctx := aws.InitiateVariablesAndResources(tfModule)
	graph, err := utils.InitiateGraph()
	if err != nil {
		utils.PrintError(err)
		os.Exit(1)
	}

	tfAws := &aws.Data{
		Vpc:				make(map[string]aws.Vpc),
		Subnet:				make(map[string]aws.Subnet),
		Instance:			make(map[string]aws.Instance),
		SecurityGroup:		make(map[string]aws.SecurityGroup),
		SecurityGroupNodeLinks:		make(map[string][]string),
	}
	
	fmt.Println("[3/7] Creating default nodes (if needed)")
	err = tfAws.CreateDefaultNodes(tfModule, graph)
	if err != nil {
		utils.PrintError(err)
	}

	fmt.Println("[4/7] Parsing TF resources")
	err = tfAws.ParseTfResources(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}

	fmt.Println("[5/7] Creating Graph nodes")
	err = tfAws.CreateGraphNodes(graph)
	if err != nil {
		utils.PrintError(err)
	}

	fmt.Println("[6/7] Creating Graph edges")
	err = tfAws.CreateGraphEdges(graph)
	if err != nil {
		utils.PrintError(err)
	}
	
	fmt.Print("[7/7] ")
	err = utils.ExportGraphToFile(*outputFlag, *formatFlag, graph)
	if err != nil {
		utils.PrintError(err)
	}
	
}