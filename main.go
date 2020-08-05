package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/awalterschulze/gographviz"
)

// AwsVpc is a structure for AWS VPC resources
type AwsVpc struct {
	// The CIDR block for the VPC
	CidrBlock				string `hcl:"cidr_block"`
	// Other arguments
	Remain     				hcl2.Body `hcl:",remain"`
}

// AwsSubnet is a structure for AWS Subnet resources
type AwsSubnet struct {
	// The CIDR block for the subnet
	CidrBlock				string `hcl:"cidr_block"`
	// The VPC ID
	VpcID					string `hcl:"vpc_id"`
	// Other arguments
	Remain     				hcl2.Body `hcl:",remain"`
}

// AwsInstance is a structure for AWS EC2 instances resources
type AwsInstance struct {
	// The type of instance to start
	InstanceType			string `hcl:"instance_type"`
	// The AMI to use for the instance
	AMI						string `hcl:"ami"`
	// A list of security group names (EC2-Classic) or IDs (default VPC) to associate with
	SecurityGroups			*[]string `hcl:"security_groups"`
	// A list of security group IDs to associate with (VPC only)
	VpcSecurityGroupIDs		*[]string `hcl:"vpc_security_group_ids"`
	// The VPC Subnet ID to launch in
	SubnetID				*string `hcl:"subnet_id"`
	// Other arguments
	Remain     				hcl2.Body `hcl:",remain"`
}

// AwsSecurityGroup is a structure for AWS Security Group resources
type AwsSecurityGroup struct {
	// The VPC ID
	VpcID					*string `hcl:"vpc_id"`
	// A list of ingress rules
	Ingress					[]AwsIngress `hcl:"ingress,block"` // FIXME make it optional?
	// Other arguments
	Remain     				hcl2.Body `hcl:",remain"`
}

// AwsIngress is a structure for AWS Security Group ingress blocks
type AwsIngress struct {
	// The start port (or ICMP type number if protocol is "icmp" or "icmpv6")
	FromPort				int `hcl:"from_port"`
	// The end range port (or ICMP code if protocol is "icmp")
	ToPort					int `hcl:"to_port"`
	// If true, the security group itself will be added as a source to this ingress rule
	Self					*bool `hcl:"self"`
	// The protocol.  icmp, icmpv6, tcp, udp, "-1" (all)
	Protocol				string `hcl:"protocol"`
	// List of CIDR blocks
	CidrBlocks				*[]string `hcl:"cidr_blocks"`
	// List of IPv6 CIDR blocks
	IPv6CidrBlocks			*[]string `hcl:"ipv6_cidr_blocks"`
	// List of security group Group Names if using EC2-Classic, or Group IDs if using a VPC
	SecurityGroups			*[]string `hcl:"security_groups"`
	// Other arguments
	Remain     				hcl2.Body `hcl:",remain"`
}

func printError(err error) {
	e := fmt.Errorf("[ERROR] %s", err)
	fmt.Println(e.Error())
}

func printDiags(diags hcl2.Diagnostics) {
	if len(diags) == 1 {
		fmt.Println("[WARNING] Diagnostics:", diags[0].Error())
	} else if len(diags) > 1 {
		fmt.Println("[WARNING] Diagnostics:")
		for _, d := range diags {
			fmt.Println("\t", d.Error())
		}
	}
}

func saveDotFile(dotfile string, graph *gographviz.Escape) error {
	fmt.Println("Exporting Graph to", dotfile, "DOT file.")
	err := ioutil.WriteFile(dotfile, []byte(graph.String()), 0644)
	return err
}

func parseTFfile(configpath string) (*tfconfigs.Module, error) {
	f, err := os.Stat(configpath);
	if err != nil {
		printError(err)
		return nil, err
	}

	tfparser := tfconfigs.NewParser(nil)
	
	switch {
	  case f.IsDir():
		fmt.Println("Parsing", configpath, "Terraform module...")
		if tfparser.IsConfigDir(configpath) == false {
			err := fmt.Errorf("[ERROR] Directory %s does not contain valid Terraform configuration files", configpath)
			fmt.Println(err.Error())
			return nil, err
		}
		module, diags := tfparser.LoadConfigDir(configpath)
		printDiags(diags)
		return module, nil
	  default:
		fmt.Println("Parsing", configpath, "Terraform file...")
		file, diags := tfparser.LoadConfigFile(configpath)
		// Return error if the TF file doesn't contain resources
		if len(file.ManagedResources) == 0 {
			err := fmt.Errorf("[ERROR] File %s does not contain valid Terraform configuration", configpath)
			fmt.Println(err.Error())
			return nil, err
		}
		module, moreDiags := tfconfigs.NewModule([]*tfconfigs.File{file}, nil)
		diags = append(diags, moreDiags...)
		printDiags(diags)
		return module, nil
	}
}

func initiateGraph() (*gographviz.Escape, error) {
	// Graph initialization
	g := gographviz.NewEscape()
	g.SetName("G")
	g.SetDir(false)

	// Adding node for Internet representation
	err := g.AddNode("G", "Internet", map[string]string{
		"shape": "octagon",
	})
	if err != nil {
		printError(err)
		return nil, err
	}
	return g, nil
}

