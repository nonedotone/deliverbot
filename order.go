package main

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	listKey               = "l-"
	tokenKey              = "t-"
	revokeKey             = "r-"
	settingKey            = "s-"
	bindingKey            = "b-"
	unbindingKey          = "u-"
	listBindingKey        = "lb-"
	listUnbindingTokenKey = "lut-"
	listBindingChatKey    = "lbc-"

	unbindingTokenSubKey = "t-"
	unbindingChatSubKey  = "c-"

	chatUpdateTicker     = 10 //10 * 60
	InlineKeyboardNumber = 3
	maxTokenCount        = 3
	tokenLength          = 30
	messageMaxLength     = 400
)

var (
	newToken = []byte("new_token")
)

const (
	InternalError    = "internal error"
	QuantityExceeded = "quantity exceeded"
)

func (h *Handler) botUpdate() {
	fmt.Println("start receive message...")
	updates := h.api.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Limit: 0, Timeout: 30})
	for {
		select {
		case u := <-updates:
			if u.Message != nil && checkAdmin(u.Message.From.ID) {
				go h.message(u.Message)
			}
			if u.CallbackQuery != nil && checkAdmin(u.CallbackQuery.From.ID) {
				go h.callback(u.CallbackQuery)
			}
		case <-h.ctx.Done():
			return
		}
	}
}

func checkAdmin(userId int64) bool {
	if admin == 0 {
		return true
	}
	if userId == admin {
		return true
	}
	return false
}

func (h *Handler) message(msg *tgbotapi.Message) {
	var err error
	var cmd = msg.Command()
	switch cmd {
	case "start":
		err = h.start(msg)
	case "new":
		err = h.new(msg)
	case "setting":
		err = h.setting(msg)
	default:
		err = h.others(msg)
	}
	if err != nil {
		log.Printf("handler message ---> command %s, error %v, message %s\n", cmd, err, msg.Text)
	}
}

func (h *Handler) callback(cbq *tgbotapi.CallbackQuery) {
	var err error
	//t-123456 -> t-
	prefix := strings.Split(cbq.Data, "-")[0] + "-"
	switch prefix {
	case listKey:
		err = h.listCallback(cbq)
	case tokenKey:
		err = h.tokenCallback(cbq)
	case revokeKey:
		err = h.revokeCallback(cbq)
	case unbindingKey:
		err = h.unbindingCallback(cbq)
	case settingKey:
		err = h.settingCallback(cbq)
	case bindingKey:
		err = h.bindingCallback(cbq)
	case listBindingChatKey:
		err = h.listBindingChatCallback(cbq)
	case listUnbindingTokenKey:
		err = h.listUnbindingTokenCallback(cbq)
	default:
		h.delete(cbq.Message)
		err = h.reply(cbq.Message.Chat.ID, "invalid request")
	}
	if err != nil {
		log.Printf("handler callback ---> key [%s], error %v, data [%s]\n", prefix, err, cbq.Data)
	}
}

func callbackSettingChatKey(chatId int64) string {
	return settingKey + strconv.FormatInt(chatId, 10)
}

func (h *Handler) start(msg *tgbotapi.Message) error {
	//private chat
	if !msg.Chat.IsPrivate() {
		return h.usage(msg.Chat.ID, msg.MessageID)
	}
	//record user
	if _, err := h.record(msg.Chat.ID, msg.From.ID); err != nil {
		return h.reply(msg.Chat.ID, InternalError)
	}
	//data
	text := strings.TrimSpace(msg.CommandArguments())
	//chat setting
	if strings.HasPrefix(text, settingKey) {
		return h.settingChat(msg, msg.From, strings.TrimPrefix(text, settingKey), false)
	}
	//list
	return h.listToken(msg, msg.From, false)
}
func (h *Handler) new(msg *tgbotapi.Message) error {
	if !msg.Chat.IsPrivate() { // only private chat
		return nil
	}
	chat, from := msg.Chat, msg.From
	user, err := h.record(chat.ID, from.ID)
	if err != nil {
		//TODO log
		return h.reply(chat.ID, InternalError)
	}
	if user.Token >= maxTokenCount {
		return h.reply(chat.ID, QuantityExceeded)
	}
	if err := h.store.PutUserStatus(user.Id, newToken); err != nil {
		//TODO log
		return h.reply(chat.ID, InternalError)
	}
	return h.reply(chat.ID, "enter a name (length 1-30)")
}
func (h *Handler) setting(msg *tgbotapi.Message) error {
	if msg.Chat.IsPrivate() {
		h.delete(msg)
		return nil
	}
	operation := callbackSettingChatKey(msg.Chat.ID)
	settingUrl := fmt.Sprintf("https://t.me/%s?start=%s", h.self.UserName, operation)
	replyMarkup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("setting", settingUrl),
		),
	)
	reply := tgbotapi.NewMessage(msg.Chat.ID, "chat with my deliver")
	reply.ReplyMarkup = replyMarkup
	_, err := h.api.Send(reply)
	return err
}
func (h *Handler) others(msg *tgbotapi.Message) error {
	if !msg.Chat.IsPrivate() {
		return nil
	}
	chat, from := msg.Chat, msg.From
	if len(msg.Command()) > 0 {
		return h.reply(chat.ID, "invalid command")
	}
	//status
	userStatus, err := h.store.UserStatus(from.ID)
	if err != nil {
		return h.reply(chat.ID, InternalError)
	}
	if bytes.EqualFold(userStatus, newToken) {
		return h.newToken(chat.ID, from, msg)
	}
	return nil
}

