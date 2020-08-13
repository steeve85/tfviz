package utils

import (
	"fmt"
	"math/rand"
	"testing"

	hcl2 "github.com/hashicorp/hcl/v2"
)

func TestPrintError(t *testing.T) {
	err := fmt.Errorf("test error")
	PrintError(err)
}

func TestPrintDiags(t *testing.T) {
	// Testing with a single Diagnostic
	oneDiag := hcl2.Diagnostics {
		&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "A single diagnostic",
			Detail:   "A single diagnostic - details",
			Subject: &hcl2.Range{
				Start: hcl2.Pos{
					Byte:   0,
					Column: 1,
					Line:   1,
				},
				End: hcl2.Pos{
					Byte:   3,
					Column: 4,
					Line:   1,
				},
			},
		},
	}
	PrintDiags(oneDiag)

	// Testing with multiple Diagnostics
	twoDiags := hcl2.Diagnostics {
		&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "1st diag",
			Detail:   "1st diag - details",
			Subject: &hcl2.Range{
				Start: hcl2.Pos{
					Byte:   0,
					Column: 1,
					Line:   1,
				},
				End: hcl2.Pos{
					Byte:   3,
					Column: 4,
					Line:   1,
				},
			},
		},
		&hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "2nd diag",
			Detail:   "2nd diag - details",
			Subject: &hcl2.Range{
				Start: hcl2.Pos{
					Byte:   0,
					Column: 1,
					Line:   1,
				},
				End: hcl2.Pos{
					Byte:   3,
					Column: 4,
					Line:   1,
				},
			},
		},
	}
	PrintDiags(twoDiags)
}

func TestInitiateGraph(t *testing.T) {
	graph, err := InitiateGraph()
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(graph.Nodes.Nodes) != 1 {
		t.Errorf("Incorrect number of nodes")
	}
}

func TestParseTFfile(t *testing.T) {
	// Testing TF file
	tfModule, err := ParseTFfile("testdata/main.tf")
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(tfModule.Variables) != 1 {
		t.Errorf("Incorrect number of Variables")
	}
	if len(tfModule.ManagedResources) != 9 {
		t.Errorf("Incorrect number of ManagedResources")
	}

	// Testing TF module
	tfModule, err = ParseTFfile("testdata/")
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(tfModule.Variables) != 1 {
		t.Errorf("Incorrect number of Variables")
	}
	if len(tfModule.ManagedResources) != 9 {
		t.Errorf("Incorrect number of ManagedResources")
	}
}


func TestExportGraphToFile(t *testing.T) {
	graph, err := InitiateGraph()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Exporting in DOT format
	outputPath := fmt.Sprintf("/tmp/tfviz_TestExportGraphToFile_%d.dot", rand.Intn(100000))
	err = ExportGraphToFile(outputPath, "dot", graph)
	if err != nil {
		t.Errorf(err.Error())
	}

	outputPath = fmt.Sprintf("/tmp/tfviz_TestExportGraphToFile_%d.png", rand.Intn(100000))
	err = ExportGraphToFile(outputPath, "png", graph)
	if err != nil {
		t.Errorf(err.Error())
	}
}


