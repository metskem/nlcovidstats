package util

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/model"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var Me tgbotapi.User
var Bot *tgbotapi.BotAPI

// TODO als de input file er niet is, dan gewoon maken en niet falen
func LoadInputFile(filename string) error {
	var err error
	// for now we read the (already downloaded) json file, later we will download it from https://data.rivm.nl/covid-19/COVID-19_aantallen_gemeente_per_dag.json
	RefreshInputFile(filename)
	log.Printf("reading file %s", filename)
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("failed reading input file %s: %s\n", filename, err)
		return err
	}

	log.Printf("json-parsing file %s", filename)
	var rawStats []model.RawStat
	err = json.Unmarshal(file, &rawStats)
	if err != nil {
		log.Printf("failed unmarshalling json from file %s, error: %s\n", filename, err)
		return err
	}
	log.Printf("we found %d elements", len(rawStats))

	for _, rawStat := range rawStats {
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
		log.Printf("writing to file %s", conf.OutputFile)
		err = ioutil.WriteFile(conf.OutputFile, ba, os.ModePerm)
		if err != nil {
			log.Printf("failed to write output file %s, error: %s", conf.OutputFile, err)
			return err
		}
	}
	return err
}

func RefreshInputFile(filename string) {
	log.Printf("downloading new data from %s ...", conf.RIVMDownloadURL)
	resp, err := http.Get(conf.RIVMDownloadURL)
	if err != nil {
		log.Printf("failed to download the RIVM data from %s, error: %s", conf.RIVMDownloadURL, err)
	} else {
		defer resp.Body.Close()

		// Create the file
		newFileName := fmt.Sprintf("%s.new", filename)
		newFile, err := os.Create(newFileName)
		if err != nil {
			log.Printf("failed to create file %s, error: %s", newFileName, err)
		} else {
			defer newFile.Close()
			// Write the body to file
			_, err = io.Copy(newFile, resp.Body)
			newHash := sha1.New()
			if _, err := io.Copy(newHash, newFile); err != nil {
				log.Printf("failed to calculate hash over %s, error: %s", newFileName, err)
			} else {
				newHashValue := hex.EncodeToString(newHash.Sum(nil))
				log.Printf("new hash value of input file %s : %s", newFileName, newHashValue)
				if newHashValue != conf.HashValueOfInputFile {
					err := os.Remove(conf.InputFile)
					if err != nil {
						log.Printf("failed to remove input file %s, error: %s", conf.InputFile, err)
					} else {
						err := os.Rename(newFileName, conf.InputFile)
						if err != nil {
							log.Printf("failed to rename %s to %s, error: %s", newFileName, conf.InputFile, err)
						} else {
							conf.HashValueOfInputFile = newHashValue
						}
					}
				} else {
					log.Printf("hash value (%s) of %s is same as hash value of %s (%s)", newHashValue, newFileName, conf.HashValueOfInputFile, conf.InputFile)
				}
			}
		}
	}
}

func GetChartFile(city string) (*os.File, error) {
	var err error
	var file *os.File
	if city == "" {
		return nil, errors.New("Geen gemeente opgegeven")
	}
	var filteredStats []model.Stat
	for _, stat := range conf.Stats {
		if strings.ToLower(stat.MunicipalityName) == strings.ToLower(city) {
			filteredStats = append(filteredStats, stat)
		}
	}
	if len(filteredStats) == 0 {
		return file, errors.New(fmt.Sprintf("Gemeente %s niet gevonden", city))
	}
	//log.Printf("found %d observations for city %s", len(filteredStats), city)

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
	//log.Printf("rendering %d plots for city %s ", len(xValues), city)
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
			Range: &chart.ContinuousRange{Min: 0, Max: float64(highestYaxisSec)},
		},
		YAxis: chart.YAxis{
			Name: "Besmettingen",
			ValueFormatter: func(v interface{}) string {
				if vf, isFloat := v.(float64); isFloat {
					return fmt.Sprintf("%0.0f", vf)
				}
				return ""
			},
			Range: &chart.ContinuousRange{Min: 0, Max: float64(highestYaxis)},
		},
		Background: chart.Style{Padding: chart.Box{Left: 125, Right: 25}},
		Series:     series,
	}

	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	file, _ = os.Create(fmt.Sprintf("%s/result.png", os.TempDir()))
	defer file.Close()
	err = graph.Render(chart.PNG, file)
	if err != nil {
		msg := fmt.Sprintf("failed to render graph, error: %s", err)
		log.Print(msg)
	}
	return file, err
}

// Returns if we are mentioned and if we were commanded
func TalkOrCmdToMe(update tgbotapi.Update) (bool, bool) {
	entities := update.Message.Entities
	var mentioned = false
	var botCmd = false
	if entities != nil {
		for _, entity := range *entities {
			if entity.Type == "mention" {
				if strings.HasPrefix(update.Message.Text, fmt.Sprintf("@%s", Me.UserName)) {
					mentioned = true
				}
			}
			if entity.Type == "bot_command" {
				botCmd = true
				if strings.Contains(update.Message.Text, fmt.Sprintf("@%s", Me.UserName)) {
					mentioned = true
				}
			}
		}
	}
	// if another bot was mentioned, the cmd is not for us
	if update.Message.Chat.IsGroup() && mentioned == false {
		botCmd = false
	}
	return mentioned, botCmd
}

func HandleCommand(update tgbotapi.Update) {
	if strings.HasPrefix(update.Message.Text, "/help") {
		log.Printf("help text requested by %s", update.Message.From)
		_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, conf.HelpText))
		return
	}

	if strings.HasPrefix(update.Message.Text, "/chart") {
		words := strings.Split(update.Message.Text, " ")
		if len(words) > 1 {
			city := update.Message.Text[len("/chart")+1 : len(update.Message.Text)]
			chartFile, err := GetChartFile(city)
			if err != nil {
				msg := fmt.Sprintf("Fout bij het genereren van de grafiek voor gemeente %s, fout: %s", city, err)
				log.Printf(msg)
				_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
			} else {
				photoConfig := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, chartFile.Name())
				_, err = Bot.Send(photoConfig)
			}
		} else {
			_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "geeft een gemeente naam op, b.v.:  /chart Rotterdam"))
		}
		return
	}

	_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Dit heb ik niet begrepen.\n%s", conf.HelpText)))
}

// nicked from https://stephenafamo.com/blog/better-scheduling-in-go/
func Cron(ctx context.Context, startTime time.Time, delay time.Duration) <-chan time.Time {
	// Create the channel which we will return
	stream := make(chan time.Time, 1)

	// Calculating the first start time in the future
	// Need to check if the time is zero (e.g. if time.Time{} was used)
	if !startTime.IsZero() {
		diff := time.Until(startTime)
		if diff < 0 {
			total := diff - delay
			times := total / delay * -1

			startTime = startTime.Add(times * delay)
		}
	}

	// Run this in a goroutine, or our function will block until the first event
	go func() {

		// Run the first event after it gets to the start time
		t := <-time.After(time.Until(startTime))
		stream <- t

		// Open a new ticker
		ticker := time.NewTicker(delay)
		// Make sure to stop the ticker when we're done
		defer ticker.Stop()

		// Listen on both the ticker and the context done channel to know when to stop
		for {
			select {
			case t2 := <-ticker.C:
				stream <- t2
			case <-ctx.Done():
				close(stream)
				return
			}
		}
	}()

	return stream
}
