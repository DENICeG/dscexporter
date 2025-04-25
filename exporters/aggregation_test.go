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

func TestFilterDimensionOne(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/aggregation/Filter/test_dataset.xml")

	allowedQtypes := []string{
		"1",   // A
		"2",   // NS
		"5",   // CNAME
		"6",   // SOA
		"12",  // PTR
		"15",  // MX
		"16",  // TXT
		"28",  // AAAA
		"33",  // SRV
		"38",  // A6
		"43",  // DS
		"48",  //DNSKEY
		"65",  // HTTPS
		"255", // ANY
		"257", // CAA
	}
	FilterDimensionOne(testDataset, allowedQtypes)

	expectedDataset := dscparser.ParseDataset("./testdata/aggregation/Filter/expected_dataset_dim1.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("FilterDimensionOne(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestFilterDimensionTwo(t *testing.T) {
	testDataset := dscparser.ParseDataset("./testdata/aggregation/Filter/test_dataset.xml")

	allowedTlds := []string{"de", "localhost"}
	FilterDimensionTwo(testDataset, allowedTlds)

	expectedDataset := dscparser.ParseDataset("./testdata/aggregation/Filter/expected_dataset_dim2.xml")

	if !testDataset.Equals(*expectedDataset) {
		t.Logf("Test Dataset: \n%+v\n\n", testDataset)
		t.Logf("Expected Output: \n%+v\n\n", expectedDataset)
		t.Errorf("FilterDimensionTwo(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestAggregateForPrometheus(t *testing.T) {

	config := config.ParseConfig("./testdata/aggregation/AggregateForPrometheus/config.yaml")

	testDSCData := dscparser.ReadFile("./testdata/aggregation/AggregateForPrometheus/test_dsc_file.xml", "loc", "ns")
	AggregateForPrometheus(testDSCData, config)

	expectedDSCData := dscparser.ReadFile("./testdata/aggregation/AggregateForPrometheus/expected_dsc_file.xml", "loc", "ns")

	if !DSCDataEquals(testDSCData, expectedDSCData) {
		t.Logf("Test Data: \n%+v\n\n", testDSCData)
		t.Logf("Expected Output: \n%+v\n\n", expectedDSCData)
		t.Errorf("AggregateForPrometheus(testDSCData) doesnt deeply match expectedDSCData")
	}
}
