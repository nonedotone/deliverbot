package main

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	ctx   context.Context
	api   *tgbotapi.BotAPI
	store Store
	self  tgbotapi.User
}

func NewHandler(ctx context.Context, botToken, path string) (h *Handler) {
	var err error
	h = &Handler{ctx: ctx}
	h.api, err = tgbotapi.NewBotAPIWithClient(botToken, tgbotapi.APIEndpoint, &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   60 * time.Second,
	})
	if err != nil {
		log.Fatalf("new bot api error %v", err)
	}
	h.store, err = NewLevelDB(path)
	if err != nil {
		log.Fatalf("new level db error %v", err)
	}

	h.self, err = h.api.GetMe()
	if err != nil {
		log.Fatalf("test bot api error %v", err)
	}
	return h
}