func (h *Handler) listCallback(cbq *tgbotapi.CallbackQuery) error {
	return h.listToken(cbq.Message, cbq.From, true)
}
func (h *Handler) tokenCallback(cbq *tgbotapi.CallbackQuery) error {
	token := strings.TrimPrefix(cbq.Data, tokenKey)
	return h.token(cbq.Message, cbq.From, token)
}
func (h *Handler) unbindingCallback(cbq *tgbotapi.CallbackQuery) error {
	//key+token+chatId
	data := strings.TrimPrefix(cbq.Data, unbindingKey)
	unbindingType := 0
	if strings.HasPrefix(data, unbindingTokenSubKey) {
		data = strings.TrimPrefix(data, unbindingTokenSubKey)
		unbindingType = 1 //chat
	}
	if strings.HasPrefix(data, unbindingChatSubKey) {
		data = strings.TrimPrefix(data, unbindingChatSubKey)
		unbindingType = 2 //token
	}
	if len(data) <= tokenLength {
		//TODO log keyboard
		h.delete(cbq.Message)
		// TODO h.listToken(cbq.Message,cbq.From,true)
		return h.reply(cbq.Message.Chat.ID, "invalid request")
	}
	token := data[:tokenLength]
	chatId, err := strconv.ParseInt(data[tokenLength:], 10, 64)
	if err != nil {
		//TODO log
		h.delete(cbq.Message)
		return h.reply(cbq.Message.Chat.ID, "invalid request")
	}
	return h.unbindingChat(cbq.Message, cbq.From, token, chatId, unbindingType)
}
func (h *Handler) revokeCallback(cbq *tgbotapi.CallbackQuery) error {
	token := strings.TrimPrefix(cbq.Data, revokeKey)
	return h.revoke(cbq.Message, cbq.From, token)
}
func (h *Handler) bindingCallback(cbq *tgbotapi.CallbackQuery) error {
	//key+token+chatId
	data := strings.TrimPrefix(cbq.Data, bindingKey)
	if len(data) <= tokenLength {
		//TODO log keyboard
		h.delete(cbq.Message)
		return h.reply(cbq.Message.Chat.ID, "invalid request")
	}
	token := data[:tokenLength]
	chatId, err := strconv.ParseInt(data[tokenLength:], 10, 64)
	if err != nil {
		//TODO log
		h.delete(cbq.Message)
		return h.reply(cbq.Message.Chat.ID, "invalid request")
	}
	return h.bindingChat(cbq.Message, cbq.From, token, chatId)
}
func (h *Handler) settingCallback(cbq *tgbotapi.CallbackQuery) error {
	return h.settingChat(cbq.Message, cbq.From, strings.TrimPrefix(cbq.Data, settingKey), true)
}
func (h *Handler) listBindingChatCallback(cbq *tgbotapi.CallbackQuery) error {
	token := strings.TrimPrefix(cbq.Data, listBindingChatKey)
	return h.listTokenChat(cbq.Message, cbq.From, token)
}

