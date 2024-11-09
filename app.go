package main

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log"
	"strings"
	"time"
)

const (
	highTempThreshold = 50.0
	sleepDuration     = 10 * time.Minute
)

type Application struct {
	conf       *Config
	bot        *bot.Bot
	bootTime   time.Time
	tempLoader TemperatureLoader
}

func NewApplication(conf *Config) (*Application, error) {
	permission := func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
			if update.Message != nil && conf.TargetId != update.Message.Chat.ID {
				return
			}
			if update.CallbackQuery != nil && conf.TargetId != update.CallbackQuery.From.ID {
				return
			}
			next(ctx, bot, update)
		}
	}
	b, err := bot.New(
		conf.Token,
		bot.WithSkipGetMe(),
		bot.WithMiddlewares(permission),
	)
	if err != nil {
		return nil, err
	}
	return &Application{
		conf:       conf,
		bot:        b,
		bootTime:   time.Now(),
		tempLoader: LoadSensorsTemperature,
	}, nil
}

func (a *Application) sendTemperatureToTelegram(ctx context.Context, render func(*SensorsTemperature) string) {
	temp, err := a.tempLoader()
	if err != nil {
		log.Printf("error getting system temp: %v", err)
		return
	}
	log.Println(RenderLogMessage(temp))
	var sb strings.Builder
	sb.WriteString(render(temp))
	sb.WriteString("\n\n")
	sb.WriteString("Uptime: ")
	sb.WriteString(time.Since(a.bootTime).String())
	sb.WriteString(" ; Boot time: ")
	sb.WriteString(a.bootTime.Format(time.DateTime))
	_, err = a.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:              a.conf.TargetId,
		Text:                sb.String(),
		ParseMode:           models.ParseModeHTML,
		DisableNotification: temp.IsHigherThanThreshold(highTempThreshold),
	})
	if err != nil {
		log.Printf("error sending message: %v", err)
	}
}

func (a *Application) startMonitoring(ctx context.Context) {
	for {
		a.sendTemperatureToTelegram(ctx, RenderTableMessage)
		time.Sleep(sleepDuration)
	}
}

func (a *Application) startPolling(ctx context.Context) {
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "/temp", bot.MatchTypePrefix, func(ctx context.Context, bot *bot.Bot, update *models.Update) {
		a.sendTemperatureToTelegram(ctx, RenderTableMessage)
	})
	_, _ = a.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "temp",
				Description: "/temp - get system temperature",
			},
		},
	})
	a.bot.Start(ctx)
}
