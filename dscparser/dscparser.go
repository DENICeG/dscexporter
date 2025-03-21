package dscparser

import (
	"encoding/xml"
	"fmt"
	"os"
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

func ParseDataset(filePath string) *Dataset {
	var dataset Dataset

	fileContent, err := os.ReadFile(filePath)
	checkError(err)

	err = xml.Unmarshal(fileContent, &dataset)
	checkError(err)

	return &dataset
}
