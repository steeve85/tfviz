package aws

import (
	//"fmt"
	//"net"
	"strings"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/awalterschulze/gographviz"

	"github.com/steeve85/tfviz/utils"
)

/*type AwsTemp struct {
	SecurityGroups		map[string][]string
	Ingress				map[string][]string
	Egress 				map[string][]string
	CidrVpc				map[string]string
	CidrSubnet			map[string]string
}*/

// AwsData is a structure that contain maps of TF parsed resources
type AwsData struct {
	Vpc						map[string]AwsVpc
	Subnet					map[string]AwsSubnet
	Instance				map[string]AwsInstance
	SecurityGroup			map[string]AwsSecurityGroup
}

// AwsVpc is a structure for AWS VPC resources
type AwsVpc struct {
	// The CIDR block for the VPC
	CidrBlock				string `hcl:"cidr_block"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// AwsSubnet is a structure for AWS Subnet resources
type AwsSubnet struct {
	// The CIDR block for the subnet
	CidrBlock				string `hcl:"cidr_block"`
	// The VPC ID
	VpcID					string `hcl:"vpc_id"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
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
	Remain					hcl2.Body `hcl:",remain"`
}

// AwsSecurityGroup is a structure for AWS Security Group resources
type AwsSecurityGroup struct {
	// The VPC ID
	VpcID					*string `hcl:"vpc_id"`
	// A list of ingress rules
	Ingress					[]AwsSGRule `hcl:"ingress,block"` // FIXME make it optional?
	// A list of egress rules
	Egress					[]AwsSGRule `hcl:"egress,block"` // FIXME make it optional?
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// AwsSGRule is a structure for AWS Security Group ingress/egress blocks
type AwsSGRule struct {
	// The start port (or ICMP type number if protocol is "icmp" or "icmpv6")
	FromPort				int `hcl:"from_port"`
	// The end range port (or ICMP code if protocol is "icmp")
	ToPort					int `hcl:"to_port"`
	// If true, the security group itself will be added as a source to this ingress/egress rule
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
	Remain					hcl2.Body `hcl:",remain"`
}

func createDefaultVpc(graph *gographviz.Escape) (error) {
	// Creating default VPC cluster
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
	return nil
}

func createDefaultSubnet(graph *gographviz.Escape, clusterName string) (error) {
	// Creating default Subnet cluster
	err := graph.AddSubGraph(clusterName, "cluster_aws_subnet_default", map[string]string{
		"label": "Subnet: default",
	})
	if err != nil {
		return err
	}

	// Adding invisible node to VPC for links
	err = graph.AddNode("cluster_aws_subnet_default", "aws_subnet_default", map[string]string{
		"shape": "point",
		"style": "invis",
	})
	if err != nil {
		return err
	}
	return nil
}

func createDefaultSecurityGroup(graph *gographviz.Escape) (error) {
	// Creating default security group
	err := graph.AddNode("G", "sg-default", map[string]string{
		"style": "dotted",
		"label": "sg-default",
	})
	if err != nil {
		return err
	}
	return nil
}

func createVpc(graph *gographviz.Escape, vpcName string) (error) {
	// Creating VPC cluster
	err := graph.AddSubGraph("G", "cluster_aws_vpc_"+vpcName, map[string]string{
		"label": "VPC: "+vpcName,
		"style": "rounded",
		"bgcolor": "#EDF1F2",
		"labeljust": "l",
	})
	if err != nil {
		return err
	}

	// Adding invisible node to VPC for links
	err = graph.AddNode("cluster_aws_vpc_"+vpcName, "aws_vpc_"+vpcName, map[string]string{
		"shape": "point",
		"style": "invis",
	})
	if err != nil {
		return err
	}
	return nil
}

func createSubnet(graph *gographviz.Escape, subnetName string, awsSubnet AwsSubnet) (error) {
	// Creating subnet cluster
	err := graph.AddSubGraph("cluster_"+strings.Replace(awsSubnet.VpcID, ".", "_", -1), "cluster_aws_subnet_"+subnetName, map[string]string{
		"label": "Subnet: "+subnetName,
		"style": "rounded",
		"bgcolor": "white",
		"labeljust": "l",
	})
	if err != nil {
		return err
	}
	
	// Adding invisible node to Subnet for links
	err = graph.AddNode("cluster_aws_subnet_"+subnetName, "aws_subnet_"+subnetName, map[string]string{
		"shape": "point",
		"style": "invis",
	})
	if err != nil {
		return err
	}
	return nil
}

func createInstance(graph *gographviz.Escape, instanceName string, awsInstance AwsInstance) (error) {
	// Creating instance node
	var clusterId string
	if awsInstance.SubnetID == nil {
		clusterId = "aws_subnet_default"
	} else {
		clusterId = strings.Replace(*awsInstance.SubnetID, ".", "_", -1)
	}
	err := graph.AddNode("cluster_"+clusterId, "aws_instance_"+instanceName, map[string]string{
		//"style": "filled",
		"label": instanceName,
		//"fontsize": "10",
		"image": "./aws/icons/ec2.png",
		//"imagescale": "true",
		//"fixedsize": "true",
		"shape": "none",
	})
	if err != nil {
		return err
	}
	return nil
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



// CreateDefaultNodes creates default VPC/Subnet/Security Groups if they don't exist in the TF module
func (a *AwsData) CreateDefaultNodes(file *tfconfigs.Module, graph *gographviz.Escape) (error) {
	var vpc, subnet, securityGroup bool
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			vpc = true
		} else if v.Type == "aws_subnet" {
			subnet = true
		} else if v.Type == "aws_security_group" {
			securityGroup = true
		}
	}

	if !vpc {
		// Creating default VPC cluster
		err := createDefaultVpc(graph)
		if err != nil {
			return err
		}
	}

	if !subnet {
		// Creating default subnet cluster
		var clusterName string
		if !vpc {
			clusterName = "cluster_aws_vpc_default"
		} else {
			clusterName = "G"
		}
		err := createDefaultSubnet(graph, clusterName)
		if err != nil {
			return err
		}
	}

	if !securityGroup {
		// Creating default security group
		err := createDefaultSecurityGroup(graph)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AwsData) ParseTfResources(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	for _, v := range file.ManagedResources {
		switch v.Type {
		case "aws_vpc":
			var awsVpc AwsVpc
			diags := gohcl.DecodeBody(v.Config, ctx, &awsVpc)
			utils.PrintDiags(diags)

			// Add AwsVpc to AwsData
			a.Vpc[v.Name] = awsVpc

		case "aws_subnet":
			var awsSubnet AwsSubnet
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSubnet)
			utils.PrintDiags(diags)

			// Add AwsSubnet to AwsData
			a.Subnet[v.Name] = awsSubnet

		case "aws_instance":
			var awsInstance AwsInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsInstance)
			utils.PrintDiags(diags)
			
			// Add AwsInstance to AwsData
			a.Instance[v.Name] = awsInstance

		case "aws_security_group":
			var awsSecurityGroup AwsSecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSecurityGroup)
			utils.PrintDiags(diags)

			// Add AwsSecurityGroup to AwsData
			a.SecurityGroup[v.Name] = awsSecurityGroup
		}
	}

	return nil
}