func (h *Handler) listBindingCallback(cbq *tgbotapi.CallbackQuery) error {
	c, u := cbq.Message.Chat, cbq.From
	//get chat
	chat, err := h.chat(StringToInt64(strings.TrimPrefix(cbq.Data, listBindingKey)))
	if err != nil {
		//TODO log
		h.delete(cbq.Message)
		return h.reply(c.ID, InternalError)
	}
	//check admin
	if !chat.IsAdmin(u.ID) {
		h.delete(cbq.Message)
		return h.reply(c.ID, "no permission")
	}
	//get tokens
	tokens, err := h.store.UserToken(u.ID)
	if err != nil {
		//TODO log
		h.delete(cbq.Message)
		return h.reply(c.ID, InternalError)
	}
	if len(tokens) == 0 {
		h.delete(cbq.Message)
		return h.reply(c.ID, "no tokens")
	}
	//list token
	filter := func(t *Token) bool {
		return !t.HasChat(chat.Id)
	}
	operation := func(t *Token) string {
		return fmt.Sprintf("%s%s%d", bindingKey, t.Id, chat.Id)
	}
	keyboardRows := listTokenInlineKeyboard(tokens, filter, operation)
	if len(keyboardRows) == 0 {
		h.delete(cbq.Message)
		return h.reply(c.ID, "all token binding, create with /new")
	}
	msg := tgbotapi.NewEditMessageText(c.ID, cbq.Message.MessageID, "select one to binding")
	msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
	_, err = h.api.Send(msg)
	return err
}
func (h *Handler) listUnbindingTokenCallback(cbq *tgbotapi.CallbackQuery) error {
	c, u := cbq.Message.Chat, cbq.From
	//get chat
	chat, err := h.chat(StringToInt64(strings.TrimPrefix(cbq.Data, listUnbindingTokenKey)))
	if err != nil {
		//TODO log
		h.delete(cbq.Message)
		return h.reply(c.ID, InternalError)
	}
	//check admin
	if !chat.IsAdmin(u.ID) {
		h.delete(cbq.Message)
		return h.reply(c.ID, "no permission")
	}
	//get tokens
	tokens, err := h.store.UserToken(u.ID)
	if err != nil {
		//TODO log
		h.delete(cbq.Message)
		return h.reply(c.ID, InternalError)
	}
	if len(tokens) == 0 {
		h.delete(cbq.Message)
		return h.reply(c.ID, "no token, create with /new")
	}
	//list token
	filter := func(t *Token) bool {
		return !t.HasChat(chat.Id)
	}
	operation := func(t *Token) string {
		return fmt.Sprintf("%s%s%d", bindingKey, t.Id, chat.Id)
	}
	keyboardRows := listTokenInlineKeyboard(tokens, filter, operation)
	if len(keyboardRows) == 0 {
		return h.reply(c.ID, "all token binding, create with /new")
	}
	msg := tgbotapi.NewEditMessageText(c.ID, cbq.Message.MessageID, "select one to binding")
	msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
	_, err = h.api.Send(msg)
	return err
}

func callbackTokenKey(t string) string {
	return tokenKey + t
}
func (h *Handler) listToken(msg *tgbotapi.Message, user *tgbotapi.User, callback bool) error {
	//get tokens
	tokens, err := h.store.UserToken(user.ID)
	if err != nil {
		//TODO log
		if callback {
			h.delete(msg)
		}
		return h.reply(msg.Chat.ID, InternalError)
	}
	if len(tokens) == 0 {
		if callback {
			h.delete(msg)
		}
		return h.reply(msg.Chat.ID, "no tokens, create with /new")
	}
	//list token
	filter := func(t *Token) bool {
		return true
	}
	operation := func(t *Token) string {
		return callbackTokenKey(t.Id)
	}
	keyboardRows := listTokenInlineKeyboard(tokens, filter, operation)
	var ct tgbotapi.Chattable
	if !callback {
		msgN := tgbotapi.NewMessage(msg.Chat.ID, "choose from the following")
		msgN.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
		ct = msgN
	} else {
		msgE := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, "choose from the following")
		msgE.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
		ct = msgE
	}
	_, err = h.api.Send(ct)
	return err
}
func (h *Handler) token(msg *tgbotapi.Message, user *tgbotapi.User, token string) error {
	t, err := h.store.Token(token)
	if err != nil {
		//TODO log
		return h.reply(msg.Chat.ID, InternalError)
	}
	if t == nil {
		//TODO log
		return h.reply(msg.Chat.ID, "token not exist")
	}
	if t.User != user.ID {
		return h.reply(msg.Chat.ID, "no permission")
	}
	buf := bytes.Buffer{}
	//TODO i18n
	buf.WriteString(fmt.Sprintf("token:   %s\n", t.Name))
	buf.WriteString(fmt.Sprintf("chat:    %d\n", len(t.Chats)))
	buf.WriteString(fmt.Sprintf("example: https://example.com/%s/hello)", t.Id))

	revokeOperation := revokeKey + t.Id
	listBindingOperation := listBindingChatKey + t.Id
	returnOperation := listKey
	buttons := make([]tgbotapi.InlineKeyboardButton, 0, 3)
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("revoke", revokeOperation))
	if len(t.Chats) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("unbinding", listBindingOperation))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("back", returnOperation))
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttons...))
	msgN := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, buf.String())
	msgN.ReplyMarkup = &numericKeyboard
	msgN.ParseMode = "markdown"
	_, err = h.api.Send(msgN)
	return err
}
func (h *Handler) revoke(msg *tgbotapi.Message, user *tgbotapi.User, token string) error {
	t, err := h.store.Token(token)
	if err != nil {
		//TODO log
		h.delete(msg)
		return h.reply(msg.Chat.ID, InternalError)
	}
	if t == nil {
		//TODO log
		//TODO h.listToken(msg, user, true)
		return h.reply(msg.Chat.ID, "token not exist")
	}
	if t.User != user.ID {
		h.delete(msg)
		return h.reply(msg.Chat.ID, "no permission")
	}
	if err := h.store.Revoke(token); err != nil {
		//TODO log
		h.delete(msg)
		return h.reply(msg.Chat.ID, InternalError)
	}

	text := "revoke token success."
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("back", listKey)),
	)
	msgN := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, text)
	msgN.ReplyMarkup = &numericKeyboard
	_, err = h.api.Send(msgN)
	return err
}

