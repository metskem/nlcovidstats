package conf

import (
	"log"
	"os"
)

const ChatIDHarry = 337345957
const ChatIDClaudia = 1140134411
const ChatIDAnneke = 1366662634
const ChatIDEsther = 1674565467
const ChatIDWim = 1619715216

var IDs = []int64{ChatIDAnneke, ChatIDClaudia, ChatIDHarry, ChatIDEsther, ChatIDWim}

var (
	// Variables to identify the build
	CommitHash string
	VersionTag string
	BuildTime  string

	BotToken  = os.Getenv("BOT_TOKEN")
	DebugStr  = os.Getenv("DEBUG")
	Debug     bool
	InputFile = "input.json"
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

	if !envComplete {
		log.Fatal("een of meer envvars ontbreken, afbreken...")
	}
}
