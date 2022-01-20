package model

import (
	"encoding/json"
	"strings"
	"time"
)

type RawStats []struct{ RawStat }
type RawStat struct {
	DateOfPublication JsonDateOfPublication `json:"Date_of_publication"`
	TotalReported     int                   `json:"Total_reported"`
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

type HospitalAdmissions []struct{ HospitalAdmission }

type HospitalAdmission struct {
	//Version                       int    `json:"Version"`
	DateOfReport string `json:"Date_of_report"`
	//DateOfStatisticsWeekStart     string `json:"Date_of_statistics_week_start"`
	//AgeGroup                      string `json:"Age_group"`
	//HospitalAdmissionNotification int    `json:"Hospital_admission_notification"`
	HospitalAdmission int `json:"Hospital_admission"`
	//ICAdmissionNotification       int    `json:"IC_admission_notification"`
	ICAdmission int `json:"IC_admission"`
}
