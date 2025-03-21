package dscparser

import (
	"encoding/xml"
)

type DSCData struct {
	XMLName    xml.Name  `xml:"dscdata"`
	Datasets   []Dataset `xml:"array"`
	NameServer string
	Location   string
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
