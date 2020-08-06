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

var Ignorewarnings bool

func PrintError(err error) {
	e := fmt.Errorf("[ERROR] %s", err)
	fmt.Println(e.Error())
}

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


func ExportGraphToFile(outputPath string, outputFormat string, graph *gographviz.Escape) error {
	fmt.Println("Exporting Graph to", outputPath)
	if outputFormat == "dot" {
		err := ioutil.WriteFile(outputPath, []byte(graph.String()), 0644)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
	} else {
		tFlag := fmt.Sprintf("-T%s", outputFormat)
		cmd := exec.Command("dot", tFlag, "-o", outputPath)
		cmd.Stdin = strings.NewReader(graph.String())
		err := cmd.Run()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
	}
	return nil
}

func ParseTFfile(configpath string) (*tfconfigs.Module, error) {
	f, err := os.Stat(configpath);
	if err != nil {
		PrintError(err)
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
		PrintDiags(diags)
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
		PrintDiags(diags)
		return module, nil
	}
}

func InitiateGraph() (*gographviz.Escape, error) {
	// Graph initialization
	g := gographviz.NewEscape()
	g.SetName("G")
	g.SetDir(false)

	// Adding node for Internet representation
	err := g.AddNode("G", "Internet", map[string]string{
		"shape": "octagon",
	})
	if err != nil {
		PrintError(err)
		return nil, err
	}
	return g, nil
}