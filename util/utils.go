package util

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/model"
	"github.com/wcharczuk/go-chart"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

var Me tgbotapi.User
var Bot *tgbotapi.BotAPI

func LoadInputFile(filename string) error {
	changed, err := RefreshInputFile()
	if err != nil {
		log.Printf("failed to refresh the file/url, error: %s", err)
	} else {
		if changed {
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

			conf.Stats = nil
			for _, rawStat := range rawStats {
				stat := model.Stat{
					DateOfPublication: rawStat.DateOfPublication,
					MunicipalityName:  rawStat.MunicipalityName,
					Province:          rawStat.Province,
					TotalReported:     rawStat.TotalReported,
					HospitalAdmission: rawStat.HospitalAdmission,
					Deceased:          rawStat.Deceased,
				}
				conf.Stats = append(conf.Stats, stat)
			}
			rawStats = nil
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
		} else {
			log.Printf("input file did not change, not reloading...")
		}
	}
	return err
}

// Return true if the file (URL) has changed
func RefreshInputFile() (bool, error) {
	var err error
	log.Printf("downloading new data from %s ...", conf.RIVMDownloadURL)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	resp, err := client.Get(conf.RIVMDownloadURL)
	if err != nil {
		log.Printf("failed to download the RIVM data from %s, error: %s", conf.RIVMDownloadURL, err)
	} else {
		if resp.StatusCode != http.StatusOK {
			if resp != nil {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("failed to download the RIVM data from %s, statuscode: %s, response: %s", conf.RIVMDownloadURL, resp.Status, body)
			} else {
				log.Printf("failed to download the RIVM data from %s, statuscode: %s", conf.RIVMDownloadURL, resp.Status)
			}
		} else {
			defer resp.Body.Close()
			// Create the file
			newFileName := fmt.Sprintf("%s.new", conf.InputFile)
			newFile, err := os.Create(newFileName)
			if err != nil {
				log.Printf("failed to create file %s, error: %s", newFileName, err)
			} else {
				defer newFile.Close()
				// Write the body to file
				_, err = io.Copy(newFile, resp.Body)
				fileContents, err := ioutil.ReadFile(newFileName) // quite inefficient to read the whole file again, just to calculate the hash
				if err != nil {
					log.Printf("failed reading file, error: %s", err)
				} else {
					newHashValue := fmt.Sprintf("%x", md5.Sum(fileContents))
					log.Printf("md5 sum of %s: %s", newFileName, newHashValue)
					if newHashValue != conf.HashValueOfInputFile {
						if _, err := os.Stat(conf.InputFile); os.IsExist(err) {
							err = os.Remove(conf.InputFile)
							if err != nil {
								log.Printf("failed to remove input file %s, error: %s", conf.InputFile, err)
								return false, err
							}
						} else {
							err = os.Rename(newFileName, conf.InputFile)
							if err != nil {
								log.Printf("failed to rename %s to %s, error: %s", newFileName, conf.InputFile, err)
							} else {
								log.Printf("renamed file %s to %s", newFileName, conf.InputFile)
								conf.HashValueOfInputFile = newHashValue
								return true, err
							}
						}
					} else {
						log.Printf("hash value (%s) of %s is same as hash value of %s (%s)", newHashValue, newFileName, conf.HashValueOfInputFile, conf.InputFile)
					}
				}
			}
		}
	}
	return false, err
}

