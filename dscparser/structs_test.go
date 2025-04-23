package dscparser

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDSCDataEquals(t *testing.T) {
	dscData := ReadFile("./testdata/structs/dscdata/unsorted_dsc_file.xml", "loc", "ns")
	expectedDscData := ReadFile("./testdata/structs/dscdata/sorted_dsc_file.xml", "loc", "ns")
	assert.True(t, dscData.Equals(*expectedDscData))
}

func TestDSCDataUnequals(t *testing.T) {
	dscData1 := ReadFile("./testdata/structs/dscdata/unequal_dsc_file1.xml", "loc", "ns")
	dscData2 := ReadFile("./testdata/structs/dscdata/unequal_dsc_file2.xml", "loc", "ns")
	dscData3 := ReadFile("./testdata/structs/dscdata/unequal_dsc_file3.xml", "loc", "ns")
	expectedDscData := ReadFile("./testdata/structs/dscdata/sorted_dsc_file.xml", "loc", "ns")
	assert.False(t, dscData1.Equals(*expectedDscData))
	assert.False(t, dscData2.Equals(*expectedDscData))
	assert.False(t, dscData3.Equals(*expectedDscData))
}

func TestSortDSCData(t *testing.T) {

	dscData := ReadFile("./testdata/structs/dscdata/unsorted_dsc_file.xml", "loc", "ns")
	expectedDscData := ReadFile("./testdata/structs/dscdata/sorted_dsc_file.xml", "loc", "ns")
	dscData.Sort()

	if !reflect.DeepEqual(dscData, expectedDscData) {
		t.Logf("Actual sorted DSCData \n%+v\n\n", dscData)
		t.Logf("Expected sorted DSCData: \n%+v\n\n", expectedDscData)
		t.Errorf("Sort(dscData) doesnt deeply match sorted DSCData")
	}
}

func TestDatasetEquals(t *testing.T) {
	dscData := ParseDataset("./testdata/structs/dataset/unsorted_dataset.xml")
	expectedDscData := ParseDataset("./testdata/structs/dataset/expected_dataset.xml")
	assert.True(t, dscData.Equals(*expectedDscData))
}

func TestDatasetUnequals(t *testing.T) {
	dscData := ParseDataset("./testdata/structs/dataset/unequal_dataset1.xml")
	expectedDscData := ParseDataset("./testdata/structs/dataset/expected_dataset.xml")
	assert.False(t, dscData.Equals(*expectedDscData))
}
