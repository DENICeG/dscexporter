package exporters

import (
	"dscexporter/config"
	"dscexporter/dscparser"
	"reflect"
	"testing"
)

func TestMaxCells(t *testing.T) {
	x := 5

	testDataset := dscparser.ParseDataset("./testdata/test_dataset.xml")
	MaxCells(testDataset, x)

	expectedDataset := dscparser.ParseDataset("./testdata/expected_dataset.xml")

	if !reflect.DeepEqual(testDataset, expectedDataset) {
		t.Logf("TestDataset: \n%+v\n\n", testDataset)
		t.Logf("Expected TestDataset: \n%+v\n\n", expectedDataset)
		t.Errorf("MaxCells(test_dataset) doesnt deeply match expected_dataset")
	}
}

func TestFilterForPrometheus(t *testing.T) {

	config := config.ParseConfig("./testdata/config.yaml")

	testDSCData := dscparser.ReadFile("./testdata/test_dsc_file.xml", "loc", "ns")
	FilterForPrometheus(testDSCData, config)

	expectedDSCData := dscparser.ReadFile("./testdata/expected_dsc_file.xml", "loc", "ns")

	if !reflect.DeepEqual(testDSCData, expectedDSCData) {
		t.Logf("TestData: \n%+v\n\n", testDSCData)
		t.Logf("Expected TestData: \n%+v\n\n", expectedDSCData)
		t.Errorf("FilterForPrometheus(testDSCData) doesnt deeply match expectedDSCData")
	}
}