func (h *Handler) settingChat(msg *tgbotapi.Message, user *tgbotapi.User, data string, callback bool) error {
	//get chat
	chat, err := h.chat(StringToInt64(data))
	if err != nil {
		//TODO log
		if callback {
			h.delete(msg)
		}
		return h.reply(msg.Chat.ID, InternalError)
	}
	//check admin
	if !chat.IsAdmin(user.ID) {
		if callback {
			h.delete(msg)
		}
		return h.reply(msg.Chat.ID, "no permission")
	}
	//get tokens
	chatTokens, err := h.store.ChatToken(chat.Id)
	if err != nil {
		//TODO log
		if callback {
			h.delete(msg)
		}
		return h.reply(msg.Chat.ID, InternalError)
	}
	if len(chatTokens) == 0 {
		userToken, err := h.store.UserToken(user.ID)
		if err != nil {
			//TODO log
			if callback {
				h.delete(msg)
			}
			return h.reply(msg.Chat.ID, InternalError)
		}
		if len(userToken) == 0 {
			if callback {
				h.delete(msg)
			}
			return h.reply(msg.Chat.ID, "no token, create with /new")
		}

		filter := func(t *Token) bool {
			return !t.HasChat(msg.Chat.ID)
		}
		operation := func(t *Token) string {
			return fmt.Sprintf("%s%s%d", bindingKey, t.Id, chat.Id)
		}
		keyboardRows := listTokenInlineKeyboard(userToken, filter, operation)

		if callback {
			msgN := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, "no token binding, choose to binding")
			inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
			msgN.ReplyMarkup = &inlineKeyboard
			_, err = h.api.Send(msgN)
			return err
		} else {
			msgN := tgbotapi.NewMessage(msg.Chat.ID, "no token binding, choose to binding")
			msgN.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
			_, err = h.api.Send(msgN)
			return err
		}
	}
	filter := func(t *Token) bool {
		return true
	}
	operation := func(t *Token) string {
		return fmt.Sprintf("%s%s%s%d", unbindingKey, unbindingTokenSubKey, t.Id, chat.Id)
	}
	keyboardRows := listTokenInlineKeyboard(chatTokens, filter, operation)
	//TODO
	bindingButton := tgbotapi.NewInlineKeyboardButtonData("binding more", fmt.Sprintf("%s%d", listUnbindingTokenKey, chat.Id))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(bindingButton))
	if callback {
		msgN := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, "choose to unbinding or binding more")
		msgN.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
		_, err = h.api.Send(msgN)
		return err
	} else {
		msgN := tgbotapi.NewMessage(msg.Chat.ID, "choose to unbinding or binding more")
		msgN.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
		_, err = h.api.Send(msgN)
		return err
	}
}
func (h *Handler) bindingChat(msg *tgbotapi.Message, user *tgbotapi.User, token string, chatId int64) error {
	t, err := h.store.Token(token)
	if err != nil {
		//TODO log
		return h.reply(msg.Chat.ID, InternalError)
	}
	if t == nil {
		//TODO log
		return h.reply(msg.Chat.ID, "token not exist")
	}
	if t.User != user.ID {
		return h.reply(msg.Chat.ID, "no permission")
	}
	if t.HasChat(chatId) {
		//TODO keyboard ...
		return h.reply(msg.Chat.ID, "already binding")
	}
	chat, err := h.chat(chatId)
	if err != nil {
		//TODO log
		return h.reply(msg.Chat.ID, InternalError)
	}
	t.BindingChat(chat.Id)
	if err := h.store.PutToken(t); err != nil {
		return h.reply(msg.Chat.ID, InternalError)
	}
	buf := bytes.Buffer{}
	//TODO i18n
	buf.WriteString("binding token success.")
	returnOperation := fmt.Sprintf("%s%d", settingKey, chatId)
	numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			//TODO i18n
			tgbotapi.NewInlineKeyboardButtonData("back", returnOperation),
		),
	)
	msgN := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, buf.String())
	msgN.ReplyMarkup = &numericKeyboard
	_, err = h.api.Send(msgN)
	return nil
}
func (h *Handler) unbindingChat(msg *tgbotapi.Message, user *tgbotapi.User, token string, chatId int64, unbindingType int) error {
	t, err := h.store.Token(token)
	if err != nil {
		//TODO log
		return h.reply(msg.Chat.ID, InternalError)
	}
	if t == nil {
		//TODO log
		return h.reply(msg.Chat.ID, "token not exist")
	}
	if !t.HasChat(chatId) {
		//TODO keyboard ...
		return h.reply(msg.Chat.ID, "not binding chat")
	}
	chat, err := h.chat(chatId)
	if err != nil {
		//TODO log
		return h.reply(msg.Chat.ID, InternalError)
	}
	if t.User != user.ID && chat.IsAdmin(user.ID) {
		//TODO log
		return h.reply(msg.Chat.ID, "no permission")
	}
	t.UnbindingChat(chat.Id)

	if err := h.store.PutToken(t); err != nil {
		//TODO log keyboard ...
		return h.reply(msg.Chat.ID, InternalError)
	}
	buf := bytes.Buffer{}
	//i18n
	buf.WriteString("unbinding token success.")
	var returnOperation string
	if unbindingType == 1 { //chat unbinding
		returnOperation = fmt.Sprintf("%s%d", settingKey, chatId)
	}
	if unbindingType == 2 { //token unbinding
		returnOperation = fmt.Sprintf("%s%s", tokenKey, t.Id)
	}
	if len(returnOperation) > 0 {
		numericKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("back", returnOperation),
			),
		)
		msgN := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, buf.String())
		msgN.ReplyMarkup = &numericKeyboard
		_, err = h.api.Send(msgN)
		return err
	}
	return nil
}
func (h *Handler) listTokenChat(msg *tgbotapi.Message, user *tgbotapi.User, token string) error {
	t, err := h.store.Token(token)
	if err != nil {
		//TODO log
		return h.reply(msg.Chat.ID, InternalError)
	}
	if t == nil {
		//TODO log
		h.delete(msg)
		return h.reply(msg.Chat.ID, "not exist")
	}
	if len(t.Chats) == 0 {
		//TODO keyboard
		return h.reply(msg.Chat.ID, "not binding")
	}
	chats := make([]*Chat, 0, len(t.Chats))
	for _, c := range t.Chats {
		chat, err := h.chat(c)
		if err != nil {
			return h.reply(msg.Chat.ID, InternalError)
		}
		chats = append(chats, chat)
	}

	//list token
	filter := func(c *Chat) bool {
		return true
	}
	operation := func(c *Chat) string {
		return fmt.Sprintf("%s%s%s%d", unbindingKey, unbindingChatSubKey, t.Id, c.Id)
	}
	keyboardRows := listChatInlineKeyboard(chats, filter, operation)

	backButton := tgbotapi.NewInlineKeyboardButtonData("back", tokenKey+t.Id)
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(backButton))

	//TODO i18n
	msgEdit := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, "select chat to unbinding")
	msgEdit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
	_, err = h.api.Send(msgEdit)
	return err
}

