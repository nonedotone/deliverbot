package main

import (
	"bytes"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	leveldbUserKey       = []byte{0x10}
	leveldbUserTokenKey  = []byte{0x11}
	leveldbUserStatusKey = []byte{0x12}

	leveldbTokenKey        = []byte{0x20}
	leveldbTokenMessageKey = []byte{0x21}

	leveldbChatKey = []byte{0x30}

	leveldbChatTokenKey = []byte{0x40}
)

type Store interface {
	Close() error

	User(uid int64) (*User, error)
	PutUser(u *User) error

	UserStatus(uid int64) ([]byte, error)
	PutUserStatus(uid int64, s []byte) error

	Token(tid string) (*Token, error)
	PutToken(t *Token) error

	Chat(cid int64) (*Chat, error)
	PutChat(c *Chat) error

	UserToken(uid int64) ([]*Token, error)
	ChatToken(cid int64) ([]*Token, error)

	Revoke(tid string) error
}

type LevelDB struct {
	db *leveldb.DB
}

func NewLevelDB(path string) (Store, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &LevelDB{db: db}, nil
}

func (l *LevelDB) Close() error {
	return l.db.Close()
}

func (l *LevelDB) User(id int64) (*User, error) {
	return l.user(getUserKey(id))
}
func (l *LevelDB) user(key []byte) (*User, error) {
	bz, err := l.db.Get(key, nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if bz == nil || len(bz) == 0 {
		return nil, nil
	}
	return BytesToUser(bz)
}
func (l *LevelDB) PutUser(u *User) error {
	return l.db.Put(getUserKey(u.Id), u.Bytes(), nil)
}

func (l *LevelDB) UserStatus(uid int64) ([]byte, error) {
	bz, err := l.db.Get(getUserStatusKey(uid), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return bz, nil
}
func (l *LevelDB) PutUserStatus(uid int64, status []byte) error {
	return l.db.Put(getUserStatusKey(uid), status, nil)
}

func (l *LevelDB) Token(tid string) (*Token, error) {
	return l.token(getTokenKey(tid))
}
func (l *LevelDB) token(key []byte) (*Token, error) {
	bz, err := l.db.Get(key, nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if bz == nil || len(bz) == 0 {
		return nil, nil
	}
	return BytesToToken(bz)
}
func (l *LevelDB) PutToken(t *Token) error {
	userKey := getUserKey(t.User)
	user, err := l.user(userKey)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found in token")
	}
	tt, err := l.Token(t.Id)
	if err != nil {
		return err
	}
	batch := new(leveldb.Batch)
	if tt == nil {
		for _, c := range t.Chats {
			batch.Put(getChatTokenKeyWithToken(c, t.Id), []byte{})
		}
		batch.Put(getUserTokenKeyWithToken(t.User, t.Id), []byte{})
	} else {
		difa, difb := diff(tt.Chats, t.Chats)
		for _, c := range difa {
			batch.Delete(getChatTokenKeyWithToken(c, t.Id))
		}
		for _, c := range difb {
			batch.Put(getChatTokenKeyWithToken(c, t.Id), []byte{})
		}
	}
	user.Token++
	batch.Put(userKey, user.Bytes())
	batch.Delete(getUserStatusKey(t.User))
	batch.Put(getTokenKey(t.Id), t.Bytes())
	return l.db.Write(batch, nil)
}
func (l *LevelDB) tokenMessageCount(tid string) (uint64, error) {
	bz, err := l.db.Get(getTokenMessageCount(tid), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return 0, err
	}
	if bz == nil || len(bz) == 0 {
		return 0, nil
	}
	return BytesToUint64(bz), nil
}

func (l *LevelDB) Chat(cid int64) (*Chat, error) {
	bz, err := l.db.Get(getChatKey(cid), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if bz == nil || len(bz) == 0 {
		return nil, nil
	}
	return BytesToChat(bz)
}
func (l *LevelDB) PutChat(c *Chat) error {
	return l.db.Put(getChatKey(c.Id), c.Bytes(), nil)
}

func (l *LevelDB) UserToken(uid int64) ([]*Token, error) {
	key := getUserTokenKey(uid)
	iter := l.db.NewIterator(&util.Range{
		Start: key,
		Limit: PrefixEndBytes(key),
	}, nil)
	ts := make([]*Token, 0, 3)
	for iter.Next() {
		bz := bytes.TrimPrefix(iter.Key(), key)
		t, err := l.token(BytesConcat(leveldbTokenKey, bz))
		if err != nil {
			return nil, err
		}
		if t == nil {
			continue
		}
		ts = append(ts, t)
	}
	iter.Release()
	return ts, iter.Error()
}
func (l *LevelDB) ChatToken(id int64) ([]*Token, error) {
	key := getChatTokenKey(id)
	iter := l.db.NewIterator(&util.Range{
		Start: key,
		Limit: PrefixEndBytes(key),
	}, nil)
	ts := make([]*Token, 0, 10)
	for iter.Next() {
		bz := bytes.TrimPrefix(iter.Key(), key)
		t, err := l.token(BytesConcat(leveldbTokenKey, bz))
		if err != nil && err != leveldb.ErrNotFound {
			return nil, err
		}
		if t == nil {
			continue
		}
		ts = append(ts, t)
	}
	iter.Release()
	return ts, iter.Error()
}

func (l *LevelDB) Revoke(t string) error {
	tKey := getTokenKey(t)
	token, err := l.token(tKey)
	if err != nil {
		return err
	}
	if token == nil {
		return nil
	}
	uKey := getUserKey(token.User)
	user, err := l.user(uKey)
	if err != nil {
		return err
	}
	batch := new(leveldb.Batch)
	if user != nil {
		user.Token--
		batch.Put(uKey, user.Bytes())
	}
	batch.Delete(tKey)
	for _, c := range token.Chats {
		batch.Delete(getChatTokenKeyWithToken(c, t))
	}
	return l.db.Write(batch, nil)
}

func getUserKey(id int64) []byte {
	return BytesConcat(leveldbUserKey, Int64ToBytes(id))
}
func getUserStatusKey(id int64) []byte {
	return BytesConcat(leveldbUserStatusKey, Int64ToBytes(id))
}
func getTokenKey(token string) []byte {
	return BytesConcat(leveldbTokenKey, []byte(token))
}
func getTokenMessageCount(token string) []byte {
	return BytesConcat(leveldbTokenMessageKey, []byte(token))
}
func getChatTokenKey(id int64) []byte {
	return BytesConcat(leveldbChatTokenKey, Int64ToBytes(id))
}
func getUserTokenKey(id int64) []byte {
	return BytesConcat(leveldbUserTokenKey, Int64ToBytes(id))
}
func getChatTokenKeyWithToken(id int64, token string) []byte {
	return BytesConcat(leveldbChatTokenKey, Int64ToBytes(id), []byte(token))
}
func getUserTokenKeyWithToken(id int64, token string) []byte {
	return BytesConcat(leveldbUserTokenKey, Int64ToBytes(id), []byte(token))
}
func getChatKey(id int64) []byte {
	return BytesConcat(leveldbChatKey, Int64ToBytes(id))
}

func diff(a, b []int64) (difa, difb []int64) {
	fn := func(a, b []int64) []int64 {
		dif := make([]int64, 0, len(a))
		for _, v1 := range a {
			have := false
			for _, v2 := range b {
				if v1 == v2 {
					have = true
					break
				}
			}
			if !have {
				dif = append(dif, v1)
			}
		}
		return dif
	}
	return fn(a, b), fn(b, a)
}
