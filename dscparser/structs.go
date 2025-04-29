package dscparser

import (
	"cmp"
	"encoding/xml"
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
	for _, dataset := range d.Datasets {
		dataset.Sort()
	}
}

func (d DSCData) Equals(d2 DSCData) bool {
	if len(d.Datasets) != len(d2.Datasets) || d.NameServer != d2.NameServer || d.Location != d2.Location {
		return false
	}
	d.Sort()
	d2.Sort()
	for i := range d.Datasets {
		if !d.Datasets[i].Equals(d2.Datasets[i]) {
			return false
		}
	}
	return true
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
	//Sort rows
	slices.SortFunc(ds.Data.Rows, cmpRow)

	//Sort cells
	for _, row := range ds.Data.Rows {
		row.Sort()
	}
}

func (ds Dataset) Equals(ds2 Dataset) bool {
	if len(ds.DimensionInfo) != len(ds2.DimensionInfo) || ds.Dimensions != ds2.Dimensions ||
		ds.StartTime != ds2.StartTime || ds.StopTime != ds2.StopTime ||
		ds.Name != ds2.Name || len(ds.Data.Rows) != len(ds2.Data.Rows) {
		return false
	}
	for i := range ds.DimensionInfo {
		if ds.DimensionInfo[i].Number != ds2.DimensionInfo[i].Number || ds.DimensionInfo[i].Type != ds2.DimensionInfo[i].Type {
			return false
		}
	}
	ds.Sort()
	ds2.Sort()
	for i := range ds.Data.Rows {
		if !ds.Data.Rows[i].Equals(ds2.Data.Rows[i]) {
			return false
		}
	}
	return true
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

func (r *Row) Sort() {
	cmpCell := func(a, b Cell) int {
		return cmp.Compare(a.Value, b.Value)
	}
	slices.SortFunc(r.Cells, cmpCell)
}

func (r Row) Equals(r2 Row) bool {
	if len(r.Cells) != len(r2.Cells) || r.XMLName.Local != r2.XMLName.Local || r.Value != r2.Value {
		return false
	}
	r.Sort()
	r2.Sort()
	for i := range r.Cells {
		if !r.Cells[i].Equals(r2.Cells[i]) {
			return false
		}
	}
	return true
}

type Cell struct {
	XMLName xml.Name
	Value   string `xml:"val,attr"`
	Count   int    `xml:"count,attr"`
}

func (c Cell) Equals(c2 Cell) bool {
	return c.XMLName.Local == c2.XMLName.Local && c.Value == c2.Value && c.Count == c2.Count
}
