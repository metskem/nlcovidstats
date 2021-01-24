package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/metskem/nlcovidstats/conf"
	"github.com/metskem/nlcovidstats/util"
	"log"
	"os"
	"time"
)

func main() {

	conf.EnvironmentComplete()

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
		log.Printf("Started Bot: %s, version:%s, build time:%s, commit hash:%s", meDetails, conf.VersionTag, conf.BuildTime, conf.CommitHash)
	} else {
		log.Printf("Bot.GetMe() failed: %v", err)
	}

	err = util.LoadInputFile(conf.InputFile)
	if err != nil {
		log.Printf("failed loading input file %s, error: %s", conf.InputFile, err)
	}

	// refresh the inputfile every day at 15:25
	go func() {
		ctx := context.Background()
		now := time.Now()
		startTime, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%d-%s-%s %s", now.Year(), fmt.Sprintf("%02d", now.Month()), fmt.Sprintf("%02d", now.Day()), conf.RefreshTime))
		if err != nil {
			log.Printf("failed parsing start datetime, error: %s", err)
		} else {
			delay := time.Hour * 24
			for _ = range util.Cron(ctx, startTime, delay) {
				err = util.LoadInputFile(conf.InputFile)
				if err != nil {
					log.Printf("failed loading input file %s, error: %s", conf.InputFile, err)
				}
			}
		}
	}()
	newUpdate := tgbotapi.NewUpdate(0)
	newUpdate.Timeout = 60

	updatesChan, err := util.Bot.GetUpdatesChan(newUpdate)
	if err == nil {
		// announce that we are live again
		log.Printf("%s has been started, buildtime: %s", util.Me.UserName, conf.BuildTime)

		// start listening for messages, and optionally respond
		for update := range updatesChan {
			if update.Message == nil { // ignore any non-Message Updates
				log.Println("ignored null update")
			} else {
				chat := update.Message.Chat
				mentionedMe, cmdMe := util.TalkOrCmdToMe(update)

				// check if someone is talking to me:
				if (chat.IsPrivate() || (chat.IsGroup() && mentionedMe)) && update.Message.Text != "/start" {
					log.Printf("[%s] [chat:%d] %s", update.Message.From.UserName, chat.ID, update.Message.Text)
					util.HandleCommand(update)
				}

				// check if someone started a new chat
				if chat.IsPrivate() && cmdMe && update.Message.Text == "/start" {
					log.Printf("new chat added, chatid: %d, chat: %s (%s %s)", chat.ID, chat.UserName, chat.FirstName, chat.LastName)
				}

				// check if someone added me to a group
				if update.Message.NewChatMembers != nil && len(*update.Message.NewChatMembers) > 0 {
					log.Printf("new chat added, chatid: %d, chat: %s (%s %s)\n", chat.ID, chat.Title, chat.FirstName, chat.LastName)
				}

				// check if someone removed me from a group
				if update.Message.LeftChatMember != nil {
					leftChatMember := *update.Message.LeftChatMember
					if leftChatMember.UserName == util.Me.UserName {
						log.Printf("chat removed, chatid: %d, chat: %s (%s %s)\n", chat.ID, chat.Title, chat.FirstName, chat.LastName)
					}

				}
			}
			fmt.Println("")
		}
	} else {
		log.Printf("failed getting Bot updatesChannel, error: %v", err)
		os.Exit(8)
	}
}