func (h *Handler) chat(chatId int64) (*Chat, error) {
	if chatId == 0 {
		return nil, fmt.Errorf("invalid chat id")
	}
	chat, err := h.store.Chat(chatId)
	if err != nil {
		return nil, fmt.Errorf("db query chat error %v", err)
	}
	if chat != nil && chat.Timestamp+chatUpdateTicker > time.Now().Unix() {
		return chat, nil
	}
	if chat == nil {
		chat = &Chat{Id: chatId, Timestamp: time.Now().Unix()}
	}
	tgChat, err := h.api.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatId}})
	if err != nil {
		return nil, fmt.Errorf("api query chat error %v", err)
	}
	chat.Title = tgChat.Title
	members, err := h.api.GetChatAdministrators(tgbotapi.ChatAdministratorsConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatId}})
	if err != nil {
		return nil, fmt.Errorf("api query admins error %v", err)
	}
	chat.Admins = make([]int64, 0, len(members))
	for _, m := range members {
		chat.Admins = append(chat.Admins, m.User.ID)
	}
	return chat, h.store.PutChat(chat)
}
func (h *Handler) newToken(chatId int64, user *tgbotapi.User, msg *tgbotapi.Message) error {
	text := msg.Text
	//check token name
	text = strings.TrimSpace(text)
	if len(text) == 0 || len(text) > 33 {
		return h.reply(chatId, "invalid name, input again")
	}
	token, err := h.randomToken()
	if err != nil {
		//TODO log
		return h.reply(chatId, InternalError)
	}
	if err := h.store.PutToken(&Token{
		Id:          token,
		Name:        text,
		User:        user.ID,
		Chats:       make([]int64, 0),
		PrivateChat: chatId,
	}); err != nil {
		//TODO log
		return h.reply(chatId, InternalError)
	}
	return h.reply(chatId, "create token success, view with /start.")
}
func (h *Handler) randomToken() (string, error) {
	token := RandomString(tokenLength)
	for {
		bz, err := h.store.Token(token)
		if err != nil {
			return "", err
		}
		if bz == nil {
			break
		}
		token = RandomString(tokenLength)
	}
	return token, nil
}