func (a *AwsData) CreateGraphNodes(graph *gographviz.Escape) (error) {
	// Add VPC clusters to graph
	for vpcName, _ := range a.Vpc {
		err := createVpc(graph, vpcName)
		if err != nil {
			return err
		}
	}

	// Add Subnet clusters to graph
	for subnetName, subnetObj := range a.Subnet {
		err := createSubnet(graph, subnetName, subnetObj)
		if err != nil {
			return err
		}
	}

	// Add Instance nodes to graph
	for instanceName, instanceObj := range a.Instance {
		err := createInstance(graph, instanceName, instanceObj)
		if err != nil {
			return err
		}
	}

	return nil
}


/*
// This function prepares a map of Security Groups
func (a *AwsTemp) PrepareSecurityGroups(file *tfconfigs.Module, ctx *hcl2.EvalContext) {
	// HCL parsing with extrapolated variables
	for _, v := range file.ManagedResources {
		if v.Type == "aws_security_group" {
			var awsSecurityGroup AwsSecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSecurityGroup)
			utils.PrintDiags(diags)

			// SecurityGroups map to link instances with Security Groups
			//instances := make(map[string][]string)
			instances := []string{}
			a.SecurityGroups["aws_security_group." + v.Name] = instances

			// Ingress
			var SGsIngress []string
			for _, ingress := range awsSecurityGroup.Ingress {
				if ingress.CidrBlocks != nil {
					SGsIngress = append(SGsIngress, *ingress.CidrBlocks...)
				}
				if ingress.IPv6CidrBlocks != nil {
					SGsIngress = append(SGsIngress, *ingress.IPv6CidrBlocks...)
				}
				if ingress.SecurityGroups != nil {
					SGsIngress = append(SGsIngress, *ingress.SecurityGroups...)
				}
				if ingress.Self != nil && *ingress.Self == true {
					SGsIngress = append(SGsIngress, "aws_security_group." + v.Name)
				}
			}
			a.Ingress["aws_security_group." + v.Name] = utils.RemoveDuplicateValues(SGsIngress)

			// Egress
			var SGsEgress []string
			for _, egress := range awsSecurityGroup.Egress {
				if egress.CidrBlocks != nil {
					SGsEgress = append(SGsEgress, *egress.CidrBlocks...)
				}
				if egress.IPv6CidrBlocks != nil {
					SGsEgress = append(SGsEgress, *egress.IPv6CidrBlocks...)
				}
				if egress.SecurityGroups != nil {
					SGsEgress = append(SGsEgress, *egress.SecurityGroups...)
				}
				if egress.Self != nil && *egress.Self == true {
					SGsEgress = append(SGsEgress, "aws_security_group." + v.Name)
				}
			}
			a.Egress["aws_security_group." + v.Name] = utils.RemoveDuplicateValues(SGsEgress)
		}
	}
}
*/

