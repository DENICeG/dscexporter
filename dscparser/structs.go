package dscparser

import (
	"cmp"
	"encoding/xml"
	"reflect"
	"slices"
)

type DSCData struct {
	XMLName    xml.Name  `xml:"dscdata"`
	Datasets   []Dataset `xml:"array"`
	NameServer string
	Location   string
}

func (d *DSCData) Sort() {
	cmpDataset := func(a, b Dataset) int {
		return cmp.Compare(a.Name, b.Name)
	}

	//Sort datasets
	slices.SortFunc(d.Datasets, cmpDataset)

	//Sort rows and cells
	for i := range len(d.Datasets) {
		dataset := d.Datasets[i]
		dataset.Sort()
	}
}

func (d DSCData) Equals(d2 DSCData) bool {
	d.Sort()
	d2.Sort()
	return reflect.DeepEqual(d, d2)
}

type Dataset struct {
	XMLName       xml.Name        `xml:"array"`
	Name          string          `xml:"name,attr"`
	StartTime     int             `xml:"start_time,attr"`
	StopTime      int             `xml:"stop_time,attr"`
	Dimensions    int             `xml:"dimensions,attr"`
	DimensionInfo []DimensionInfo `xml:"dimension"`
	Data          Data            `xml:"data"`
}

func (ds *Dataset) Sort() {
	cmpRow := func(a, b Row) int {
		return cmp.Compare(a.Value, b.Value)
	}
	cmpCell := func(a, b Cell) int {
		return cmp.Compare(a.Value, b.Value)
	}

	//Sort rows
	slices.SortFunc(ds.Data.Rows, cmpRow)

	//Sort cells
	for i := range len(ds.Data.Rows) {
		ds1Row := ds.Data.Rows[i]
		slices.SortFunc(ds1Row.Cells, cmpCell)
	}
}

func (ds Dataset) Equals(ds2 Dataset) bool {
	ds.Sort()
	ds2.Sort()
	return reflect.DeepEqual(ds, ds2)
}

type DimensionInfo struct {
	Number int    `xml:"number,attr"`
	Type   string `xml:"type,attr"`
}

type Data struct {
	XMLName xml.Name `xml:"data"`
	Rows    []Row    `xml:",any"`
}

type Row struct {
	XMLName xml.Name
	Value   string `xml:"val,attr"`
	Cells   []Cell `xml:",any"`
}

type Cell struct {
	XMLName xml.Name
	Value   string `xml:"val,attr"`
	Count   int    `xml:"count,attr"`
}