func (h *Handler) delete(msg *tgbotapi.Message) {
	del := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
	h.api.Request(del)
}
func (h *Handler) reply(chatId int64, text string) error {
	msg := tgbotapi.NewMessage(chatId, text)
	_, err := h.api.Send(msg)
	return err
}
func (h *Handler) usage(chatId int64, msgId int) error {
	msg := tgbotapi.NewMessage(chatId, "usage:---->")
	if msgId > 0 {
		msg.ReplyToMessageID = msgId
	}
	_, err := h.api.Send(msg)
	return err
}
func (h *Handler) record(chatId int64, userId int64) (*User, error) {
	user, err := h.store.User(userId)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}
	user = &User{Id: userId, Private: chatId, Token: 0}
	return user, h.store.PutUser(user)
}

func listTokenInlineKeyboard(ts []*Token, ft func(t *Token) bool, op func(t *Token) string) [][]tgbotapi.InlineKeyboardButton {
	keyboardButtons := make([]tgbotapi.InlineKeyboardButton, 0, InlineKeyboardNumber)
	keyboardRows := make([][]tgbotapi.InlineKeyboardButton, 0, len(ts)/InlineKeyboardNumber+1)
	//filter token
	for _, t := range ts {
		if !ft(t) {
			continue
		}
		operation := op(t)
		button := tgbotapi.NewInlineKeyboardButtonData(t.Name, operation)
		keyboardButtons = append(keyboardButtons, button)
		if len(keyboardButtons) == InlineKeyboardNumber {
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(keyboardButtons...))
			keyboardButtons = make([]tgbotapi.InlineKeyboardButton, 0, InlineKeyboardNumber)
		}
	}
	if len(keyboardButtons) > 0 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(keyboardButtons...))
	}
	return keyboardRows
}
func listChatInlineKeyboard(cs []*Chat, ft func(c *Chat) bool, op func(c *Chat) string) [][]tgbotapi.InlineKeyboardButton {
	keyboardButtons := make([]tgbotapi.InlineKeyboardButton, 0, InlineKeyboardNumber)
	keyboardRows := make([][]tgbotapi.InlineKeyboardButton, 0, len(cs)/InlineKeyboardNumber+1)
	//filter token
	for _, c := range cs {
		if !ft(c) {
			continue
		}
		operation := op(c)
		button := tgbotapi.NewInlineKeyboardButtonData(c.Title, operation)
		keyboardButtons = append(keyboardButtons, button)
		if len(keyboardButtons) == InlineKeyboardNumber {
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(keyboardButtons...))
			keyboardButtons = make([]tgbotapi.InlineKeyboardButton, 0, InlineKeyboardNumber)
		}
	}
	if len(keyboardButtons) > 0 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(keyboardButtons...))
	}
	return keyboardRows
}
