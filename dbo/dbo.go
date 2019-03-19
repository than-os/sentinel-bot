package dbo

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/than-os/globex-bot/utils"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/handlers"
)

type Level struct {
	db *leveldb.DB
	//nodes []models.TONNode
}

func NewDB() (ldb.BotDB, *models.Nodes, error) {

	nodes, err := handlers.GetNodes()
	if err != nil {
		return nil, nil, err
	}

	db, err := leveldb.OpenFile("./store", nil)
	return Level{db: db}, &nodes, err
}

func (l Level) Insert(key, username, value string) error {
	k := []byte(key + username)
	v := []byte(value)
	return l.db.Put(k, v, nil)
}

func (l Level) Update(key, username, value string) error {

	return nil
}

func (l Level) Delete(key, username string) error {

	return nil
}

func (l Level) Read(key, username string) (models.KV, error) {
	k := []byte(key + username)
	v, e := l.db.Get(k, nil)
	if e != nil {
		return models.KV{}, e
	}

	return models.KV{
		Key:   fmt.Sprintf("%s", k),
		Value: fmt.Sprintf("%s", v),
	}, e
}

func (l Level) MultiWriter(pairs []models.KV, username string) error {
	for _, pair := range pairs {
		err := l.Insert(pair.Key, username, pair.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l Level) MultiReader(keys []string, username string) ([]models.KV, error) {
	var result []models.KV
	for _, key := range keys {
		kv, err := l.Read(key, username)
		if err != nil {
			return result, err
		}
		result = append(result, kv)
	}
	return result, nil
}

func (l Level) IterateExpired() ([]models.ExpiredUsers, error) {
	itr := l.db.NewIterator(util.BytesPrefix([]byte(constants.Timestamp)), nil)

	var usersWithTimestamp []models.ExpiredUsers
	for itr.Next() {
		usersWithTimestamp = append(usersWithTimestamp, models.ExpiredUsers{
			Key: fmt.Sprintf("%s", itr.Key()), Value: fmt.Sprintf("%s", itr.Value()),
		})
	}
	itr.Release()
	err := itr.Error()
	if err != nil {
		return usersWithTimestamp, err
	}

	return usersWithTimestamp, err
}

func (l Level) Iterate() []models.User {
	itr := l.db.NewIterator(nil, nil)

	var p []models.User
	var w []models.KV
	for itr.Next() {
		w = append(w, models.KV{Key: fmt.Sprintf("%s", itr.Key()), Value: fmt.Sprintf("%s", itr.Value())})

	}
	defer itr.Release()
	err := itr.Error()

	if err != nil {
		return []models.User{}
	}

	for _, user := range w {
		username := utils.GetTelegramUsername(user.Key)
		var participant models.User
		if username != "" {
			for _, u := range w {
				if u.Key == constants.EthAddr+username {
					participant.EthAddr = u.Value
					participant.TelegramUsername = username
				} else if u.Key == constants.Timestamp+username {
					t, err := time.Parse(time.RFC3339, fmt.Sprintf("%s", u.Value))
					if err != nil {
						color.Red("%s", "error while adding user timestamp")
						return []models.User{}
					}
					participant.Timestamp = t
				} else if u.Key == constants.Node+username {

				}
			}
			color.Cyan("user: %v", user)
		}
		if participant.EthAddr != "" && participant.TelegramUsername != "" {
			p = append(p, participant)
		}
	}

	return p
}

func (l Level) RemoveETHUser(username string) error {
	if e := l.db.Delete([]byte(constants.Timestamp+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.IsAuth+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.Node+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.Password+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.Bandwidth+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.AssignedNodeURI+username), nil); e != nil {
		return e
	}
	return nil
}

func (l Level) RemoveTMUser(username string) error {
	if e := l.db.Delete([]byte(constants.Timestamp+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.IsAuthTM+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.NodeTM+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.PasswordTM+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.BandwidthTM+username), nil); e != nil {
		return e
	}
	if e := l.db.Delete([]byte(constants.AssignedNodeURITM+username), nil); e != nil {
		return e
	}
	return nil
}
