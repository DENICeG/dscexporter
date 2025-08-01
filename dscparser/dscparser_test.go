package dscparser

import (
	"encoding/xml"
	"reflect"
	"testing"
)

func TestReadFile(t *testing.T) {
	dscData := ReadFile("./testdata/test_dsc_file.xml", "loc", "ns")

	expectedDscData := DSCData{
		XMLName:    xml.Name{Local: "dscdata"},
		NameServer: "ns",
		Location:   "loc",
		Datasets: []Dataset{
			Dataset{
				XMLName:    xml.Name{Local: "array"},
				Name:       "pcap_stats",
				StartTime:  int64(1741170540),
				StopTime:   int64(1741170600),
				Dimensions: 2,
				DimensionInfo: []DimensionInfo{
					DimensionInfo{Number: 1, Type: "ifname"},
					DimensionInfo{Number: 2, Type: "pcap_stat"},
				},
				Data: Data{
					XMLName: xml.Name{Local: "data"},
					Rows: []Row{
						Row{
							XMLName: xml.Name{Local: "ifname"},
							Value:   "ens2f0",
							Cells: []Cell{
								Cell{
									XMLName: xml.Name{Local: "pcap_stat"},
									Value:   "filter_received",
									Count:   8,
								},
								Cell{
									XMLName: xml.Name{Local: "pcap_stat"},
									Value:   "pkts_captured",
									Count:   8,
								},
							},
						},
					},
				},
			},
			Dataset{
				XMLName:    xml.Name{Local: "array"},
				Name:       "servfail_qname",
				StartTime:  int64(1741170540),
				StopTime:   int64(1741170600),
				Dimensions: 2,
				DimensionInfo: []DimensionInfo{
					DimensionInfo{Number: 1, Type: "ALL"},
					DimensionInfo{Number: 2, Type: "Qname"},
				},
				Data: Data{
					XMLName: xml.Name{Local: "data"},
					Rows:    nil,
				},
			},
		},
	}

	if !reflect.DeepEqual(*dscData, expectedDscData) {
		t.Logf("Parsed DscData: \n%+v\n\n", dscData)
		t.Logf("Expected DscData: \n%+v\n\n", expectedDscData)
		t.Errorf("Parsed DscData doesnt deeply match expected DscData")
	}
}
