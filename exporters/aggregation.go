package exporters

import (
	"cmp"
	"dscexporter/config"
	"dscexporter/dscparser"
	"encoding/xml"
	"slices"
	"strings"
)

//ToDo: Keine MaxRows! Einfach keine Änderung.
//ToDo: MaxCells Testen

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

func FilterForPrometheus(dscData *dscparser.DSCData, config config.Config) {

	var newDatasets []dscparser.Dataset
	for i := range dscData.Datasets {
		dataset := &dscData.Datasets[i]
		metricConfig, ok := config.Prometheus.Metrics[dataset.Name]
		if !ok {
			continue
		}

		//label1 := dataset.DimensionInfo[0].Type
		label2 := dataset.DimensionInfo[1].Type

		aggregation, ok := metricConfig.Aggregations[label2]

		if ok && strings.EqualFold(aggregation.Type, "MaxCells") {
			MaxCells(&dscData.Datasets[i], aggregation.Params["x"])
		}

		newDatasets = append(newDatasets, *dataset)
	}

	dscData.Datasets = newDatasets
}
