package aggregation

import (
	"cmp"
	"encoding/xml"
	"slices"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"
)

var REPLACEMENTS = map[string]map[string]string{
	"Qtype": {
		"1":   "A",
		"2":   "NS",
		"5":   "CNAME",
		"6":   "SOA",
		"12":  "PTR",
		"15":  "MX",
		"16":  "TXT",
		"24":  "SIG",
		"25":  "KEY",
		"28":  "AAAA",
		"30":  "NXT",
		"33":  "SRV",
		"38":  "A6",
		"43":  "DS",
		"46":  "RRSIG",
		"47":  "NSEC",
		"48":  "DNSKEY",
		"50":  "NSEC3",
		"65":  "HTTPS",
		"255": "ANY",
		"257": "CAA",
	},
	"Rcode": {
		"0": "NOERROR",
		"1": "FORMERR",
		"2": "SERVFAIL",
		"3": "NXDOMAIN",
		"4": "NOTIMP",
		"5": "REFUSED",
		"6": "YXDOMAIN",
		"7": "XRRSET",
		"8": "NOTAUTH",
		"9": "NOTZONE",
	},
	"Opcode": {
		"0": "Query",
		"1": "IQuery",
		"2": "Status",
		"3": "Unassigned",
		"4": "Notify",
		"5": "Update",
		"6": "DSO",
	},
	"QRAABits": { // Yes, you could just base64 decode, but this works too
		"cXI9MCxhYT0w": "qr=0,aa=0",
		"cXI9MSxhYT0w": "qr=1,aa=0",
		"cXI9MSxhYT0x": "qr=1,aa=1",
		"cXI9MCxhYT0x": "qr=0,aa=1",
	},
}

func MaxCells(dataset *dscparser.Dataset, x int) {
	// Sort Cells
	cmpCell := func(a, b dscparser.Cell) int {

		if a.Value == "-:SKIPPED:-" {
			return 1 // a is greater
		}
		if b.Value == "-:SKIPPED:-" {
			return -1 // b is greater
		}
		if a.Value == "-:SKIPPED_SUM:-" {
			return 1 // a is greater
		}
		if b.Value == "-:SKIPPED_SUM:-" {
			return -1 // b is greater
		}

		return cmp.Compare(a.Count, b.Count)
	}

	label2 := dataset.DimensionInfo[1].Type

	for i := range dataset.Data.Rows {
		row := &dataset.Data.Rows[i]

		//Sort cells in descending order
		slices.SortFunc(row.Cells, cmpCell)
		slices.Reverse(row.Cells)

		if len(row.Cells) <= x {
			continue
		}

		if len(row.Cells) < 2 || row.Cells[0].Value != "-:SKIPPED:-" {
			skippedEntries := []dscparser.Cell{
				dscparser.Cell{XMLName: xml.Name{Local: label2}, Value: "-:SKIPPED:-", Count: 0},
				dscparser.Cell{XMLName: xml.Name{Local: label2}, Value: "-:SKIPPED_SUM:-", Count: 0},
			}
			row.Cells = append(skippedEntries, row.Cells...)
		}

		for i := len(row.Cells) - 1; i > (x+2)-1; i-- {
			row.Cells[0].Count += 1                  //-:SKIPPED:-
			row.Cells[1].Count += row.Cells[i].Count // -:SKIPPED_SUM:-
		}
		row.Cells = row.Cells[:min(len(row.Cells), x+2)]
	}

}

func EliminateDimensionOne(dataset *dscparser.Dataset) {
	dataset.DimensionInfo[0].Type = "All"
	cells := make(map[string]int)

	for _, row := range dataset.Data.Rows {
		for _, cell := range row.Cells {
			cells[cell.Value] = cells[cell.Value] + cell.Count
		}
	}
	cellObjects := make([]dscparser.Cell, 0, len(cells))
	for value, count := range cells {
		cellObjects = append(cellObjects, dscparser.Cell{XMLName: xml.Name{Local: dataset.DimensionInfo[1].Type}, Value: value, Count: count})
	}
	row := dscparser.Row{XMLName: xml.Name{Local: "All"}, Value: "All", Cells: cellObjects}
	dataset.Data.Rows = []dscparser.Row{row}
}

func EliminateDimensionTwo(dataset *dscparser.Dataset) {
	dataset.DimensionInfo[1].Type = "All"
	for i := range dataset.Data.Rows {
		row := &dataset.Data.Rows[i]
		sum := 0
		for _, cell := range row.Cells {
			sum += cell.Count
		}
		row.Cells = []dscparser.Cell{dscparser.Cell{XMLName: xml.Name{Local: "All"}, Value: "All", Count: sum}}
	}
}

