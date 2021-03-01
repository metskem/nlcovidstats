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
	HelpText             = "Deze Bot leest dagelijks om 15:25 data van RIVM, en genereert grafieken en overzichten.\nDe volgende commando's kunt u geven:\n/help - Geeft deze tekst\n/laatsteweek - geeft de landelijke cijfers van de afgelopen 7 dagen\n/land - Geeft COVID-19 grafiek van heel NL\n/gemeente <Gemeentenaam> - Geeft COVID-19 grafiek van gemeentenaam, b.v. /gemeente Rijssen-Holten"
	HashValueOfInputFile string
	//RefreshTime          = "18:12" // GMT
	RefreshTime = "14:20" // GMT
	DateFormat  = "2006-01-02"
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
