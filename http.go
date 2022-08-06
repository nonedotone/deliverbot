package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
	"net/http"
)

func (h *Handler) Serve(addr string) error {
	fmt.Printf("start serve http %s ...\n", addr)
	r := mux.NewRouter()
	r.HandleFunc("/{token}/{message}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		w.WriteHeader(http.StatusOK)
		if err := h.Message(vars["token"], vars["message"]); err != nil {
			fmt.Fprint(w, err.Error())
			return
		}
		fmt.Fprint(w, "success")
	}).Methods(http.MethodGet)
	return http.ListenAndServe(addr, r)
}

func (h *Handler) Message(token, message string) error {
	t, err := h.store.Token(token)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("invalid token")
	}
	chat := t.PrivateChat
	if len(t.Chats) == 0 {
		msg := tgbotapi.NewMessage(chat, message)
		msg.ParseMode = "markdown"
		_, err = h.api.Send(msg)
		return err
	} else {
		for _, c := range t.Chats {
			msg := tgbotapi.NewMessage(c, message)
			msg.ParseMode = "markdown"
			if _, err = h.api.Send(msg); err != nil {
				return err
			}
		}
		return nil
	}
}
