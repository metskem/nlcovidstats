package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/util"
	"log"
	"os"
	"strings"
	"time"
)

const MagicTimeForStart = "2 January, 2006 15:04 (MST)"

func main() {

	//  used for memory profiling, import net/http/pprof
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	conf.EnvironmentComplete()
	log.SetOutput(os.Stdout)

	var err error

	util.Bot, err = tgbotapi.NewBotAPI(conf.BotToken)
	if err != nil {
		log.Panic(err.Error())
	}

	util.Bot.Debug = conf.Debug

	util.Me, err = util.Bot.GetMe()
	meDetails := "unknown"
	if err == nil {
		meDetails = fmt.Sprintf("BOT: ID:%d UserName:%s FirstName:%s LastName:%s", util.Me.ID, util.Me.UserName, util.Me.FirstName, util.Me.LastName)
		log.Printf("Bot gestart: %s, versie:%s, bouw-tijdstip:%s, commit hash:%s", meDetails, conf.VersionTag, conf.BuildTime, conf.CommitHash)
		log.Printf("Gevonden chat ids: %v", conf.ChatIDs)
	} else {
		log.Printf("Bot.GetMe() gefaald: %v", err)
	}

	err = util.LoadInputFile1()
	if err != nil {
		log.Printf("fout bij laden invoer bestand %s: %s", conf.InputFile1, err)
	}
	err = util.LoadInputFile2()
	if err != nil {
		log.Printf("fout bij laden invoer bestand %s: %s", conf.InputFile2, err)
	}

	// refresh the inputfile every day at 15:15
	go func() {
		ctx := context.Background()
		now := time.Now()
		tz := "CET"
		if util.InDST(now) {
			tz = "CEST"
		}
		startTime, err := time.Parse(MagicTimeForStart, fmt.Sprintf("%d %s, %d %s:%s (%s)", now.Day(), now.Month(), now.Year(), strings.Split(conf.RefreshTime, ":")[0], strings.Split(conf.RefreshTime, ":")[1], tz))
		if err != nil {
			log.Printf("fout bij parsen start datum-tijd: %s", err)
		} else {
			delay := time.Hour * 24
			log.Printf("herlaad schema met starttijd %s en vertraging %s", startTime, delay)
			for range util.Cron(ctx, startTime, delay) {
				if err = util.LoadInputFile1(); err != nil {
					log.Printf("fout bij laden invoer bestand %s: %s", conf.InputFile2, err)
					for _, id := range conf.ChatIDs {
						_, _ = util.Bot.Send(tgbotapi.NewMessage(id, fmt.Sprintf("Fout bij laden Nieuwe RIVM data: %s", err)))
					}
				} else {
					if err = util.LoadInputFile2(); err != nil {
						log.Printf("fout bij laden invoer bestand %s: %s", conf.InputFile2, err)
						for _, id := range conf.ChatIDs {
							_, _ = util.Bot.Send(tgbotapi.NewMessage(id, fmt.Sprintf("Fout bij laden Nieuwe RIVM data: %s", err)))
						}
					} else {
						for _, id := range conf.ChatIDs {
							msgConfig := tgbotapi.NewMessage(id, util.GetRecentData(1))
							msgConfig.ParseMode = tgbotapi.ModeMarkdown
							_, _ = util.Bot.Send(msgConfig)
						}
					}
				}
			}
		}
	}()
	newUpdate := tgbotapi.NewUpdate(0)
	newUpdate.Timeout = 60

	updatesChan, err := util.Bot.GetUpdatesChan(newUpdate)
	if err == nil {
		// announce that we are live again
		log.Printf("%s is gestart, bouw-tijdstip: %s", util.Me.UserName, conf.BuildTime)

		// start listening for messages, and optionally respond
		for update := range updatesChan {
			if update.Message == nil { // ignore any non-Message Updates
				log.Println("negeren null update")
			} else {
				chat := update.Message.Chat
				mentionedMe, cmdMe := util.TalkOrCmdToMe(&update)

				// check if someone is talking to me:
				if (chat.IsPrivate() || (chat.IsGroup() && mentionedMe)) && update.Message.Text != "/start" {
					log.Printf("[%s] [chat:%d] %s", update.Message.From.UserName, chat.ID, update.Message.Text)
					util.HandleCommand(&update)
				}

				// check if someone started a new chat
				if chat.IsPrivate() && cmdMe && update.Message.Text == "/start" {
					log.Printf("nieuwe chat toegevoegd, chatid: %d, chat: %s (%s %s)", chat.ID, chat.UserName, chat.FirstName, chat.LastName)
					_, _ = util.Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, conf.HelpText))
				}

				// check if someone added me to a group
				if update.Message.NewChatMembers != nil && len(*update.Message.NewChatMembers) > 0 {
					log.Printf("nieuwe chat toegevoegd, chatid: %d, chat: %s (%s %s)", chat.ID, chat.Title, chat.FirstName, chat.LastName)
					_, _ = util.Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, conf.HelpText))
				}

				// check if someone removed me from a group
				if update.Message.LeftChatMember != nil {
					leftChatMember := *update.Message.LeftChatMember
					if leftChatMember.UserName == util.Me.UserName {
						log.Printf("chat verwijderd, chatid: %d, chat: %s (%s %s)", chat.ID, chat.Title, chat.FirstName, chat.LastName)
					}

				}
			}
		}
	} else {
		log.Printf("fout bij ophalen Bot updatesChannel: %v", err)
		os.Exit(8)
	}
}
