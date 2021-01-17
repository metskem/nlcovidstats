package util

import (
	"errors"
	"fmt"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/model"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"log"
	"os"
	"time"
)

func CreateChartFile(city string) (*os.File, error) {
	var err error
	var file *os.File
	if city == "" {
		return nil, errors.New("no city specified")
	}
	var filteredStats []model.Stat
	for _, stat := range conf.Stats {
		if stat.MunicipalityName == city {
			filteredStats = append(filteredStats, stat)
		}
	}
	if len(filteredStats) == 0 {
		return file, errors.New(fmt.Sprintf("city %s not found", city))
	}
	log.Printf("found %d observations for city %s", len(filteredStats), city)

	var cumulative = false
	var cases []float64
	var deceased []float64
	var hospital []float64
	var xValues []time.Time
	var highestYaxis, highestYaxisSec, cumulCases, cumulHospital, cumulDeceased int
	var casesStyle, hospitalStyle, deceasedStyle chart.Style
	for ix, stat := range filteredStats {
		if ix > conf.MaxPlots {
			break
		}
		if cumulative {
			cumulCases = cumulCases + stat.TotalReported
			cumulHospital = cumulHospital + stat.HospitalAdmission
			cumulDeceased = cumulDeceased + stat.Deceased
			cases = append(cases, float64(cumulCases))
			hospital = append(hospital, float64(cumulHospital))
			deceased = append(deceased, float64(cumulDeceased))
		} else {
			cases = append(cases, float64(stat.TotalReported))
			hospital = append(hospital, float64(stat.HospitalAdmission))
			deceased = append(deceased, float64(stat.Deceased))
			if stat.TotalReported > highestYaxis {
				highestYaxis = stat.TotalReported
			}
			if stat.Deceased > highestYaxisSec {
				highestYaxisSec = stat.Deceased
			}
		}
		//log.Printf("%v: %d %d %d", stat.DateOfPublication.Time(), stat.TotalReported, stat.HospitalAdmission, stat.Deceased)
		xValues = append(xValues, stat.DateOfPublication.Time())
	}

	if cumulative {
		highestYaxis = cumulCases
		highestYaxisSec = cumulDeceased
	} else {
		casesStyle = chart.Style{FillColor: drawing.ColorFromHex("9ddceb")}
		hospitalStyle = chart.Style{FillColor: drawing.ColorFromHex("63c522")}
		deceasedStyle = chart.Style{FillColor: drawing.ColorFromHex("ff7654")}
	}
	log.Printf("rendering %d plots for city %s ", len(xValues), city)
	var series []chart.Series
	chart1 := chart.TimeSeries{Name: "Besmettingen", Style: casesStyle, XValues: xValues, YValues: cases}
	chart2 := chart.TimeSeries{Name: "ZKH opnames", Style: hospitalStyle, XValues: xValues, YValues: hospital, YAxis: chart.YAxisSecondary}
	chart3 := chart.TimeSeries{Name: "Overleden", Style: deceasedStyle, XValues: xValues, YValues: deceased, YAxis: chart.YAxisSecondary}
	series = append(series, chart1, chart2, chart3)

	graph := chart.Chart{
		Title: city,
		XAxis: chart.XAxis{
			ValueFormatter: chart.TimeValueFormatterWithFormat("2006-01-02"),
		},
		YAxisSecondary: chart.YAxis{
			Name: "ZKH opnames / Overleden",
			//Style: chart.Style{
			//	TextRotationDegrees: 90,
			//},
			ValueFormatter: func(v interface{}) string {
				if vf, isFloat := v.(float64); isFloat {
					return fmt.Sprintf("%0.0f", vf)
				}
				return ""
			},
			Range: &chart.ContinuousRange{Min: 0, Max: float64(highestYaxisSec),
			},
		},
		YAxis: chart.YAxis{
			Name: "Besmettingen",
			ValueFormatter: func(v interface{}) string {
				if vf, isFloat := v.(float64); isFloat {
					return fmt.Sprintf("%0.0f", vf)
				}
				return ""
			},
			Range: &chart.ContinuousRange{Min: 0, Max: float64(highestYaxis),
			},
		},
		Background: chart.Style{Padding: chart.Box{Left: 125, Right: 25}},
		Series:     series,
	}

	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	file, _ = os.Create("result.png")
	defer file.Close()
	err = graph.Render(chart.PNG, file)
	if err != nil {
		msg := fmt.Sprintf("failed to render graph, error: %s", err)
		log.Print(msg)
	}
	return file, err
}
