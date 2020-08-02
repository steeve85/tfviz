package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	//"./aws"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/awalterschulze/gographviz"
)

type AwsVpc struct {
	CidrBlock				string `hcl:"cidr_block"`
	Remain     				hcl2.Body `hcl:",remain"`
}

type AwsSubnet struct {
	CidrBlock				string `hcl:"cidr_block"`
	VpcId					string `hcl:"vpc_id"`
	Remain     				hcl2.Body `hcl:",remain"`
}

type AwsInstance struct {
	InstanceType			string `hcl:"instance_type"`
	AMI						string `hcl:"ami"`
	SecurityGroups			*[]string `hcl:"security_groups"`
	VpcSecurityGroupIDs		*[]string `hcl:"vpc_security_group_ids"`
	SubnetId				*string `hcl:"subnet_id"`
	Remain     				hcl2.Body `hcl:",remain"`
}

type AwsSecurityGroup struct {
	VpcId					*string `hcl:"vpc_id"`
	Ingress					[]Ingress `hcl:"ingress,block"` // FIXME make it optional?
	Remain     				hcl2.Body `hcl:",remain"`
}

type Ingress struct {
	FromPort				int `hcl:"from_port"`
	ToPort					int `hcl:"to_port"`
	Self					*bool `hcl:"self"`
	Protocol				string `hcl:"protocol"`
	CidrBlocks				*[]string `hcl:"cidr_blocks"`
	IPv6CidrBlocks			*[]string `hcl:"ipv6_cidr_blocks"`
	SecurityGroups			*[]string `hcl:"security_groups"`
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
		module, diags := tfparser.LoadConfigDir(configpath)
		printDiags(diags)
		return module, nil
	  default:
		file, diags := tfparser.LoadConfigFile(configpath)
		printDiags(diags)
		module, diags := tfconfigs.NewModule([]*tfconfigs.File{file}, nil)
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
	ctx_variables := make(map[string]cty.Value)
	ctx_aws_vpc := make(map[string]cty.Value)
	ctx_aws_subnet := make(map[string]cty.Value)
	ctx_aws_instance := make(map[string]cty.Value)
	ctx_aws_security_group := make(map[string]cty.Value)

	// Prepare context with TF variables
	for _, v := range file.Variables {
		ctx_variables[v.Name] = v.Default
	}

	// Prepare context with named values to resources
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			ctx_aws_vpc[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_subnet" {
			ctx_aws_subnet[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_instance" {
			ctx_aws_instance[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_security_group" {
			ctx_aws_security_group[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		}
	}
	
	ctx := &hcl2.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(ctx_variables),
			"aws_vpc" : cty.ObjectVal(ctx_aws_vpc),
			"aws_subnet" : cty.ObjectVal(ctx_aws_subnet),
			"aws_instance" : cty.ObjectVal(ctx_aws_instance),
			"aws_security_group" : cty.ObjectVal(ctx_aws_security_group),
		},
	}
	return ctx
}

type Aws struct {
	AwsSecurityGroups 	map[string][]string
	Cidr				map[string]string
}

// This function creates Graphviz nodes from the TF file 
func (a *Aws) createGraphNodes(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			var aws_vpc AwsVpc
			diags := gohcl.DecodeBody(v.Config, ctx, &aws_vpc)
			printDiags(diags)

			a.Cidr[aws_vpc.CidrBlock] = v.Type+"_"+v.Name

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
			var aws_subnet AwsSubnet
			diags := gohcl.DecodeBody(v.Config, ctx, &aws_subnet)
			printDiags(diags)

			a.Cidr[aws_subnet.CidrBlock] = v.Type+"_"+v.Name

			// Creating subnet boxes
			err := graph.AddSubGraph("cluster_"+strings.Replace(aws_subnet.VpcId, ".", "_", -1), "cluster_"+v.Type+"_"+v.Name, map[string]string{
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
			var aws_instance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &aws_instance)
			printDiags(diags)

			// Creating Instance nodes			
			err := graph.AddNode("cluster_"+strings.Replace(*aws_instance.SubnetId, ".", "_", -1), v.Type+"_"+v.Name, map[string]string{
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
func (a *Aws) prepareSecurityGroups(file *tfconfigs.Module, ctx *hcl2.EvalContext) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_security_group" {
			var aws_security_group AwsSecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &aws_security_group)
			printDiags(diags)

			var SGs []string
			for _,sg := range aws_security_group.Ingress {
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

func (a *Aws) createGraphEdges(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		
		if v.Type == "aws_instance" {
			var aws_instance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &aws_instance)
			printDiags(diags)

			// Get Security Groups of the AWS instance
			var SGs []string
			if aws_instance.SecurityGroups != nil {
				SGs = append(SGs, *aws_instance.SecurityGroups...)
			}
			if aws_instance.VpcSecurityGroupIDs != nil {
				SGs = append(SGs, *aws_instance.VpcSecurityGroupIDs...)
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

	// DEBUG
	fmt.Println("DEBUG - CLI args: \nInput:", *inputFlag, "\nOutput:", *outputFlag)
	
	tfModule, _ := parseTFfile(*inputFlag)
	ctx := initiateVariablesAndResources(tfModule)
	graph, _ := initiateGraph()

	tfAws := &Aws{
		AwsSecurityGroups:	make(map[string][]string),
		Cidr:				make(map[string]string),
	}
	
	tfAws.createGraphNodes(tfModule, ctx, graph)
	tfAws.prepareSecurityGroups(tfModule, ctx)
	tfAws.createGraphEdges(tfModule, ctx, graph)

	saveDotFile(*outputFlag, graph)
}