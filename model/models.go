package model

import (
	"encoding/json"
	"strings"
	"time"
)

type RawStats []struct{ RawStat }
type RawStat struct {
	DateOfReport           string                `json:"Date_of_report"`
	DateOfPublication      JsonDateOfPublication `json:"Date_of_publication"`
	MunicipalityCode       string                `json:"Municipality_code"`
	MunicipalityName       string                `json:"Municipality_name"`
	Province               string                `json:"Province"`
	SecurityRegionCode     string                `json:"Security_region_code"`
	SecurityRegionName     string                `json:"Security_region_name"`
	MunicipalHealthService string                `json:"Municipal_health_service"`
	ROAZRegion             string                `json:"ROAZ_region"`
	TotalReported          int                   `json:"Total_reported"`
	HospitalAdmission      int                   `json:"Hospital_admission"`
	Deceased               int                   `json:"Deceased"`
}

type Stats []struct{ Stat }
type Stat struct {
	DateOfPublication JsonDateOfPublication `json:"Date_of_publication"`
	MunicipalityName  string                `json:"Municipality_name"`
	Province          string                `json:"Province"`
	TotalReported     int                   `json:"Total_reported"`
	HospitalAdmission int                   `json:"Hospital_admission"`
	Deceased          int                   `json:"Deceased"`
}

type JsonDateOfPublication time.Time

func (j *JsonDateOfPublication) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*j = JsonDateOfPublication(t)
	return nil
}

func (j JsonDateOfPublication) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Time())
}

func (j JsonDateOfPublication) Time() time.Time {
	return time.Time(j)
}

type ChartInput struct {
	Title           string
	TimeStamps      []time.Time
	Cases           []float64
	Hospital        []float64
	Deceased        []float64
	HighestYAxisSec int
	HighestYAxis    int
}
