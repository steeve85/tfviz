package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	tfconfigs "github.com/hashicorp/terraform/configs"
	hcl2 "github.com/hashicorp/hcl/v2"
	"github.com/awalterschulze/gographviz"
)

// Ignorewarnings is used to ignore warnings if set to true. Default is false (warnings will be displayed)
var Ignorewarnings bool

// Verbose enables verbose mode if set to true
var Verbose bool

// PrintError displays errors
func PrintError(err error) {
	e := fmt.Errorf("[ERROR] %s", err)
	fmt.Println(e)
}

// PrintDiags diaplays warning messages
func PrintDiags(diags hcl2.Diagnostics) {
	if !Ignorewarnings {
		if len(diags) == 1 {
			fmt.Println("[WARNING] Diagnostics:", diags[0].Error())
		} else if len(diags) > 1 {
			fmt.Println("[WARNING] Diagnostics:")
			for _, d := range diags {
				fmt.Println("\t", d.Error())
			}
		}
	}
}

// ExportGraphToFile exports Graph to file
func ExportGraphToFile(outputPath string, outputFormat string, graph *gographviz.Escape) error {
	fmt.Println("Exporting Graph to", outputPath)
	if outputFormat == "dot" {
		err := ioutil.WriteFile(outputPath, []byte(graph.String()), 0644)
		if err != nil {
			return err
		}
	} else {
		tFlag := fmt.Sprintf("-T%s", outputFormat)
		cmd := exec.Command("dot", tFlag, "-o", outputPath)
		cmd.Stdin = strings.NewReader(graph.String())
		if Verbose == true {
			fmt.Printf("[VERBOSE] Running command: %s\n", cmd.String())
		}
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseTFfile loads a file path and returns a TF module
func ParseTFfile(configpath string) (*tfconfigs.Module, error) {
	f, err := os.Stat(configpath);
	if err != nil {
		return nil, err
	}

	tfparser := tfconfigs.NewParser(nil)

	switch {
	  case f.IsDir():
		fmt.Println("Parsing", configpath, "Terraform module...")
		if tfparser.IsConfigDir(configpath) == false {
			err := fmt.Errorf("[ERROR] Directory %s does not contain valid Terraform configuration files", configpath)
			return nil, err
		}
		module, diags := tfparser.LoadConfigDir(configpath)
		PrintDiags(diags)
		return module, nil
	  default:
		fmt.Println("Parsing", configpath, "Terraform file...")
		file, diags := tfparser.LoadConfigFile(configpath)
		// Return error if the TF file doesn't contain resources
		if len(file.ManagedResources) == 0 {
			err := fmt.Errorf("[ERROR] File %s does not contain valid Terraform configuration", configpath)
			return nil, err
		}
		module, moreDiags := tfconfigs.NewModule([]*tfconfigs.File{file}, nil)
		diags = append(diags, moreDiags...)
		PrintDiags(diags)
		return module, nil
	}
}

// InitiateGraph initializes the graph
func InitiateGraph() (*gographviz.Escape, error) {
	// Graph initialization
	g := gographviz.NewEscape()
	g.SetName("G")
	g.SetDir(true)
	//g.AddAttr("G", "fontsize", "10")

	if Verbose == true {
		fmt.Println("[VERBOSE] AddNode: Internet to G")
	}
	// Adding node for Internet representation
	err := g.AddNode("G", "Internet", map[string]string{
		"shape": "none",
		"label": "Internet",
		"labelloc": "b",
		"image": "./aws/icons/internet.png",
	})
	return g, err
}

// RemoveDuplicateValues removes duplicate strings in []string slices
// Ref: https://www.geeksforgeeks.org/how-to-remove-duplicate-values-from-slice-in-golang/
func RemoveDuplicateValues(strSlice []string) []string { 
    keys := make(map[string]bool) 
    list := []string{} 
  
    // If the key(values of the slice) is not equal 
    // to the already present value in new slice (list) 
    // then we append it. else we jump on another element. 
    for _, entry := range strSlice { 
        if _, value := keys[entry]; !value { 
            keys[entry] = true
            list = append(list, entry) 
        } 
    } 
    return list 
}

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
