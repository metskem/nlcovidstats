package util

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/model"
	"github.com/wcharczuk/go-chart/v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const httpTimeout = time.Second * 60
const MagicTimeLastModified = "Mon, 2 Jan 2006 15:04:05 MST"

var Me tgbotapi.User
var Bot *tgbotapi.BotAPI
var downloadedOnce = false

var totalStats int
var casesByDate = make(map[int64]int)
var hospitalByDate = make(map[int64]int)
var deceasedByDate = make(map[int64]int)

func LoadInputFile(filename string) error {
	var err error
	var fileChanged = true
	if downloadedOnce {
		fileChanged = checkIfFileChanged()
	}
	if fileChanged {
		changed, err := refreshInputFile()
		if err != nil {
			log.Printf("fout bij verversen bestand/url : %s", err)
		} else {
			downloadedOnce = true
			if changed {
				log.Printf("lezen bestand %s", filename)
				file, err := ioutil.ReadFile(filename)
				if err != nil {
					log.Printf("fout bij lezen bestand %s: %s", filename, err)
					return err
				}
				log.Printf("json-parsen bestand %s", filename)
				var rawStats []model.RawStat
				err = json.Unmarshal(file, &rawStats)
				if err != nil {
					log.Printf("fout bij unmarshalling json bestand %s : %s", filename, err)
					return err
				}
				file = nil
				log.Printf("we found %d elements", len(rawStats))

				casesByDate = make(map[int64]int)
				hospitalByDate = make(map[int64]int)
				deceasedByDate = make(map[int64]int)

				for ix, rawStat := range rawStats {
					casesByDate[rawStat.DateOfPublication.Time().Unix()] = casesByDate[rawStat.DateOfPublication.Time().Unix()] + rawStat.TotalReported
					hospitalByDate[rawStat.DateOfPublication.Time().Unix()] = hospitalByDate[rawStat.DateOfPublication.Time().Unix()] + rawStat.HospitalAdmission
					deceasedByDate[rawStat.DateOfPublication.Time().Unix()] = deceasedByDate[rawStat.DateOfPublication.Time().Unix()] + rawStat.Deceased
					totalStats = ix
				}
				log.Printf("%d stats gelezen", totalStats)
				rawStats = nil
			} else {
				log.Printf("invoer bestand niet gewijzigd, niet herladen...")
			}
		}
	}
	return err
}

/** Check max 30 times each 30 secs if file has changed, if lastModified.Day is today then return true */
func checkIfFileChanged() bool {
	now := time.Now()
	client := &http.Client{Timeout: httpTimeout}
	var maxTries = 30
	for i := 0; i < maxTries; i++ {
		response, err := client.Head(conf.RIVMDownloadURL)
		if err != nil {
			log.Printf("fout bij http HEAD %s: %s", conf.RIVMDownloadURL, err)
			return false
		} else {
			lastModifiedStr := response.Header.Get("Last-Modified")
			lastModified, err := time.Parse(MagicTimeLastModified, lastModifiedStr)
			if err != nil {
				log.Printf("fout bij parsen van Last-Modified header (%s) : %s", lastModifiedStr, err)
			}
			log.Printf("(%d/%d) Last-Modified voor %s: %d %d:%d", i, maxTries, conf.RIVMDownloadURL, lastModified.Day(), lastModified.Hour(), lastModified.Minute())
			if lastModified.Day() == now.Day() {
				return true
			}
		}
		time.Sleep(time.Second * 30)
	}
	return false
}

