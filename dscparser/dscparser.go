package dscparser

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ReadFile(filePath string, location string, nameServer string) *DSCData {

	//start := time.Now()
	var dscData DSCData

	fileContent, err := os.ReadFile(filePath)
	checkError(err)

	err = xml.Unmarshal(fileContent, &dscData)
	checkError(err)

	dscData.Location = location
	dscData.NameServer = nameServer

	//elapsed := time.Since(start)
	//log.Printf("Took %s\n", elapsed)

	return &dscData
}

func ReadFileWithCustomTimestamp(filePath string, location string, nameServer string, stopTime time.Time) *DSCData {

	dscData := ReadFile(filePath, location, nameServer)

	stopTimeUnix := stopTime.Unix()
	stopTimeUnix = stopTimeUnix - (stopTimeUnix % 60) // So the stoptime ends with a full minute

	for i := range dscData.Datasets {
		dscData.Datasets[i].StartTime = stopTimeUnix - 60
		dscData.Datasets[i].StopTime = stopTimeUnix
	}

	return dscData
}

func ParseDataset(filePath string) *Dataset {
	var dataset Dataset

	fileContent, err := os.ReadFile(filePath)
	checkError(err)

	err = xml.Unmarshal(fileContent, &dataset)
	checkError(err)

	return &dataset
}

func ParseRow(filePath string) *Row {
	var row Row

	fileContent, err := os.ReadFile(filePath)
	checkError(err)

	err = xml.Unmarshal(fileContent, &row)
	checkError(err)

	return &row
}
