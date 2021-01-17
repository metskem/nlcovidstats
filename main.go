package main

import (
	"encoding/json"
	"fmt"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/model"
	"github.com/metskem/nlcovidstats/util"
	"io/ioutil"
	"log"
	"os"
)

var outputFile = "output.json"

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("specify 2 arguments: <input file> <City Name>")
	}
	inputFile := os.Args[1]
	city := os.Args[2]
	log.Printf("reading file %s", inputFile)
	file, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("failed reading input file %s: %s\n", inputFile, err)
		os.Exit(8)
	}
	log.Printf("json-parsing file %s", inputFile)
	err = json.Unmarshal(file, &conf.RawStats)
	if err != nil {
		fmt.Printf("failed unmarshalling json from file %s, error: %s\n", inputFile, err)
		os.Exit(8)
	}
	log.Printf("we found %d elements", len(conf.RawStats))

	for _, rawStat := range conf.RawStats {
		stat := model.Stat{
			DateOfPublication:      rawStat.DateOfPublication,
			MunicipalityName:       rawStat.MunicipalityName,
			Province:               rawStat.Province,
			SecurityRegionName:     rawStat.SecurityRegionName,
			MunicipalHealthService: rawStat.MunicipalHealthService,
			TotalReported:          rawStat.TotalReported,
			HospitalAdmission:      rawStat.HospitalAdmission,
			Deceased:               rawStat.Deceased,
		}
		conf.Stats = append(conf.Stats, stat)
	}
	log.Print("json marshalling...")
	ba, err := json.MarshalIndent(conf.Stats, "", " ")
	if err != nil {
		log.Printf("failed json marshalling, error: %s", err)
	} else {
		log.Printf("writing to file %s", outputFile)
		err = ioutil.WriteFile(outputFile, ba, os.ModePerm)
		if err != nil {
			log.Printf("failed to write output file %s, error: %s", outputFile, err)
		} else {
			file, err := util.CreateChartFile(city)
			if err != nil {
				log.Printf("failed to create graph, error: %s", err)
			} else {
				log.Printf("graph for %s rendered to file %s", city, file.Name())
			}
		}
	}
}