func initiateVariablesAndResources(file *tfconfigs.Module) (*hcl2.EvalContext) {
	// Create map for EvalContext to replace variables names by their values inside HCL file using DecodeBody
	ctxVariables := make(map[string]cty.Value)
	ctxAwsVpc := make(map[string]cty.Value)
	ctxAwsSubnet := make(map[string]cty.Value)
	ctxAwsInstance := make(map[string]cty.Value)
	ctxAwsSecurityGroup := make(map[string]cty.Value)

	// Prepare context with TF variables
	for _, v := range file.Variables {
		ctxVariables[v.Name] = v.Default
	}

	// Prepare context with named values to resources
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			ctxAwsVpc[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_subnet" {
			ctxAwsSubnet[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_instance" {
			ctxAwsInstance[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_security_group" {
			ctxAwsSecurityGroup[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		}
	}
	
	ctx := &hcl2.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(ctxVariables),
			"aws_vpc" : cty.ObjectVal(ctxAwsVpc),
			"aws_subnet" : cty.ObjectVal(ctxAwsSubnet),
			"aws_instance" : cty.ObjectVal(ctxAwsInstance),
			"aws_security_group" : cty.ObjectVal(ctxAwsSecurityGroup),
		},
	}
	return ctx
}

type aws struct {
	AwsSecurityGroups 	map[string][]string
	Cidr				map[string]string
}

// This function creates Graphviz nodes from the TF file 
func (a *aws) createGraphNodes(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// Setting CIDR for public network (Internet)
	a.Cidr["0.0.0.0/0"] = "Internet"
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			var awsVpc AwsVpc
			diags := gohcl.DecodeBody(v.Config, ctx, &awsVpc)
			printDiags(diags)

			a.Cidr[awsVpc.CidrBlock] = v.Type+"_"+v.Name

			// Creating VPC boxes
			err := graph.AddSubGraph("G", "cluster_"+v.Type+"_"+v.Name, map[string]string{
				"label": "VPC: "+v.Name,
			})
			if err != nil {
				printError(err)
				return err
			}

			// Adding invisible node to VPC for links
			err = graph.AddNode("cluster_"+v.Type+"_"+v.Name, v.Type+"_"+v.Name, map[string]string{
				"shape": "point",
				"style": "invis",
			})
			if err != nil {
				printError(err)
				return err
			}
		} else if v.Type == "aws_subnet" {
			var awsSubnet AwsSubnet
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSubnet)
			printDiags(diags)

			a.Cidr[awsSubnet.CidrBlock] = v.Type+"_"+v.Name

			// Creating subnet boxes
			err := graph.AddSubGraph("cluster_"+strings.Replace(awsSubnet.VpcID, ".", "_", -1), "cluster_"+v.Type+"_"+v.Name, map[string]string{
				"label": "Subnet: "+v.Name,
			})
			if err != nil {
				printError(err)
				return err
			}
			
			// Adding invisible node to VPC for links
			err = graph.AddNode("cluster_"+v.Type+"_"+v.Name, v.Type+"_"+v.Name, map[string]string{
				"shape": "point",
				"style": "invis",
			})
			if err != nil {
				printError(err)
				return err
			}
		} else if v.Type == "aws_instance" {
			var awsInstance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsInstance)
			printDiags(diags)

			// Creating Instance nodes			
			err := graph.AddNode("cluster_"+strings.Replace(*awsInstance.SubnetID, ".", "_", -1), v.Type+"_"+v.Name, map[string]string{
				"style": "filled",
			})
			if err != nil {
				printError(err)
				return err
			}
		}
	}
	return nil
}

// This function prepares a map of Security Groups
func (a *aws) prepareSecurityGroups(file *tfconfigs.Module, ctx *hcl2.EvalContext) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_security_group" {
			var awsSecurityGroup AwsSecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSecurityGroup)
			printDiags(diags)

			var SGs []string
			for _,sg := range awsSecurityGroup.Ingress {
				if sg.CidrBlocks != nil {
					SGs = append(SGs, *sg.CidrBlocks...)
				}
				if sg.IPv6CidrBlocks != nil {
					SGs = append(SGs, *sg.IPv6CidrBlocks...)
				}
				if sg.SecurityGroups != nil {
					SGs = append(SGs, *sg.SecurityGroups...)
				}
			}
			a.AwsSecurityGroups["aws_security_group." + v.Name] = SGs
		}
	}
}

func (a *aws) createGraphEdges(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		
		if v.Type == "aws_instance" {
			var awsInstance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsInstance)
			printDiags(diags)

			// Get Security Groups of the AWS instance
			var SGs []string
			if awsInstance.SecurityGroups != nil {
				SGs = append(SGs, *awsInstance.SecurityGroups...)
			}
			if awsInstance.VpcSecurityGroupIDs != nil {
				SGs = append(SGs, *awsInstance.VpcSecurityGroupIDs...)
			}

			for _, sg := range SGs {
				for _, cidr := range a.AwsSecurityGroups[sg] {
					err := graph.AddEdge(a.Cidr[cidr], v.Type+"_"+v.Name, false, nil)
					if err != nil {
						printError(err)
						return err
					}
				}	
			}
		}
	}
	return nil
}


func main() {
	inputFlag := flag.String("input", ".", "Path to Terraform file or directory ")
	outputFlag := flag.String("output", "tfviz.dot", "Path to the exported Dot file")
	flag.Parse()

	// if output file exists, exit
	if _, err := os.Stat(*outputFlag); err == nil {
		fmt.Printf("[ERROR] File %s already exists. Quitting...\n", *outputFlag)
		os.Exit(1)
	}

	tfModule, err := parseTFfile(*inputFlag)
	if err != nil {
		// invalid input directory/file
		os.Exit(1)
	}
	ctx := initiateVariablesAndResources(tfModule)
	graph, err := initiateGraph()
	if err != nil {
		os.Exit(1)
	}

	tfAws := &aws{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}
	
	tfAws.createGraphNodes(tfModule, ctx, graph)
	tfAws.prepareSecurityGroups(tfModule, ctx)
	tfAws.createGraphEdges(tfModule, ctx, graph)
	
	saveDotFile(*outputFlag, graph)
}