// Return true if the file (URL) has changed
func refreshInputFile() (bool, error) {
	var err error
	log.Printf("downloaden nieuwe data van %s ...", conf.RIVMDownloadURL)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: httpTimeout,
	}
	resp, err := client.Get(conf.RIVMDownloadURL)
	if err != nil {
		log.Printf("fout bij downloaden RIVM data van %s: %s", conf.RIVMDownloadURL, err)
	} else {
		if resp.StatusCode != http.StatusOK {
			if resp != nil {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("fout bij downloaden RIVM data van %s, statuscode: %s, response: %s", conf.RIVMDownloadURL, resp.Status, body)
			} else {
				log.Printf("fout bij downloaden RIVM data van %s, statuscode: %s", conf.RIVMDownloadURL, resp.Status)
			}
		} else {
			defer resp.Body.Close()
			// Create the file
			newFileName := fmt.Sprintf("%s.new", conf.InputFile)
			newFile, err := os.Create(newFileName)
			if err != nil {
				log.Printf("fout bij aanmaken bestand %s: %s", newFileName, err)
			} else {
				defer newFile.Close()
				// Write the body to file
				_, err = io.Copy(newFile, resp.Body)
				fileContents, err := ioutil.ReadFile(newFileName) // quite inefficient to read the whole file again, just to calculate the hash
				if err != nil {
					log.Printf("fout bij inlezen bestand %s: %s", newFileName, err)
				} else {
					newHashValue := fmt.Sprintf("%x", md5.Sum(fileContents))
					log.Printf("md5 sum van %s: %s", newFileName, newHashValue)
					if newHashValue != conf.HashValueOfInputFile {
						if _, err := os.Stat(conf.InputFile); os.IsExist(err) {
							err = os.Remove(conf.InputFile)
							if err != nil {
								log.Printf("fout bij verwijderen van invoer bestand %s: %s", conf.InputFile, err)
								return false, err
							}
						} else {
							err = os.Rename(newFileName, conf.InputFile)
							if err != nil {
								log.Printf("fout bij hernoemen %s naar %s: %s", newFileName, conf.InputFile, err)
							} else {
								log.Printf("bestand %s hernoemd naar %s", newFileName, conf.InputFile)
								conf.HashValueOfInputFile = newHashValue
								return true, err
							}
						}
					} else {
						log.Printf("hash waarde (%s) van %s is gelijk aan hash waarde van %s (%s)", newHashValue, newFileName, conf.HashValueOfInputFile, conf.InputFile)
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
		msg := fmt.Sprintf("fout bij het renderen van de grafiek: %s", err)
		log.Print(msg)
	}
	return file, err
}

func GetChartInput() (*model.ChartInput, error) {
	var err error
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

	if strings.HasPrefix(update.Message.Text, "/grafiek") {
		chartInput, err := GetChartInput()
		if err == nil {
			chartFile, err := GetChartFile(chartInput)
			if err != nil {
				msg := fmt.Sprintf("Fout bij het genereren van de grafiek: %s", err)
				log.Printf(msg)
				_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
			} else {
				photoConfig := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, chartFile.Name())
				_, err = Bot.Send(photoConfig)
			}
			return
		} else {
			msg := fmt.Sprintf("Fout bij het genereren van de grafiek: %s", err)
			log.Printf(msg)
			_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		}
		return
	}

	if strings.HasPrefix(update.Message.Text, "/recent") {
		msg := GetRecentData()
		msgConfig := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
		msgConfig.ParseMode = tgbotapi.ModeMarkdown
		_, _ = Bot.Send(msgConfig)
		return
	}

	_, _ = Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Dit heb ik niet begrepen.\n%s", conf.HelpText)))
}

func GetRecentData() string {
	var casesByDateLastXDays = make(map[int64]int)
	var hospitalByDateLastXDays = make(map[int64]int)
	var deceasedByDateLastXDays = make(map[int64]int)
	tenDaysAgo := time.Now().Add(time.Hour * -24 * 10).Unix()
	for date, _ := range casesByDate {
		if date > tenDaysAgo {
			casesByDateLastXDays[date] = casesByDate[date]
			hospitalByDateLastXDays[date] = hospitalByDate[date]
			deceasedByDateLastXDays[date] = deceasedByDate[date]
		}
	}
	// do the sorting
	keys := make([]int, 0, len(casesByDateLastXDays))
	for k := range casesByDateLastXDays {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	msg := "```\ndatum      besmet  zkh  overldn\n"
	for _, key := range keys {
		key64 := int64(key)
		msg = fmt.Sprintf("%s%s", msg, fmt.Sprintf("%s  %5d %4d  %4d\n", time.Unix(key64, 0).Format(conf.DateFormat), casesByDateLastXDays[key64], hospitalByDateLastXDays[key64], deceasedByDateLastXDays[key64]))
	}
	return fmt.Sprintf("%s\n```", msg)
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

func InDST(t time.Time) bool {
	jan1st := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()) // January 1st is always outside DST window
	_, off1 := t.Zone()
	_, off2 := jan1st.Zone()
	return off1 != off2
}