func createAllowedValuesSet(allowedValues []string) map[string]bool {
	set := map[string]bool{}
	for _, allowedValue := range allowedValues {
		set[allowedValue] = true
	}
	return set
}

func FilterDimensionOne(dataset *dscparser.Dataset, allowedValues []string) {
	allowedValuesSet := createAllowedValuesSet(allowedValues)
	delete(allowedValuesSet, "other") // Other rows are created by this method, so they cant exist before
	rows := []dscparser.Row{}
	other := map[string]int{}
	for _, row := range dataset.Data.Rows {
		if allowedValuesSet[row.Value] {
			rows = append(rows, row)
		} else {
			for _, cell := range row.Cells {
				other[cell.Value] += cell.Count
			}
		}
	}

	otherCells := []dscparser.Cell{}
	for value, count := range other {
		otherCells = append(otherCells, dscparser.Cell{
			XMLName: xml.Name{Local: dataset.DimensionInfo[1].Type},
			Value:   value,
			Count:   count,
		})
	}
	if len(otherCells) > 0 {
		rows = append(rows,
			dscparser.Row{
				XMLName: xml.Name{Local: dataset.DimensionInfo[0].Type},
				Value:   "other",
				Cells:   otherCells,
			})
	}
	dataset.Data.Rows = rows
}

func FilterDimensionTwo(dataset *dscparser.Dataset, allowedValues []string) {
	allowedValuesSet := createAllowedValuesSet(allowedValues)
	delete(allowedValuesSet, "other") // Other cells are created by this method, so they cant exist before
	for i := range dataset.Data.Rows {
		row := &dataset.Data.Rows[i]
		cells := []dscparser.Cell{}
		other := 0
		for _, cell := range row.Cells {
			if allowedValuesSet[cell.Value] {
				cells = append(cells, cell)
			} else {
				other += cell.Count
			}
		}
		if other > 0 {
			cells = append(cells, dscparser.Cell{
				XMLName: xml.Name{Local: dataset.DimensionInfo[1].Type},
				Value:   "other",
				Count:   other,
			})
		}
		row.Cells = cells
	}
}

func getAllowedValuesForLabel(label string) []string {
	allowedValues := []string{}
	if replacementsForLabel, ok := REPLACEMENTS[label]; ok {
		for key, _ := range replacementsForLabel {
			allowedValues = append(allowedValues, key)
		}
	}
	return allowedValues
}

func ReplaceLabels(dataset *dscparser.Dataset) {

	label1 := dataset.DimensionInfo[0].Type
	if _, ok := REPLACEMENTS[label1]; ok {
		allowedValues := getAllowedValuesForLabel(label1)
		FilterDimensionOne(dataset, allowedValues)
		for i := range dataset.Data.Rows {
			row := &dataset.Data.Rows[i]
			if newValue, ok := REPLACEMENTS[label1][row.Value]; ok {
				row.Value = newValue
			}
		}
	}

	label2 := dataset.DimensionInfo[1].Type
	if _, ok := REPLACEMENTS[label2]; ok {
		allowedValues := getAllowedValuesForLabel(label2)
		FilterDimensionTwo(dataset, allowedValues)
		for i := range dataset.Data.Rows {
			row := &dataset.Data.Rows[i]
			for j := range row.Cells {
				cell := &row.Cells[j]
				if newValue, ok := REPLACEMENTS[label2][cell.Value]; ok {
					cell.Value = newValue
				}
			}
		}
	}

}

func AggregateForPrometheus(dscData *dscparser.DSCData, config config.Config) {

	var newDatasets []dscparser.Dataset
	for i := range dscData.Datasets {
		dataset := &dscData.Datasets[i]
		metricConfig, ok := config.Prometheus.Metrics[dataset.Name]
		if !ok {
			continue
		}
		ReplaceLabels(dataset)

		//Aggregate dimension 1
		label1 := dataset.DimensionInfo[0].Type
		if metricConfig.IsEliminateDimension(label1) {
			EliminateDimensionOne(&dscData.Datasets[i])
		}
		if isFilter, allowedValues := metricConfig.IsFilter(label1); isFilter {
			FilterDimensionOne(dataset, allowedValues)
		}

		//Aggregate dimension 2
		label2 := dataset.DimensionInfo[1].Type
		if metricConfig.IsEliminateDimension(label2) {
			EliminateDimensionTwo(&dscData.Datasets[i])
		}
		if isBucket, params := metricConfig.IsMaxCells(label2); isBucket { // max cells only works on dimension 2
			MaxCells(&dscData.Datasets[i], params.X)
		}
		if isFilter, allowedValues := metricConfig.IsFilter(label2); isFilter {
			FilterDimensionTwo(dataset, allowedValues)
		}

		newDatasets = append(newDatasets, *dataset)
	}

	dscData.Datasets = newDatasets
}
