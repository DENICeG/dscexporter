package aggregation

import (
	"cmp"
	"slices"
	"testing"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"
)

func TestMaxCells(t *testing.T) {
	x := 5

	testDataset := dscparser.ParseDataset("./testdata/MaxCells/test_dataset.xml")
	MaxCells(testDataset, x)

	expectedDataset := dscparser.ParseDataset("./testdata/MaxCells/expected_dataset.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("MaxCells(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestEliminateDimensionOne(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/EliminateDimension/Dimension1/test_dataset.xml")

	EliminateDimensionOne(testDataset)
	//Sort cells after label
	cmpCell := func(a, b dscparser.Cell) int {
		return cmp.Compare(a.Value, b.Value)
	}
	slices.SortFunc(testDataset.Data.Rows[0].Cells, cmpCell)

	expectedDataset := dscparser.ParseDataset("./testdata/EliminateDimension/Dimension1/expected_dataset.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("EliminateDimension(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestEliminateDimensionTwo(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/EliminateDimension/Dimension2/test_dataset.xml")
	EliminateDimensionTwo(testDataset)

	expectedDataset := dscparser.ParseDataset("./testdata/EliminateDimension/Dimension2/expected_dataset.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("EliminateDimension(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestFilterDimensionOne(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/Filter/test_dataset.xml")

	allowedQtypes := []string{
		"1",  // A
		"12", // PTR
	}
	FilterDimensionOne(testDataset, allowedQtypes)

	expectedDataset := dscparser.ParseDataset("./testdata/Filter/expected_dataset_dim1.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("FilterDimensionOne(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestFilterDimensionTwo(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/Filter/test_dataset.xml")

	allowedTlds := []string{"de", "localhost"}
	FilterDimensionTwo(testDataset, allowedTlds)

	expectedDataset := dscparser.ParseDataset("./testdata/Filter/expected_dataset_dim2.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("FilterDimensionTwo(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestReplaceLabels(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/ReplaceLabels/test_dataset.xml")
	ReplaceLabels(testDataset)

	expectedDataset := dscparser.ParseDataset("./testdata/ReplaceLabels/expected_dataset.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("ReplaceLabels(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestAggregateForPrometheus(t *testing.T) {

	config := config.ParseConfig("./testdata/AggregateForPrometheus/config.yaml")

	testDSCData := dscparser.ReadFile("./testdata/AggregateForPrometheus/test_dsc_file.xml", "loc", "ns")
	AggregateForPrometheus(testDSCData, config)

	expectedDSCData := dscparser.ReadFile("./testdata/AggregateForPrometheus/expected_dsc_file.xml", "loc", "ns")

	if !testDSCData.Equals(*expectedDSCData) {
		t.Logf("Test Data: \n%+v\n\n", testDSCData)
		t.Logf("Expected Output: \n%+v\n\n", expectedDSCData)
		t.Errorf("AggregateForPrometheus(testDSCData) doesnt deeply match expectedDSCData")
	}
}
