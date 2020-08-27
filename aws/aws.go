package aws

import (
	//"fmt"
	"net"
	"strings"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/awalterschulze/gographviz"

	"github.com/steeve85/tfviz/utils"
)

// Data is a structure that contain maps of TF parsed resources
type Data struct {
	defaultVpc				bool
	defaultSubnet			bool
	defaultSecurityGroup	bool
	Vpc						map[string]Vpc
	Subnet					map[string]Subnet
	Instance				map[string]Instance
	SecurityGroup			map[string]SecurityGroup
	// list of security groups not defined in the TF module
	undefinedSecurityGroups		[]string
	// map of resources linked to a security group
	SecurityGroupNodeLinks	map[string][]string
}

// Vpc is a structure for AWS VPC resources
type Vpc struct {
	// The CIDR block for the VPC
	CidrBlock				string `hcl:"cidr_block"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// Subnet is a structure for AWS Subnet resources
type Subnet struct {
	// The CIDR block for the subnet
	CidrBlock				string `hcl:"cidr_block"`
	// The VPC ID
	VpcID					string `hcl:"vpc_id"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// Instance is a structure for AWS EC2 instances resources
type Instance struct {
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

// SecurityGroup is a structure for AWS Security Group resources
type SecurityGroup struct {
	// The VPC ID
	VpcID					*string `hcl:"vpc_id"`
	// A list of ingress rules
	Ingress					[]SGRule `hcl:"ingress,block"` // FIXME make it optional?
	// A list of egress rules
	Egress					[]SGRule `hcl:"egress,block"` // FIXME make it optional?
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// SGRule is a structure for AWS Security Group ingress/egress blocks
type SGRule struct {
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
	// Create default VPC cluster
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
	// Create default Subnet cluster
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
	// Create default security group
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
	// Create VPC cluster
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

func createSubnet(graph *gographviz.Escape, subnetName string, awsSubnet Subnet) (error) {
	// Create subnet cluster
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

func createInstance(graph *gographviz.Escape, instanceName string, awsInstance Instance) (error) {
	// Create instance node
	var clusterID string
	if awsInstance.SubnetID == nil {
		clusterID = "aws_subnet_default"
	} else {
		clusterID = strings.Replace(*awsInstance.SubnetID, ".", "_", -1)
	}
	err := graph.AddNode("cluster_"+clusterID, "aws_instance_"+instanceName, map[string]string{
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




// InitiateVariablesAndResources parses TF file to create Variables / Obj references for interpolation
func InitiateVariablesAndResources(file *tfconfigs.Module) (*hcl2.EvalContext) {
	// Create map for EvalContext to replace variables names by their values inside HCL file using DecodeBody
	ctxVariables := make(map[string]cty.Value)
	ctxVpc := make(map[string]cty.Value)
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
			ctxVpc[v.Name] = cty.ObjectVal(map[string]cty.Value{
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
			"aws_vpc" : cty.ObjectVal(ctxVpc),
			"aws_subnet" : cty.ObjectVal(ctxAwsSubnet),
			"aws_instance" : cty.ObjectVal(ctxAwsInstance),
			"aws_security_group" : cty.ObjectVal(ctxAwsSecurityGroup),
		},
	}
	return ctx
}



// CreateDefaultNodes creates default VPC/Subnet/Security Groups if they don't exist in the TF module
func (a *Data) CreateDefaultNodes(file *tfconfigs.Module, graph *gographviz.Escape) (error) {
	for _, v := range file.ManagedResources {
		if v.Type == "aws_vpc" {
			a.defaultVpc = true
		} else if v.Type == "aws_subnet" {
			a.defaultSubnet = true
		} else if v.Type == "aws_security_group" {
			a.defaultSecurityGroup = true
		}
	}

	if !a.defaultVpc {
		// Create default VPC cluster
		err := createDefaultVpc(graph)
		if err != nil {
			return err
		}
	}

	if !a.defaultSubnet {
		// Create default subnet cluster
		var clusterName string
		if !a.defaultVpc {
			clusterName = "cluster_aws_vpc_default"
		} else {
			clusterName = "G"
		}
		err := createDefaultSubnet(graph, clusterName)
		if err != nil {
			return err
		}
	}

	if !a.defaultSecurityGroup {
		// Create default security group
		err := createDefaultSecurityGroup(graph)
		if err != nil {
			return err
		}
		a.undefinedSecurityGroups = append(a.undefinedSecurityGroups, "sg-default")
	}
	return nil
}

// ParseTfResources parse the TF file / module to identify resources that will be used later on to create the graph
func (a *Data) ParseTfResources(file *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	for _, v := range file.ManagedResources {
		switch v.Type {
		case "aws_vpc":
			var Vpc Vpc
			diags := gohcl.DecodeBody(v.Config, ctx, &Vpc)
			utils.PrintDiags(diags)

			// Add Vpc to Data
			a.Vpc[v.Name] = Vpc

		case "aws_subnet":
			var awsSubnet Subnet
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSubnet)
			utils.PrintDiags(diags)

			// Add Subnet to Data
			a.Subnet[v.Name] = awsSubnet

		case "aws_instance":
			var awsInstance Instance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsInstance)
			utils.PrintDiags(diags)
			
			// Add Instance to Data
			a.Instance[v.Name] = awsInstance

			// Creating SG - Instance connections to facilitate the edges creation for the graph
			if awsInstance.SecurityGroups != nil {
				for _, sg := range *awsInstance.SecurityGroups {
					_, found := a.SecurityGroupNodeLinks[sg]
					if found {
						a.SecurityGroupNodeLinks[sg] = append(a.SecurityGroupNodeLinks[sg], v.Type+"."+v.Name)
					} else {
						a.SecurityGroupNodeLinks[sg] = []string{v.Type+"."+v.Name}
					}
				}
			}
			if awsInstance.VpcSecurityGroupIDs != nil {
				for _, sg := range *awsInstance.VpcSecurityGroupIDs {
					_, found := a.SecurityGroupNodeLinks[sg]
					if found {
						a.SecurityGroupNodeLinks[sg] = append(a.SecurityGroupNodeLinks[sg], v.Type+"."+v.Name)
					} else {
						a.SecurityGroupNodeLinks[sg] = []string{v.Type+"."+v.Name}
					}
				}
			}

		case "aws_security_group":
			var awsSecurityGroup SecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSecurityGroup)
			utils.PrintDiags(diags)

			// Add SecurityGroup to Data
			a.SecurityGroup["aws_security_group."+v.Name] = awsSecurityGroup
		}
	}

	return nil
}

// CreateGraphNodes creates the nodes for the graph
func (a *Data) CreateGraphNodes(graph *gographviz.Escape) (error) {
	// Add VPC clusters to graph
	for vpcName := range a.Vpc {
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

func createInternetIngressEdge(dst string, graph *gographviz.Escape) (error) {
	// Highlight Ingress from 0.0.0.0/0 in red
	err := graph.AddEdge("Internet", dst, true, map[string]string{
		"color": "red",
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Data) parseIngress(dst string, sgName string, graph *gographviz.Escape) (error) {
	if _, found := a.SecurityGroup[sgName]; !found {
		_, found := utils.Find(a.undefinedSecurityGroups, sgName)
		if !found {
			// If the SG is not in defined in TF, we need to create the Node before the Edges
			err := graph.AddNode("G", sgName, map[string]string{
				"style": "dotted",
				"label": sgName,
			})
			if err != nil {
				return err
			}
			a.undefinedSecurityGroups = append(a.undefinedSecurityGroups, sgName)
		}

		// The SG exists, we just need to link it with the appropriate intances
		err := graph.AddEdge(sgName, dst, true, nil)
		if err != nil {
			return err
		}
	}
	for _, rule := range a.SecurityGroup[sgName].Ingress {
		if *rule.CidrBlocks != nil {
			for _, src := range *rule.CidrBlocks {
				// Special ingress rule for 0.0.0.0/0
				if src == "0.0.0.0/0" {
					err := createInternetIngressEdge(dst, graph)
					if err != nil {
						return err
					}
				} else {
					ipAddrSG, _, err := net.ParseCIDR(src)
					if err != nil {
						// Unrecognized SG name
						utils.PrintError(err)
					} else {
						// The source is a valid CIDR
						edgeCreated := false
						for k, v := range(a.Subnet) {
							// Checking for Security Group source IP / Subnet matching
							_, ipNetSubnet, err := net.ParseCIDR(v.CidrBlock)
							if err != nil {
								return err
							}
							if ipNetSubnet.Contains(ipAddrSG) {
								// the source IP is part of this subnet CIDR
								err = graph.AddEdge("aws_subnet_"+k, dst, true, nil)
								if err != nil {
									return err
								}
								edgeCreated = true
							}
						}

						if !edgeCreated {
							// Security Group source IP did not matched with Subnet CIDRs
							// Now checking with VPC CIDRs
							for k, v := range a.Vpc {
								_, ipNetVpc, err := net.ParseCIDR(v.CidrBlock)
								if err != nil {
									return err
								}
								if ipNetVpc.Contains(ipAddrSG) {
									// the source IP is part of this VPC CIDR
									err = graph.AddEdge("aws_vpc_"+k, dst, true, nil)
									if err != nil {
										return err
									}
									edgeCreated = true
								}
							}
						}

						if !edgeCreated {
							// Security Group source IP did not matched with Subnet and VPC CIDRs
							// Creating a node for the source as it is likely to be an undefined IP/CIDR
							err := graph.AddNode("G", src, nil)
							if err != nil {
								return err
							}
							err = graph.AddEdge(src, dst, true, nil)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}

		// Create edges for all instances linked to Ingress.Self
		if rule.Self != nil && *rule.Self != false {
			for _, v1 := range a.SecurityGroupNodeLinks[sgName] {
				v2 := strings.Replace(v1, ".", "_", -1)
				if v2 != dst {
					err := graph.AddEdge(v2, dst, true, nil)
					if err != nil {
						return err
					}
				}
			}
		}

		// Create edges for all instances linked to Ingress.SecurityGroups
		if rule.SecurityGroups != nil {
			for _, v1 := range *rule.SecurityGroups {
				for _, v2 := range a.SecurityGroupNodeLinks[v1] {
					v3 := strings.Replace(v2, ".", "_", -1)
					if v3 != dst {
						err := graph.AddEdge(v3, dst, true, nil)
						if err != nil {
							return err
						}
					}
				}
			}
			
		}
	}

	return nil
}

// CreateGraphEdges creates edges for the graph
func (a *Data) CreateGraphEdges(graph *gographviz.Escape) (error) {
	// Link Instances with their Security Groups
	for instanceName, instanceObj := range a.Instance {

		// Get the Security Groups of the AWS instance
		var SGs []string
		if instanceObj.SecurityGroups != nil {
			SGs = append(SGs, *instanceObj.SecurityGroups...)
		}
		if instanceObj.VpcSecurityGroupIDs != nil {
			SGs = append(SGs, *instanceObj.VpcSecurityGroupIDs...)
		}

		// This instance has no SG attached and so will inherit from the default SG
		if len(SGs) == 0 {
			_, found := utils.Find(a.undefinedSecurityGroups, "sg-default")
			if !found {
				// Create default security group
				err := createDefaultSecurityGroup(graph)
				if err != nil {
					return err
				}
				a.undefinedSecurityGroups = append(a.undefinedSecurityGroups, "sg-default")
			}
			err := graph.AddEdge("sg-default", "aws_instance_"+instanceName, true, nil)
			if err != nil {
				return err
			}
		}
		// The instance has at least one SG attached to it
		for _, sg := range SGs {

			// sg should looks like 'aws_security_group.SG_NAME'
			//sgName := strings.Split(sg, ".")[1]
			//fmt.Println("\t", sg)//, "->", sgName)

			//fmt.Println("\tdetails:", a.SecurityGroup[sgName])
			//fmt.Println("\t# egress:", len(a.SecurityGroup[sgName].Egress))
			//fmt.Println("\t# ingress:", len(a.SecurityGroup[sgName].Ingress))
			
			//parseSGRules(graph, "aws_instance_"+instanceName, instanceObj, a.SecurityGroup[sgName])

			// Parse Ingress SG rules
			//a.parseIngress("aws_instance_"+instanceName, a.SecurityGroup[sgName].Ingress, graph)
			a.parseIngress("aws_instance_"+instanceName, sg, graph)

			// Parse Egress SG rules
			//TODO
		}
	}

	return nil
}