func GetChartFile(chartInput *model.ChartInput) (*os.File, error) {
	var err error
	var file *os.File
	//casesStyle := chart.Style{FillColor: drawing.ColorFromHex("9ddceb")}
	//hospitalStyle := chart.Style{FillColor: drawing.ColorFromHex("63c522")}
	//deceasedStyle := chart.Style{FillColor: drawing.ColorFromHex("ff7654")}
	casesStyle := chart.Style{}
	hospitalStyle := chart.Style{}
	deceasedStyle := chart.Style{}

	var series []chart.Series
	series1 := chart.TimeSeries{Name: "Besmettingen", Style: casesStyle, XValues: chartInput.TimeStamps, YValues: chartInput.Cases}
	series2 := chart.TimeSeries{Name: "ZKH opnames", Style: hospitalStyle, XValues: chartInput.TimeStamps, YValues: chartInput.Hospital, YAxis: chart.YAxisSecondary}
	series3 := chart.TimeSeries{Name: "Overleden", Style: deceasedStyle, XValues: chartInput.TimeStamps, YValues: chartInput.Deceased, YAxis: chart.YAxisSecondary}
	series = append(series, series1, series2, series3)

	graph := chart.Chart{
		Title: chartInput.Title,
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
			Range: &chart.ContinuousRange{Min: 0, Max: float64(chartInput.HighestYAxisSec)},
		},
		YAxis: chart.YAxis{
			Name: "Besmettingen",
			ValueFormatter: func(v interface{}) string {
				if vf, isFloat := v.(float64); isFloat {
					return fmt.Sprintf("%0.0f", vf)
				}
				return ""
			},
			Range: &chart.ContinuousRange{Min: 0, Max: float64(chartInput.HighestYAxis)},
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

func GetChartInputCountry() (*model.ChartInput, error) {
	var err error
	var totalStats int
	var casesByDate = make(map[int64]int)
	var hospitalByDate = make(map[int64]int)
	var deceasedByDate = make(map[int64]int)
	for ix, stat := range conf.Stats {
		casesByDate[stat.DateOfPublication.Time().Unix()] = casesByDate[stat.DateOfPublication.Time().Unix()] + stat.TotalReported
		hospitalByDate[stat.DateOfPublication.Time().Unix()] = hospitalByDate[stat.DateOfPublication.Time().Unix()] + stat.HospitalAdmission
		deceasedByDate[stat.DateOfPublication.Time().Unix()] = deceasedByDate[stat.DateOfPublication.Time().Unix()] + stat.Deceased
		totalStats = ix
	}
	log.Printf("processed %d stats voor whole country", totalStats)
	// do the sorting
	keys := make([]int, 0, len(casesByDate))
	for k := range casesByDate {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	var cases []float64
	var deceased []float64
	var hospital []float64
	var xValues []time.Time
	var highestYaxis, highestYaxisSec int
	for _, key := range keys {
		key64 := int64(key)
		xValues = append(xValues, time.Unix(key64, 0))
		cases = append(cases, float64(casesByDate[key64]))
		hospital = append(hospital, float64(hospitalByDate[key64]))
		deceased = append(deceased, float64(deceasedByDate[key64]))
		if casesByDate[int64(key)] > highestYaxis {
			highestYaxis = casesByDate[key64]
		}
		if hospitalByDate[int64(key)] > highestYaxisSec {
			highestYaxisSec = hospitalByDate[key64]
		}
		if deceasedByDate[int64(key)] > highestYaxisSec {
			highestYaxisSec = deceasedByDate[key64]
		}
	}

	return &model.ChartInput{
		Title:           "NL",
		TimeStamps:      xValues,
		Cases:           cases,
		Hospital:        hospital,
		Deceased:        deceased,
		HighestYAxisSec: highestYaxisSec,
		HighestYAxis:    highestYaxis,
	}, err
}

// Returns the XAxis, and YAxis values (besmettingen, hospital, deceased) and the highest values for left and right YAxis
func GetChartInput(city string) (*model.ChartInput, error) {
	var err error
	var cases []float64
	var deceased []float64
	var hospital []float64
	var xValues []time.Time
	var highestYaxis, highestYaxisSec int
	if city == "" {
		return &model.ChartInput{}, errors.New("geen gemeente opgegeven")
	}
	var filteredStats []model.Stat
	for _, stat := range conf.Stats {
		if strings.ToLower(stat.MunicipalityName) == strings.ToLower(city) {
			filteredStats = append(filteredStats, stat)
		}
	}
	if len(filteredStats) == 0 {
		return &model.ChartInput{}, errors.New(fmt.Sprintf("Gemeente %s niet gevonden", city))
	}
	//log.Printf("found %d observations for city %s", len(filteredStats), city)

	for ix, stat := range filteredStats {
		if ix > conf.MaxPlots {
			break
		}
		cases = append(cases, float64(stat.TotalReported))
		hospital = append(hospital, float64(stat.HospitalAdmission))
		deceased = append(deceased, float64(stat.Deceased))
		if stat.TotalReported > highestYaxis {
			highestYaxis = stat.TotalReported
		}
		if stat.Deceased > highestYaxisSec {
			highestYaxisSec = stat.Deceased
		}
		//log.Printf("%v: %d %d %d", stat.DateOfPublication.Time(), stat.TotalReported, stat.HospitalAdmission, stat.Deceased)
		xValues = append(xValues, stat.DateOfPublication.Time())
	}

	chartInput := model.ChartInput{
		TimeStamps:      xValues,
		Cases:           cases,
		Hospital:        hospital,
		Deceased:        deceased,
		HighestYAxisSec: highestYaxisSec,
		HighestYAxis:    highestYaxis,
	}
	return &chartInput, err
}

// Returns if we are mentioned and if we were commanded
func TalkOrCmdToMe(update *tgbotapi.Update) (bool, bool) {
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

func HandleCommand(update *tgbotapi.Update) {
	if strings.HasPrefix(update.Message.Text, "/help") {
		log.Printf("help text requested by %s", update.Message.From)
		_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, conf.HelpText))
		return
	}

	if strings.HasPrefix(update.Message.Text, "/gemeente") {
		words := strings.Split(update.Message.Text, " ")
		if len(words) > 1 {
			city := update.Message.Text[len("/gemeente")+1 : len(update.Message.Text)]
			chartInput, err := GetChartInput(city)
			if err != nil {
				msg := fmt.Sprintf("Fout bij het genereren van de grafiek data voor gemeente %s, fout: %s", city, err)
				log.Printf(msg)
				_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
			} else {
				chartFile, err := GetChartFile(chartInput)
				if err != nil {
					msg := fmt.Sprintf("Fout bij het genereren van de grafiek voor gemeente %s, fout: %s", city, err)
					log.Printf(msg)
					_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
				} else {
					photoConfig := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, chartFile.Name())
					_, err = Bot.Send(photoConfig)
				}
			}
		} else {
			_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "geeft een gemeente naam op, b.v.:  /gemeente Rotterdam"))
		}
		return
	}

	if strings.HasPrefix(update.Message.Text, "/land") {
		chartInput, err := GetChartInputCountry()
		if err == nil {
			chartFile, err := GetChartFile(chartInput)
			if err != nil {
				msg := fmt.Sprintf("Fout bij het genereren van de grafiek voor land, fout: %s", err)
				log.Printf(msg)
				_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
			} else {
				photoConfig := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, chartFile.Name())
				_, err = Bot.Send(photoConfig)
			}
			return
		} else {
			msg := fmt.Sprintf("Fout bij het genereren van de grafiek date voor land, fout: %s", err)
			log.Printf(msg)
			_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		}
		return
	}

	if strings.HasPrefix(update.Message.Text, "/laatsteweek") {
		var casesByDate = make(map[int64]int)
		var hospitalByDate = make(map[int64]int)
		var deceasedByDate = make(map[int64]int)
		sevenDaysAgo := time.Now().Add(time.Hour * -24 * 7)
		for _, stat := range conf.Stats {
			if stat.DateOfPublication.Time().After(sevenDaysAgo) {
				casesByDate[stat.DateOfPublication.Time().Unix()] = casesByDate[stat.DateOfPublication.Time().Unix()] + stat.TotalReported
				hospitalByDate[stat.DateOfPublication.Time().Unix()] = hospitalByDate[stat.DateOfPublication.Time().Unix()] + stat.HospitalAdmission
				deceasedByDate[stat.DateOfPublication.Time().Unix()] = deceasedByDate[stat.DateOfPublication.Time().Unix()] + stat.Deceased
			}
		}
		// do the sorting
		keys := make([]int, 0, len(casesByDate))
		for k := range casesByDate {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		msg := "```\ndatum      besmet    zkh    overldn\n"
		for _, key := range keys {
			key64 := int64(key)
			msg = fmt.Sprintf("%s%s", msg, fmt.Sprintf("%s  %5d   %4d    %4d\n", time.Unix(key64, 0).Format(conf.DateFormat), casesByDate[key64], hospitalByDate[key64], deceasedByDate[key64]))
		}
		msg = fmt.Sprintf("%s\n```", msg)
		msgConfig := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
		msgConfig.ParseMode = tgbotapi.ModeMarkdown
		_, _ = Bot.Send(msgConfig)
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
