package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/jasonlvhit/gocron"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var src = rand.NewSource(time.Now().UnixNano())

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func GetGeoLocation(ipAddr string) (models.GeoLocation, error) {
	var gl models.GeoLocation
	resp, err := http.Get(constants.IPLEAKURL + ipAddr)
	if err != nil {
		return gl, err
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&gl); err != nil {
		return gl, err
	}

	return gl, err
}

func StrongPassword(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func AddUser(ipAddr, userName, passwordForNetwork string,
	db ldb.BotDB) error {
	var res models.UserResp

	err := DeleteUser(userName, ipAddr)
	if err != nil {
		return err
	}

	password := StrongPassword(constants.PasswordLength)
	uri := fmt.Sprintf(constants.NodeBaseUrl, ipAddr)
	err = db.Insert(passwordForNetwork, userName, password)
	if err != nil {
		return err
	}

	req := models.AddUser{Username: userName, Password: password}
	b, e := json.Marshal(req)
	if e != nil {
		return e
	}
	resp, err := http.Post(uri, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&res)

	color.Green("Add User: %s %d %s\n", res, resp.StatusCode, b)
	return err
}

func DeleteUser(username, ipAddr string) error {
	client := &http.Client{}

	uri := fmt.Sprintf(constants.NodeBaseUrl, ipAddr)
	body := models.RemoveUser{Username: username}

	b, e := json.Marshal(body)
	if e != nil {
		return e
	}
	// Create request
	req, err := http.NewRequest("DELETE", uri, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read Response Body
	b, err = ioutil.ReadAll(resp.Body)

	color.Green("Delete User: %s", b)
	return err
}

func RemoveExpiredUsers(db ldb.BotDB) {
	usersWithTimestamp, err := db.IterateExpired()
	if err != nil {
		return
	}
	today := time.Now()
	for _, user := range usersWithTimestamp {
		userExpiryTime, err := time.Parse(time.RFC3339, user.Value)
		if err != nil {
			break
		}
		if math.Signbit(userExpiryTime.Sub(today).Hours()) {
			username := strings.TrimLeft(fmt.Sprintf("%s", user.Key), "timestamp")
			ip, err := db.Read(constants.IPAddr, username[2:])
			if err != nil {
				return
			}
			err = DeleteUser(username[2:], fmt.Sprintf("%s", ip))
			if err != nil {
				return
			}
			err = db.RemoveETHUser(username[2:])
			if err != nil {
				break
			}
			err = db.RemoveTMUser(username[2:])
			if err != nil {
				break
			}
		}
	}
}

func RemoveUserJob() {
	s := gocron.NewScheduler()
	s.Every(3).Hours().Do(RemoveExpiredUsers)
	<-s.Start()
}
