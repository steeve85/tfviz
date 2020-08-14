package aws

import (
	"strings"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/awalterschulze/gographviz"

	"github.com/steeve85/tfviz/utils"
)

type AwsTemp struct {
	AwsSecurityGroups 	map[string][]string
	Cidr				map[string]string
}

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

func InitiateVariablesAndResources(file *tfconfigs.Module) (*hcl2.EvalContext) {
	// Create map for EvalContext to replace variables names by their values inside HCL file using DecodeBody
	ctxVariables := make(map[string]cty.Value)
	ctxAwsVpc := make(map[string]cty.Value)
	ctxAwsSubnet := make(map[string]cty.Value)
	ctxAwsInstance := make(map[string]cty.Value)
	ctxAwsSecurityGroup := make(map[string]cty.Value)

	// Prepare context with TF variables
	for _, v := range file.Variables {
		// Handling the case there is no default value for the variable
		if v.Default.IsNull() {
			ctxVariables[v.Name] = cty.StringVal("var_" + v.Name)
		} else {
			ctxVariables[v.Name] = v.Default
		}
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

// This function creates default VPC/Subnet if they don't exist
func (a *AwsTemp) DefaultVpcSubnet(file *tfconfigs.Module, graph *gographviz.Escape) (error) {
	var vpc, subnet bool
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			vpc = true
		} else if v.Type == "aws_subnet" {
			subnet = true
		}
	}

	if !vpc {
		// Creating default VPC boxe
		err := graph.AddSubGraph("G", "cluster_aws_vpc_default", map[string]string{
			"label": "VPC: default",
		})
		if err != nil {
			return err
		}
		// Adding invisible node to VPC for links
		err = graph.AddNode("cluster_aws_vpc_default", "aws_vpc_default", map[string]string{
			"shape": "point",
			"style": "invis",
		})
		if err != nil {
			return err
		}
	}
	if !subnet {
		// Creating default subnet boxe
		var clusterName string
		if !vpc {
			clusterName = "cluster_aws_vpc_default"
		} else {
			clusterName = "G"
		}
		err := graph.AddSubGraph(clusterName, "cluster_aws_subnet_default", map[string]string{
			"label": "Subnet: default",
		})
		if err != nil {
			return err
		}

		// Adding invisible node to VPC for links
		err = graph.AddNode("cluster_aws_vpc_default", "aws_subnet_default", map[string]string{
			"shape": "point",
			"style": "invis",
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// This function creates Graphviz nodes from the TF file 
func (a *AwsTemp) CreateGraphNodes(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// Setting CIDR for public network (Internet)
	a.Cidr["0.0.0.0/0"] = "Internet"
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			var awsVpc AwsVpc
			diags := gohcl.DecodeBody(v.Config, ctx, &awsVpc)
			utils.PrintDiags(diags)

			a.Cidr[awsVpc.CidrBlock] = v.Type+"_"+v.Name

			// Creating VPC boxes
			err := graph.AddSubGraph("G", "cluster_"+v.Type+"_"+v.Name, map[string]string{
				"label": "VPC: "+v.Name,
			})
			if err != nil {
				return err
			}

			// Adding invisible node to VPC for links
			err = graph.AddNode("cluster_"+v.Type+"_"+v.Name, v.Type+"_"+v.Name, map[string]string{
				"shape": "point",
				"style": "invis",
			})
			if err != nil {
				return err
			}
		} else if v.Type == "aws_subnet" {
			var awsSubnet AwsSubnet
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSubnet)
			utils.PrintDiags(diags)

			a.Cidr[awsSubnet.CidrBlock] = v.Type+"_"+v.Name

			// Creating subnet boxes
			err := graph.AddSubGraph("cluster_"+strings.Replace(awsSubnet.VpcID, ".", "_", -1), "cluster_"+v.Type+"_"+v.Name, map[string]string{
				"label": "Subnet: "+v.Name,
			})
			if err != nil {
				return err
			}
			
			// Adding invisible node to VPC for links
			err = graph.AddNode("cluster_"+v.Type+"_"+v.Name, v.Type+"_"+v.Name, map[string]string{
				"shape": "point",
				"style": "invis",
			})
			if err != nil {
				return err
			}
		} else if v.Type == "aws_instance" {
			var awsInstance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsInstance)
			utils.PrintDiags(diags)

			// Creating Instance nodes
			var clusterId string
			if awsInstance.SubnetID == nil {
				clusterId = "aws_subnet_default"
			} else {
				clusterId = strings.Replace(*awsInstance.SubnetID, ".", "_", -1)
			}
			err := graph.AddNode("cluster_"+clusterId, v.Type+"_"+v.Name, map[string]string{
				"style": "filled",
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// This function prepares a map of Security Groups
func (a *AwsTemp) PrepareSecurityGroups(file *tfconfigs.Module, ctx *hcl2.EvalContext) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_security_group" {
			var awsSecurityGroup AwsSecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSecurityGroup)
			utils.PrintDiags(diags)

			var SGs []string
			for _, ingress := range awsSecurityGroup.Ingress {
				if ingress.CidrBlocks != nil {
					SGs = append(SGs, *ingress.CidrBlocks...)
				}
				if ingress.IPv6CidrBlocks != nil {
					SGs = append(SGs, *ingress.IPv6CidrBlocks...)
				}
				if ingress.SecurityGroups != nil {
					SGs = append(SGs, *ingress.SecurityGroups...)
				}
			}
			a.AwsSecurityGroups["aws_security_group." + v.Name] = SGs
		}
	}
}

func (a *AwsTemp) CreateGraphEdges(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		
		if v.Type == "aws_instance" {
			var awsInstance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsInstance)
			utils.PrintDiags(diags)

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
						return err
					}
				}	
			}
		}
	}
	return nil
}