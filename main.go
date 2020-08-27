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
	disableEdge := flag.Bool("disableedges", false, "Set to disable edges (Security Groups rules) on the graph")
	verbose := flag.Bool("verbose", false, "Set to enable verbose output")
	flag.BoolVar(&utils.Ignorewarnings, "ignorewarnings", false, "Set to ignore warning messages")
	flag.BoolVar(&aws.IgnoreIngress, "ignoreingress", false, "Set to ignore ingress rules")
	flag.BoolVar(&aws.IgnoreEgress, "ignoreegress", false, "Set to ignore egress rules")
	flag.Parse()

	// Verbose mode
	if *verbose {
		aws.Verbose = true
		utils.Verbose = true
	}

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

	stepsNb := 7
	if *disableEdge {
		stepsNb--
	}
	fmt.Printf("[1/%d] ", stepsNb)
	tfModule, err := utils.ParseTFfile(*inputFlag)
	if err != nil {
		// invalid input directory/file
		utils.PrintError(err)
		os.Exit(1)
	}

	fmt.Printf("[2/%d] Initiating variables and Terraform references\n", stepsNb)
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

	fmt.Printf("[3/%d] Creating default nodes (if needed)\n", stepsNb)
	err = tfAws.CreateDefaultNodes(tfModule, graph)
	if err != nil {
		utils.PrintError(err)
	}

	fmt.Printf("[4/%d] Parsing TF resources\n", stepsNb)
	err = tfAws.ParseTfResources(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}

	fmt.Printf("[5/%d] Creating Graph nodes\n", stepsNb)
	err = tfAws.CreateGraphNodes(graph)
	if err != nil {
		utils.PrintError(err)
	}

	if !*disableEdge {
		fmt.Printf("[6/%d] Creating Graph edges\n", stepsNb)
		err = tfAws.CreateGraphEdges(graph)
		if err != nil {
			utils.PrintError(err)
		}
	}

	fmt.Printf("[%d/%d] ", stepsNb, stepsNb)
	err = utils.ExportGraphToFile(*outputFlag, *formatFlag, graph)
	if err != nil {
		utils.PrintError(err)
	}
}
