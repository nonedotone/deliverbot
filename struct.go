package main

import (
	"encoding/json"
	"fmt"
)

type User struct {
	Id      int64 `json:"id"`
	Private int64 `json:"private"`
	Token   int   `json:"token"`
}

func BytesToUser(bz []byte) (*User, error) {
	u := &User{}
	err := json.Unmarshal(bz, u)
	if err != nil {
		return nil, fmt.Errorf("unmarshal user error: %v", err)
	}
	return u, nil
}
func (u *User) Bytes() []byte {
	bz, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return bz
}

type Token struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	User        int64   `json:"user"`
	Chats       []int64 `json:"chat"`
	PrivateChat int64   `json:"private_chat"`
}

func BytesToToken(bz []byte) (*Token, error) {
	t := &Token{}
	err := json.Unmarshal(bz, t)
	if err != nil {
		return nil, fmt.Errorf("unmarshal user error: %v", err)
	}
	return t, nil
}
func (t *Token) Bytes() []byte {
	bz, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return bz
}
func (t *Token) HasChat(cid int64) bool {
	for _, c := range t.Chats {
		if c == cid {
			return true
		}
	}
	return false
}
func (t *Token) BindingChat(cid int64) {
	t.Chats = append(t.Chats, cid)
}
func (t *Token) UnbindingChat(cid int64) {
	for i, c := range t.Chats {
		if c == cid {
			length := len(t.Chats)
			t.Chats[i], t.Chats[length-1] = t.Chats[length-1], t.Chats[i]
			t.Chats = t.Chats[:length-1]
		}
	}
}

type Chat struct {
	Id        int64   `json:"id"`
	Title     string  `json:"title"`
	Admins    []int64 `json:"admins"`
	Timestamp int64   `json:"timestamp"`
}

func BytesToChat(bz []byte) (*Chat, error) {
	c := &Chat{}
	err := json.Unmarshal(bz, c)
	if err != nil {
		return nil, fmt.Errorf("unmarshal user error: %v", err)
	}
	return c, nil
}
func (c *Chat) Bytes() []byte {
	bz, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return bz
}
func (c *Chat) IsAdmin(uid int64) bool {
	for _, a := range c.Admins {
		if a == uid {
			return true
		}
	}
	return false
}
