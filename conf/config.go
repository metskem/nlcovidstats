package conf

import (
	"github.com/metskem/nlcovidstats/model"
	"log"
	"os"
)

var (
	// Variables to identify the build
	CommitHash string
	VersionTag string
	BuildTime  string

	Stats      []model.Stat
	BotToken   = os.Getenv("BOT_TOKEN")
	DebugStr   = os.Getenv("DEBUG")
	Debug      bool
	InputFile  = "input.json"
	OutputFile = "output.json"
	MaxPlots   = 1000
	//RIVMDownloadURL      = "http://www.computerhok.nl/input.json"
	RIVMDownloadURL      = "https://data.rivm.nl/covid-19/COVID-19_aantallen_gemeente_per_dag.json"
	HelpText             = "Deze Bot leest de (dagelijks actuele) data van RIVM, en genereert een grafiek voor de gemeente die u opgeeft\nDe volgende commando's kunt u geven:\n/help - Geeft deze tekst\n/gemeente <Gemeentenaam> - Geeft grafiek van gemeentenaam, b.v. /gemeente Rijssen-Holten"
	HashValueOfInputFile string
	//RefreshTime          = "18:12" // GMT
	RefreshTime = "14:25" // GMT
)

func EnvironmentComplete() {
	envComplete := true

	if len(BotToken) == 0 {
		log.Print("missing envvar \"BOT_TOKEN\"")
		envComplete = false
	}

	Debug = false
	if DebugStr == "true" {
		Debug = true
	}

	if !envComplete {
		log.Fatal("one or more envvars missing, aborting...")
	}
}
