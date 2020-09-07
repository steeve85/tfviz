package aws

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"
	"strings"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/awalterschulze/gographviz"

	"github.com/steeve85/tfviz/utils"
)


// IgnoreIngress can be used to not create edges for Ingress rules
var IgnoreIngress bool

// IgnoreEgress can be used to not create edges for Egress rules
var IgnoreEgress bool

// Verbose enables verbose mode if set to true
var Verbose bool

// Defining values for ingress / egress rules
const ingressRule = 1
const egressRule = 2

// Data is a structure that contain maps of TF parsed resources
type Data struct {
	defaultVpc				bool
	defaultSubnet			bool
	defaultSecurityGroup	bool
	Vpc						map[string]Vpc
	Subnet					map[string]Subnet
	Instance				map[string]Instance
	DBInstance				map[string]DBInstance
	DBSubnetGroup 			map[string]DBSubnetGroup
	SecurityGroup			map[string]SecurityGroup
	LB						map[string]LB
	ASG						map[string]AutoscalingGroup
	// list of security groups not defined in the TF module
	undefinedSecurityGroups		[]string
	// map of resources linked to a security group
	SecurityGroupNodeLinks	map[string][]string
	// list of unsupported resources
	unsupportedResources	[]string
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

// Instance is a structure for AWS EC2 instance resources
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

// DBInstance is a structure for AWS RDS instance resources
type DBInstance struct {
	// The allocated storage in Gb
	AllocatedStorage		*int `hcl:"allocated_storage"`
	// Name of DB subnet group
	DBSubnetGroupName 		*string `hcl:"db_subnet_group_name"`
	// The database engine to use
	Engine					*string `hcl:"engine"`
	// The instance type of the RDS instance
	InstanceClass			*string `hcl:"instance_class"`
	// Password for the master DB user
	Password				*string `hcl:"password"`
	// Bool to control if instance is publicly accessible
	PubliclyAccessible		*bool `hcl:"publicly_accessible"`
	// Username for the master DB user
	Username				*string `hcl:"username"`
	// List of VPC security groups to associate
	VpcSecurityGroupIDs		*[]string `hcl:"vpc_security_group_ids"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// DBSubnetGroup is a structure for RDS DB subnet group resources
type DBSubnetGroup struct {
	// A list of VPC subnet IDs
	SubnetIDs				[]string `hcl:"subnet_ids"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// LB is a structure for Load Balancer resources
type LB struct {
	// The type of load balancer to create. Possible values are 'application' or 'network'
	LoadBalancerType		*string `hcl:"load_balancer_type"`
	// A list of security group IDs to assign to the LB. Only valid for Load Balancers of type 'application'
	SecurityGroups			*[]string `hcl:"security_groups"`
	// A list of subnet IDs to attach to the LB
	Subnets					*[]string `hcl:"subnets"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// LBListener is a structure for Load Balancer Listener resources
type LBListener struct {
	// The ARN of the load balancer
	LoadBalancerARN			string `hcl:"load_balancer_arn"`
	// An Action block
	DefaultAction			LBAction `hcl:"default_action,block"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// LBAction is used to define Action blocks for Load Balancers
type LBAction struct {
	// The type of routing action
	Type					string `hcl:"type"`
	// The ARN of the Target Group to which to route traffic
	TargetGroupARN			string `hcl:"target_group_arn"`
	// TODO: might add 'forward' field and block later
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// LBTargetGroup is used to define a Target Group resource for use with Load Balancer resources
type LBTargetGroup struct {
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}


// LBTargetGroupAttachment is used to have the ability to register instances and containers with an Application Load Balancer (ALB) 
type LBTargetGroupAttachment struct {
	// The ARN of the target group with which to register targets
	TargetGroupArn			string `hcl:"target_group_arn"`
	// The ID of the target
	TargetID				string `hcl:"target_id"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// AutoscalingAttachment is used to define an AutoScaling Attachment resource
type AutoscalingAttachment struct {
	// Name of ASG to associate with the ELB
	AutoscalingGroupName	string `hcl:"autoscaling_group_name"`
	// The name of the ELB
	Elb						*string `hcl:"elb"`
	// The ARN of an ALB Target Group
	AlbTargetGroupArn		*string `hcl:"alb_target_group_arn"`
}

// LaunchConfiguration structure is used to create a new launch configuration, used for autoscaling groups
type LaunchConfiguration struct {
	// The size of instance to launch
	InstanceType			string `hcl:"instance_type"`
	// The EC2 image ID to launch
	ImageID						string `hcl:"image_id"`
	// A list of associated security group IDS
	SecurityGroups			*[]string `hcl:"security_groups"`
	// Other arguments
	Remain					hcl2.Body `hcl:",remain"`
}

// AutoscalingGroup is used to define an AutoScaling Group resource
type AutoscalingGroup struct {
	// The maximum size of the auto scale group
	MaxSize					int `hcl:"max_size"`
	// The minimum size of the auto scale group
	MinSize					int `hcl:"min_size"`
	// The name of the launch configuration to use
	LaunchConfiguration		*string `hcl:"launch_configuration"`
	// A set of aws_alb_target_group ARNs, for use with Application or Network Load Balancing
	TargetGroupArns			*[]string `hcl:"target_group_arns"`
	// A list of subnet IDs to launch resources in
	VPCZoneIdentifier		*[]string `hcl:"vpc_zone_identifier"`
	// TODO:
	// Implement 'launch_template' and 'mixed_instances_policy'
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
	if Verbose == true {
		fmt.Println("[VERBOSE] AddSubGraph: cluster_aws_vpc_default // Create Default VPC")
	}
	err := graph.AddSubGraph("G", "cluster_aws_vpc_default", map[string]string{
		"label": "VPC: default",
	})
	if err != nil {
		return err
	}
	// Adding invisible node to VPC for links
	if Verbose == true {
		fmt.Println("[VERBOSE] AddNode: aws_vpc_default to cluster_aws_vpc_default")
	}
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
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: cluster_aws_subnet_default to %s // Create Default Subnet\n", clusterName)
	}
	err := graph.AddSubGraph(clusterName, "cluster_aws_subnet_default", map[string]string{
		"label": "Subnet: default",
	})
	if err != nil {
		return err
	}

	// Adding invisible node to VPC for links
	if Verbose == true {
		fmt.Println("[VERBOSE] AddNode: aws_subnet_default to cluster_aws_subnet_default")
	}
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
	if Verbose == true {
		fmt.Println("[VERBOSE] AddNode: sg-default to G // Create default Security Group")
	}
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
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddSubGraph: cluster_aws_vpc_%s to G // Create VPC\n", vpcName)
	}
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
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: aws_vpc_%s to cluster_aws_vpc_%s\n", vpcName, vpcName)
	}
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
	vpcID := strings.Replace(awsSubnet.VpcID, ".", "_", -1)
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddSubGraph: cluster_aws_subnet_%s to cluster_%s // Create Subnet\n", subnetName, vpcID)
	}
	err := graph.AddSubGraph("cluster_"+vpcID, "cluster_aws_subnet_"+subnetName, map[string]string{
		"label": "Subnet: "+subnetName,
		"style": "rounded",
		"bgcolor": "white",
		"labeljust": "l",
	})
	if err != nil {
		return err
	}

	// Adding invisible node to Subnet for links
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: aws_subnet_%s to cluster_aws_subnet_%s\n", subnetName, subnetName)
	}
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
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: aws_instance_%s to cluster_%s // Create Instance\n", instanceName, clusterID)
	}

	// Splitting label if more than 8 chars
	re := regexp.MustCompile(`(\S{8})`)
	labelName := strings.Join(re.FindAllString(instanceName, -1), "\\n")
	// If instanceName is shorter than 8, labelName needs to be manually set, otherwise the string will be empty
	if labelName == "" {
		labelName = instanceName
	}

	err := graph.AddNode("cluster_"+clusterID, "aws_instance_"+instanceName, map[string]string{
		"label": labelName,
		"image": "./aws/icons/ec2.png",
		"width": "1",
		"height": "1",
		"fixedsize": "true",
		"shape": "none",
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Data) createDBInstance(graph *gographviz.Escape, instanceName string, awsInstance DBInstance) (error) {
	// Create DB instance node
	var clusterID string
	// if there is no DB Subnet Group, the DB instance is created in the default VPC
	// same if there is no VPC defined in the TF module
	if awsInstance.DBSubnetGroupName == nil || len(a.Vpc) == 0{
		clusterID = "aws_vpc_default"
	} else {
		// TODO: support multiple subnets from the DB Subnet group
		// - how to show a DB in multiple subnets?
		// - can a node be part of 2 subgraph (in graphviz)?
		// For now, only the first one is used
		fmt.Println("*awsInstance.DBSubnetGroupName", *awsInstance.DBSubnetGroupName)
		tmpDBname := strings.Split(*awsInstance.DBSubnetGroupName, ".")[1]
		fmt.Println("a.DBSubnetGroup", a.DBSubnetGroup)
		tmpSubnetName := strings.Split(a.DBSubnetGroup[tmpDBname].SubnetIDs[0], ".")[1]
		fmt.Println("tmpSubnetName", tmpSubnetName)
		fmt.Println("tmpSubnetName VPC", a.Subnet[tmpSubnetName].VpcID)
		clusterID = strings.Replace(a.Subnet[tmpSubnetName].VpcID, ".", "_", -1)
	}
	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: aws_db_instance_%s to cluster_%s // Create DB Instance\n", instanceName, clusterID)
	}

	// Splitting label if more than 8 chars
	labelName := strings.Join(utils.ChunkString(instanceName, 8), "\n")

	fontColor := "black"
	// DB is publicly available, so setting label color as red
	if awsInstance.PubliclyAccessible != nil && *awsInstance.PubliclyAccessible == true {
		fontColor = "red"
	}

	err := graph.AddNode("cluster_"+clusterID, "aws_db_instance_"+instanceName, map[string]string{
		"label": labelName,
		"fontcolor": fontColor,
		"image": "./aws/icons/db.png",
		"width": "1",
		"height": "1",
		"fixedsize": "true",
		"shape": "none",
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Data) createALB(graph *gographviz.Escape, instanceName string, awsLB LB) (error) {
	// Create ALB node
	var clusterID string
	// if there is no Subnet, the ALB is created in the default subnet
	// same if there is no subnet defined in the TF module
	if awsLB.Subnets == nil || len(a.Subnet) == 0{
		clusterID = "aws_subnet_default"
	} else {
		// TODO: support multiple subnets
		// - how to show a LB in multiple subnets?
		// - can a node be part of 2 subgraph (in graphviz)?
		// For now, only the first one is used
		tmpSubnetName := strings.Split((*awsLB.Subnets)[0], ".")[1]
		clusterID = "aws_subnet_" + tmpSubnetName
	}

	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: aws_lb_%s to cluster_%s // Create ALB\n", instanceName, clusterID)
	}

	// Splitting label if more than 8 chars
	labelName := strings.Join(utils.ChunkString(instanceName, 8), "\n")

	err := graph.AddNode("cluster_"+clusterID, "aws_lb_"+instanceName, map[string]string{
		"label": labelName,
		"image": "./aws/icons/alb.png",
		"width": "1",
		"height": "1",
		"fixedsize": "true",
		"shape": "none",
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Data) createASG(graph *gographviz.Escape, instanceName string, awsASG AutoscalingGroup) (error) {
	// Create ASG node
	var clusterID string
	// if there is no Subnet, the ASG is created in the default subnet
	// same if there is no subnet defined in the TF module
	if awsASG.VPCZoneIdentifier == nil || len(a.Subnet) == 0{
		clusterID = "aws_subnet_default"
	} else {
		// TODO: support multiple subnets
		// - how to show a LB in multiple subnets?
		// - can a node be part of 2 subgraph (in graphviz)?
		// For now, only the first one is used
		tmpSubnetName := strings.Split((*awsASG.VPCZoneIdentifier)[0], ".")[1]
		clusterID = "aws_subnet_" + tmpSubnetName
	}

	if Verbose == true {
		fmt.Printf("[VERBOSE] AddNode: aws_autoscaling_group_%s to cluster_%s // Create ASG\n", instanceName, clusterID)
	}

	// Splitting label if more than 8 chars
	fmt.Println("DEBUG instanceName =>", instanceName)
	
	labelName := strings.Join(utils.ChunkString(instanceName, 8), "\n")
	fmt.Println("DEBUG labelName =>", labelName)

	err := graph.AddNode("cluster_"+clusterID, "aws_autoscaling_group_"+instanceName, map[string]string{
		"label": labelName,
		"image": "./aws/icons/asg.png",
		"width": "1",
		"height": "1",
		"fixedsize": "true",
		"shape": "none",
	})
	if err != nil {
		return err
	}
	return nil
}

// InitiateVariablesAndResources parses TF file to create Variables / Obj references for interpolation
func InitiateVariablesAndResources(tfModule *tfconfigs.Module) (*hcl2.EvalContext, error) {
	// Create map for EvalContext to replace variables names by their values inside HCL file using DecodeBody
	ctxVariables := make(map[string]cty.Value)
	ctxVpc := make(map[string]cty.Value)
	ctxAwsSubnet := make(map[string]cty.Value)
	ctxAwsInstance := make(map[string]cty.Value)
	ctxAwsSecurityGroup := make(map[string]cty.Value)
	ctxDBInstance := make(map[string]cty.Value)
	ctxDBSubnetGroup := make(map[string]cty.Value)
	ctxLB := make(map[string]cty.Value)
	ctxASG := make(map[string]cty.Value)

	// Prepare context with TF variables
	for _, v := range tfModule.Variables {
		// Handling the case there is no default value for the variable
		if v.Default.IsNull() {
			ctxVariables[v.Name] = cty.StringVal("var_" + v.Name)
		} else {
			ctxVariables[v.Name] = v.Default
		}
	}

	// Load variables from Variable Definitions (.tfvars) Files
	// Start with terraform.tfvars file:
	inputVariablesFile := path.Join(tfModule.SourceDir, "terraform.tfvars")
	_, err := os.Stat(inputVariablesFile)
	if err == nil {
		vars, diags := tfconfigs.NewParser(nil).LoadValuesFile(inputVariablesFile)
		utils.PrintDiags(diags)
		for varName, varValue := range vars {
			ctxVariables[varName] = varValue
		}
	}
	// Search for .auto.tfvars files
	files, err := ioutil.ReadDir(tfModule.SourceDir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".auto.tfvars") {
			inputVariablesFile := path.Join(tfModule.SourceDir, f.Name())
			vars, diags := tfconfigs.NewParser(nil).LoadValuesFile(inputVariablesFile)
			utils.PrintDiags(diags)
			for varName, varValue := range vars {
				ctxVariables[varName] = varValue
			}
		}
	}

	// Prepare context with named values to resources
	for _, v := range tfModule.ManagedResources {
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
		} else if v.Type == "aws_db_instance" {
			ctxDBInstance[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_db_subnet_group" {
			ctxDBSubnetGroup[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_lb" || v.Type == "aws_alb" {
			ctxLB[v.Name] = cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal(v.Type + "." + v.Name),
				})
		} else if v.Type == "aws_autoscaling_group" {
			ctxASG[v.Name] = cty.ObjectVal(map[string]cty.Value{
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
			"aws_db_instance" : cty.ObjectVal(ctxDBInstance),
			"aws_db_subnet_group" : cty.ObjectVal(ctxDBSubnetGroup),
			"aws_alb" : cty.ObjectVal(ctxLB),	// aws_lb and aws_alb are the same
			"aws_lb" : cty.ObjectVal(ctxLB),
			"aws_autoscaling_group" : cty.ObjectVal(ctxASG),
		},
	}


	fmt.Println("CTX:", ctx)
	return ctx, nil
}

// CreateDefaultNodes creates default VPC/Subnet/Security Groups if they don't exist in the TF module
func (a *Data) CreateDefaultNodes(tfModule *tfconfigs.Module, graph *gographviz.Escape) (error) {
	for _, v := range tfModule.ManagedResources {
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
func (a *Data) ParseTfResources(tfModule *tfconfigs.Module, ctx *hcl2.EvalContext, graph *gographviz.Escape) (error) {
	for _, v := range tfModule.ManagedResources {
		switch v.Type {
		case "aws_vpc":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var Vpc Vpc
			diags := gohcl.DecodeBody(v.Config, ctx, &Vpc)
			utils.PrintDiags(diags)

			// Add Vpc to Data
			a.Vpc[v.Name] = Vpc

		case "aws_subnet":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var awsSubnet Subnet
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSubnet)
			utils.PrintDiags(diags)

			// Add Subnet to Data
			a.Subnet[v.Name] = awsSubnet

		case "aws_instance":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
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
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var awsSecurityGroup SecurityGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsSecurityGroup)
			utils.PrintDiags(diags)

			// Add SecurityGroup to Data
			a.SecurityGroup["aws_security_group."+v.Name] = awsSecurityGroup

		case "aws_db_instance":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var awsDBInstance DBInstance
			diags := gohcl.DecodeBody(v.Config, ctx, &awsDBInstance)
			utils.PrintDiags(diags)
			
			// Add DBInstance to Data
			a.DBInstance[v.Name] = awsDBInstance

			if awsDBInstance.VpcSecurityGroupIDs != nil {
				fmt.Println("DEBUG (awsDBInstance.VpcSecurityGroupIDs):", *awsDBInstance.VpcSecurityGroupIDs)
				for _, sg := range *awsDBInstance.VpcSecurityGroupIDs {
					_, found := a.SecurityGroupNodeLinks[sg]
					if found {
						a.SecurityGroupNodeLinks[sg] = append(a.SecurityGroupNodeLinks[sg], v.Type+"."+v.Name)
					} else {
						a.SecurityGroupNodeLinks[sg] = []string{v.Type+"."+v.Name}
					}
					fmt.Println("DEBUG (a.SecurityGroupNodeLinks[sg]):", a.SecurityGroupNodeLinks[sg])
				}
			}

		case "aws_db_subnet_group":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var awsDBSubnetGroup DBSubnetGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsDBSubnetGroup)
			utils.PrintDiags(diags)
			
			fmt.Println("DEBUG =>", awsDBSubnetGroup)
			// Add DBSubnetGroup to Data
			a.DBSubnetGroup[v.Name] = awsDBSubnetGroup

		case "aws_alb", "aws_lb":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var awsLB LB
			diags := gohcl.DecodeBody(v.Config, ctx, &awsLB)
			utils.PrintDiags(diags)
			
			// Add LB to Data
			if awsLB.LoadBalancerType == nil || *awsLB.LoadBalancerType == "application" {
				a.LB[v.Name] = awsLB
			} else {
				fmt.Println("[VERBOSE] Network Load Balancer is not yet supported")
			}
			

		case "aws_autoscaling_group":
			if Verbose == true {
				fmt.Printf("[VERBOSE] Decoding %s.%s\n", v.Type, v.Name)
			}
			var awsASG AutoscalingGroup
			diags := gohcl.DecodeBody(v.Config, ctx, &awsASG)
			utils.PrintDiags(diags)
			
			// Add ASG to Data
			a.ASG[v.Name] = awsASG

		default:
			if Verbose == true {
				fmt.Printf("[VERBOSE] Can't decode %s.%s (not yet supported)\n", v.Type, v.Name)
			}
			a.unsupportedResources = append(a.unsupportedResources, v.Type + "." + v.Name)
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

	// Add DB Instance nodes to graph
	for instanceName, instanceObj := range a.DBInstance {
		err := a.createDBInstance(graph, instanceName, instanceObj)
		if err != nil {
			return err
		}
	}

	// Add ALB nodes to graph
	for lbName, lbObj := range a.LB {
		err := a.createALB(graph, lbName, lbObj)
		if err != nil {
			return err
		}
	}

	// Add ASG nodes to graph
	for asgName, asgObj := range a.ASG {
		err := a.createASG(graph, asgName, asgObj)
		if err != nil {
			return err
		}
	}

	return nil
}

func createInternetSGRuleEdge(ruleType int, nodeName string, graph *gographviz.Escape) (error) {
	// Highlight Ingress from 0.0.0.0/0 and Egress to 0.0.0.0/0 in red

	// Based on the rule type Ingress or Egress define the source and destination items
	var src, dst string
	if ruleType == ingressRule {
		src, dst = "Internet", nodeName
	} else {
		src, dst = nodeName, "Internet"
	}

	if Verbose == true {
		fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
	}
	err := graph.AddEdge(src, dst, true, map[string]string{
		"color": "red",
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Data) parseSGRule(ruleType int, nodeName string, sgName string, graph *gographviz.Escape) (error) {
	// Based on the rule type Ingress or Egress define the source and destination items
	var src, dst string
	var sgRule []SGRule
	if ruleType == ingressRule {
		src, dst = sgName, nodeName
		sgRule = a.SecurityGroup[sgName].Ingress
	} else {
		src, dst = nodeName, sgName
		sgRule = a.SecurityGroup[sgName].Egress
	}

	if _, found1 := a.SecurityGroup[sgName]; !found1 {
		_, found2 := utils.Find(a.undefinedSecurityGroups, sgName)
		if !found2 {
			// If the SG is not defined in TF, we need to create the Node before the Edges
			if Verbose == true {
				fmt.Printf("[VERBOSE] AddNode: %s to G\n", sgName)
			}
			err := graph.AddNode("G", sgName, map[string]string{
				"style": "dotted",
				"label": sgName,
			})
			if err != nil {
				return err
			}
			a.undefinedSecurityGroups = append(a.undefinedSecurityGroups, sgName)
		}

		// The SG exists, we just need to link it with the appropriate nodes
		if Verbose == true {
			fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
		}
		err := graph.AddEdge(src, dst, true, nil)
		if err != nil {
			return err
		}
	}
	for _, rule := range sgRule {
		fmt.Println("DEBUG RULE =>", rule)
		fmt.Println("DEBUG RULE =>", rule.CidrBlocks)
		if rule.CidrBlocks != nil {
			for _, cidr := range *rule.CidrBlocks {
				// Special ingress/egress rule for 0.0.0.0/0
				if cidr == "0.0.0.0/0" {
					err := createInternetSGRuleEdge(ruleType, nodeName, graph)
					if err != nil {
						return err
					}
				} else {
					ipAddrSG, _, err := net.ParseCIDR(cidr)
					if err != nil {
						// Unrecognized SG name
						utils.PrintError(err)
					} else {
						// The source/destination is a valid CIDR
						edgeCreated := false
						for k, v := range(a.Subnet) {
							// Checking for Security Group source/destination IP / Subnet matching
							_, ipNetSubnet, err := net.ParseCIDR(v.CidrBlock)
							if err != nil {
								return err
							}
							if ipNetSubnet.Contains(ipAddrSG) {
								// the source/destination IP is part of this subnet CIDR
								if ruleType == ingressRule {
									src, dst = "aws_subnet_"+k, nodeName
								} else {
									src, dst = nodeName, "aws_subnet_"+k
								}
								if Verbose == true {
									fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
								}
								err = graph.AddEdge(src, dst, true, nil)
								if err != nil {
									return err
								}
								edgeCreated = true
							}
						}

						if !edgeCreated {
							// Security Group source/destination IP did not matched with Subnet CIDRs
							// Now checking with VPC CIDRs
							for k, v := range a.Vpc {
								_, ipNetVpc, err := net.ParseCIDR(v.CidrBlock)
								if err != nil {
									return err
								}
								if ipNetVpc.Contains(ipAddrSG) {
									// the source/destination IP is part of this VPC CIDR
									if ruleType == ingressRule {
										src, dst = "aws_vpc_"+k, nodeName
									} else {
										src, dst = nodeName, "aws_vpc_"+k
									}
									if Verbose == true {
										fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
									}
									err = graph.AddEdge(src, dst, true, nil)
									if err != nil {
										return err
									}
									edgeCreated = true
								}
							}
						}

						if !edgeCreated {
							// Security Group source/destination IP did not matched with Subnet and VPC CIDRs
							// Creating a node for the source/destination as it is likely to be an undefined IP/CIDR
							if Verbose == true {
								fmt.Printf("[VERBOSE] AddNode: %s to G\n", cidr)
							}
							err := graph.AddNode("G", cidr, nil)
							if err != nil {
								return err
							}
							if ruleType == ingressRule {
								src, dst = cidr, nodeName
							} else {
								src, dst = nodeName, cidr
							}
							if Verbose == true {
								fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
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

		// Create edges for all instances linked to SGRule.Self
		if rule.Self != nil && *rule.Self != false {
			for _, v1 := range a.SecurityGroupNodeLinks[sgName] {
				v2 := strings.Replace(v1, ".", "_", -1)
				if v2 != nodeName {
					if ruleType == ingressRule {
						src, dst = v2, nodeName
					} else {
						src, dst = nodeName, v2
					}
					if Verbose == true {
						fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
					}
					err := graph.AddEdge(src, dst, true, nil)
					if err != nil {
						return err
					}
				}
			}
		}

		// Create edges for all instances linked to SGRule.SecurityGroups
		if rule.SecurityGroups != nil {
			for _, v1 := range *rule.SecurityGroups {
				for _, v2 := range a.SecurityGroupNodeLinks[v1] {
					v3 := strings.Replace(v2, ".", "_", -1)
					if v3 != nodeName {
						if ruleType == ingressRule {
							src, dst = v3, nodeName
						} else {
							src, dst = nodeName, v3
						}
						if Verbose == true {
							fmt.Printf("[VERBOSE] AddEdge: %s -> %s\n", src, dst)
						}
						err := graph.AddEdge(src, dst, true, nil)
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
			if Verbose == true {
				fmt.Printf("[VERBOSE] AddEdge: sg-default -> aws_instance_%s\n", instanceName)
			}
			err := graph.AddEdge("sg-default", "aws_instance_"+instanceName, true, nil)
			if err != nil {
				return err
			}
		}
		// The instance has at least one SG attached to it
		for _, sg := range SGs {
			// Parse Ingress SG rules
			if !IgnoreIngress {
				a.parseSGRule(ingressRule, "aws_instance_"+instanceName, sg, graph)
			}

			// Parse Egress SG rules
			if !IgnoreEgress {
				a.parseSGRule(egressRule, "aws_instance_"+instanceName, sg, graph)
			}
		}
	}

	// Link DB Instances with their Security Groups
	for instanceName, instanceObj := range a.DBInstance {

		// Get the Security Groups of the DB instance
		var SGs []string
		if instanceObj.VpcSecurityGroupIDs != nil {
			SGs = append(SGs, *instanceObj.VpcSecurityGroupIDs...)
		}

		// The instance has at least one SG attached to it
		for _, sg := range SGs {
			// Parse Ingress SG rules
			if !IgnoreIngress {
				a.parseSGRule(ingressRule, "aws_db_instance_"+instanceName, sg, graph)
			}

			// Parse Egress SG rules
			if !IgnoreEgress {
				a.parseSGRule(egressRule, "aws_db_instance_"+instanceName, sg, graph)
			}
		}
	}

	return nil
}

// PrintUnsupportedResources displays all resources currently unsupported by tfviz
func (a *Data) PrintUnsupportedResources() {
	if len(a.unsupportedResources) > 0 {
		fmt.Println("[WARNING] Unsupported resources:")
		for _, r := range a.unsupportedResources {
			fmt.Println(" -", r)
		}
	}
}