/*
func (a *AwsTemp) CreateGraphEdges(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	// Link Instances with their Security Groups
	// FIXME/TODO this is a duplicate of some code below. Need refactor
	defaultSG := false
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
				a.SecurityGroups[sg] = append(a.SecurityGroups[sg], v.Type+"."+v.Name)
			}
		}
	}
	// Create edges based on Security Groups
	// FIXME/TODO this is a duplicate of some code above. Need refactor
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

			if len(SGs) == 0 {
				// This instance has no SG attached and so will inherit from the default SG
				if !defaultSG {
					// Create the default SG node
					err := graph.AddNode("G", "sg-default", map[string]string{
						"style": "dotted",
						"label": "sg-default",
					})
					if err != nil {
						return err
					}
				}
				err := graph.AddEdge("sg-default", v.Type+"_"+v.Name, true, nil)
				if err != nil {
					return err
				}
			}
			// The instance has at least a SG atteched to it
			for _, sg := range SGs {

				for _, source := range a.Ingress[sg] {
					// Highlight Ingress from 0.0.0.0/0 in red
					if source == "0.0.0.0/0" {
						err := graph.AddEdge("Internet", v.Type+"_"+v.Name, true, map[string]string{
							"color": "red",
						})
						if err != nil {
							return err
						}
					} else {
						ipAddrSG, _, err := net.ParseCIDR(source)
						if err != nil {
							// This is not a valid CIDR, so it is probably a Security Group name
							if len(a.SecurityGroups[source]) != 0 {
								// If the SG is found in the TF module, create edges between the nodes that have this SG attached to them
								for _, instanceSrc := range(a.SecurityGroups[source]) {
									instanceSrc = strings.Replace(instanceSrc, ".", "_", -1)
									if instanceSrc != v.Type+"_"+v.Name {
										err = graph.AddEdge(instanceSrc, v.Type+"_"+v.Name, true, nil)
										if err != nil {
											return err
										}
									}
								}
							} else if strings.HasPrefix(source, "sg-") {
								// If the SG is not found in the TF module, it's probably a SG defined outside of TF
								_, found := a.SecurityGroups[source]
								if !found {
									// If the SG is not in a.SecurityGroups, we need to create the Node before the Edges
									err := graph.AddNode("G", source, map[string]string{
										"style": "dotted",
										"label": source,
									})
									if err != nil {
										return err
									}
								}
								// The SG exists, we just need to link it with the appropriate intances
								err = graph.AddEdge(source, v.Type+"_"+v.Name, true, nil)
								if err != nil {
									return err
								}
							} else {
								// Unrecognized SG name
								fmt.Println(err)
							}
						} else {
							// The source is a valid CIDR
							edgeCreated := false
							for cidr, _ := range a.CidrSubnet {
								// Checking for Security Group source IP / Subnet matching
								_, ipNetSubnet, err := net.ParseCIDR(cidr)
								if ipNetSubnet.Contains(ipAddrSG) {
									// the source IP is part of this subnet CIDR
									err = graph.AddEdge(a.CidrSubnet[ipNetSubnet.String()], v.Type+"_"+v.Name, true, nil)
									if err != nil {
										return err
									}
									edgeCreated = true
								}
							}
							if !edgeCreated {
								// Security Group source IP did not matched with Subnet CIDRs
								// Now checking with VPC CIDRs
								for cidr, _ := range a.CidrVpc {
									_, ipNetVpc, err := net.ParseCIDR(cidr)
									if ipNetVpc.Contains(ipAddrSG) {
										// the source IP is part of this VPC CIDR
										err = graph.AddEdge(a.CidrVpc[ipNetVpc.String()], v.Type+"_"+v.Name, true, nil)
										if err != nil {
											return err
										}
										edgeCreated = true
									}
								}
							}
							if !edgeCreated {
								// Security Group source IP did not matched with Subnet and VPC CIDRs
								// Creating a node for the source as it likely to be an undefined IP/CIDR
								err := graph.AddNode("G", source, nil)
								if err != nil {
									return err
								}
								err = graph.AddEdge(source, v.Type+"_"+v.Name, true, nil)
								if err != nil {
									return err
								}
							}
						}
					}
				}
				/*
				for _, source := range a.Egress[sg] {
					err := graph.AddEdge(v.Type+"_"+v.Name, a.Cidr[source], true, map[string]string{
						"style": "dashed",
					})
					if err != nil {
						return err
					}
				}
				*//*
			}
		}
	}
	return nil
}
*/