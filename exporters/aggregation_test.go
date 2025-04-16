package exporters

import (
	"cmp"
	"reflect"
	"slices"
	"testing"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"
)

func TestMaxCells(t *testing.T) {
	x := 5

	testDataset := dscparser.ParseDataset("./testdata/aggregation/MaxCells/test_dataset.xml")
	MaxCells(testDataset, x)

	expectedDataset := dscparser.ParseDataset("./testdata/aggregation/MaxCells/expected_dataset.xml")

	if !reflect.DeepEqual(testDataset, expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("MaxCells(test_dataset) doesnt deeply match expected_dataset")
	}
}

func DSCDataEquals(expected *dscparser.DSCData, actual *dscparser.DSCData) bool {
	// Sort cells after label
	cmpCell := func(a, b dscparser.Cell) int {
		return cmp.Compare(a.Value, b.Value)
	}

	dscDatas := []*dscparser.DSCData{expected, actual}
	for _, dscData := range dscDatas {
		for j := range dscData.Datasets {
			dataset := &dscData.Datasets[j]
			for k := range dataset.Data.Rows {
				row := &dataset.Data.Rows[k]
				slices.SortFunc(row.Cells, cmpCell)
			}
		}
	}

	return reflect.DeepEqual(expected, actual)
}

func TestEliminateDimensionOne(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/aggregation/EliminateDimension/Dimension1/test_dataset.xml")

	EliminateDimensionOne(testDataset)
	//Sort cells after label
	cmpCell := func(a, b dscparser.Cell) int {
		return cmp.Compare(a.Value, b.Value)
	}
	slices.SortFunc(testDataset.Data.Rows[0].Cells, cmpCell)

	expectedDataset := dscparser.ParseDataset("./testdata/aggregation/EliminateDimension/Dimension1/expected_dataset.xml")

	if !reflect.DeepEqual(testDataset, expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("EliminateDimension(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestEliminateDimensionTwo(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/aggregation/EliminateDimension/Dimension2/test_dataset.xml")
	EliminateDimensionTwo(testDataset)

	expectedDataset := dscparser.ParseDataset("./testdata/aggregation/EliminateDimension/Dimension2/expected_dataset.xml")

	if !reflect.DeepEqual(testDataset, expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("EliminateDimension(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestFilterForPrometheus(t *testing.T) {

	config := config.ParseConfig("./testdata/aggregation/FilterForPrometheus/config.yaml")

	testDSCData := dscparser.ReadFile("./testdata/aggregation/FilterForPrometheus/test_dsc_file.xml", "loc", "ns")
	FilterForPrometheus(testDSCData, config)

	expectedDSCData := dscparser.ReadFile("./testdata/aggregation/FilterForPrometheus/expected_dsc_file.xml", "loc", "ns")

	if !DSCDataEquals(testDSCData, expectedDSCData) {
		t.Logf("Test Data: \n%+v\n\n", testDSCData)
		t.Logf("Expected Output: \n%+v\n\n", expectedDSCData)
		t.Errorf("FilterForPrometheus(testDSCData) doesnt deeply match expectedDSCData")
	}
}
