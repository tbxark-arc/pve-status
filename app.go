package main

import (
	"context"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log"
	"time"
)

const (
	highTempThreshold = 50.0
	sleepDuration     = 10 * time.Minute
)

type Application struct {
	conf *Config
	bot  *bot.Bot
}

func NewApplication(conf *Config) (*Application, error) {
	b, err := bot.New(conf.Token, bot.WithSkipGetMe())
	if err != nil {
		return nil, err
	}
	return &Application{
		conf: conf,
		bot:  b,
	}, nil
}

func (a *Application) sendTemperatureToTelegram(ctx context.Context, render func(*SensorsTemperature) string) {
	temp, err := LoadSensorsTemperature()
	if err != nil {
		log.Printf("error getting system temp: %v", err)
		return
	}
	_, err = a.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:              a.conf.TargetId,
		Text:                render(temp),
		DisableNotification: temp.IsHigherThanThreshold(highTempThreshold),
	})
	if err != nil {
		log.Printf("error sending message: %v", err)
	}
}

func (a *Application) startMonitoring(ctx context.Context) {
	for {
		a.sendTemperatureToTelegram(ctx, RenderPlainMessage)
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
				Command:     "/temp",
				Description: "/temp - get system temperature",
			},
		},
	})
	a.bot.Start(ctx)
}
