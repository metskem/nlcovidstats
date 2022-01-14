package conf

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	// Variables to identify the build
	CommitHash string
	VersionTag string
	BuildTime  string

	BotToken   = os.Getenv("BOT_TOKEN")
	DebugStr   = os.Getenv("DEBUG")
	ChatIDsStr = os.Getenv("CHAT_IDS")
	ChatIDs    = make(map[int]int64)
	Debug      bool
	InputFile  = "input.json"
	//RIVMDownloadURL      = "https://www.computerhok.nl/input.json"
	RIVMDownloadURL      = "https://data.rivm.nl/covid-19/COVID-19_aantallen_gemeente_per_dag.json"
	HelpText             = "Deze Bot leest dagelijks om 15:15 data van RIVM, en genereert grafieken en overzichten.\nDe volgende commando's kunt u geven:\n/help - Geeft deze tekst\n/recent - geeft de landelijke cijfers van de afgelopen 10 dagen\n/grafiek - Geeft de landelijke COVID-19 grafiek"
	HashValueOfInputFile string
	RefreshTime          = "15:15" // local time
	DateFormat           = "2006-01-02"
)

func EnvironmentComplete() {
	envComplete := true

	if len(BotToken) == 0 {
		log.Print("ontbrekende envvar \"BOT_TOKEN\"")
		envComplete = false
	}

	Debug = false
	if DebugStr == "true" {
		Debug = true
	}

	chatIDsString := strings.Split(ChatIDsStr, ",")
	var chatids string
	for i := 0; i < len(chatIDsString); i++ {
		ChatIDs[i], _ = strconv.ParseInt(chatIDsString[i], 0, 64)
		chatids = fmt.Sprintf("%s %d", chatids, ChatIDs[i])
	}
	log.Printf("gevonden chat ids: %s\n", chatids)

	if !envComplete {
		log.Fatal("een of meer envvars ontbreken, afbreken...")
	}
}
