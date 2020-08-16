package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/steeve85/tfviz/utils"
	"github.com/steeve85/tfviz/aws"
)

var exportFormats = []string{"dot", "jpeg", "pdf", "png"}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
// https://golangcode.com/check-if-element-exists-in-slice/
func Find(slice []string, val string) (int, bool) {
    for i, item := range slice {
        if item == val {
            return i, true
        }
    }
    return -1, false
}

func main() {
	inputFlag := flag.String("input", ".", "Path to Terraform file or directory ")
	outputFlag := flag.String("output", "tfviz.bin", "Path to the exported file")
	formatFlag := flag.String("format", "png", "Format for the output file: dot, jpeg, pdf, png")
	flag.BoolVar(&utils.Ignorewarnings, "Ignorewarnings", false, "Set to ignore warning messages")
	flag.Parse()

	// checking that export format is supported
	_, found := Find(exportFormats, *formatFlag)
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

	tfAws := &aws.AwsTemp{
		Ingress:			make(map[string][]string),
		Egress:				make(map[string][]string),
		CidrVpc:			make(map[string]string),
		CidrSubnet:			make(map[string]string),
	}
	
	err = tfAws.DefaultVpcSubnet(tfModule, graph)
	if err != nil {
		utils.PrintError(err)
	}
	err = tfAws.CreateGraphNodes(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}
	tfAws.PrepareSecurityGroups(tfModule, ctx)

	// DEBUG
	fmt.Println("tfAws.Ingress", tfAws.Ingress)
	fmt.Println("tfAws.Egress", tfAws.Egress)
	fmt.Println("tfAws.CidrVpc", tfAws.CidrVpc)
	fmt.Println("tfAws.CidrSubnet", tfAws.CidrSubnet)

	err = tfAws.CreateGraphEdges(tfModule, ctx, graph)
	if err != nil {
		utils.PrintError(err)
	}
	
	err = utils.ExportGraphToFile(*outputFlag, *formatFlag, graph)
	if err != nil {
		utils.PrintError(err)
	}
}