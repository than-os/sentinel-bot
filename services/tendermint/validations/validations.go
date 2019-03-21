package validations

import (
	"encoding/json"
	"fmt"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/models"
	"math"
	"net/http"
	"strconv"
)

func CheckTMBalance(address string) (float64, bool) {
	var body models.TMMsg
	resp, err := http.Get(fmt.Sprintf(constants.TMBalanceURL, address))
	if err != nil {
		return 0, false
	}

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, false
	}

	userBalance, err := strconv.ParseFloat(body.Value.Coins[0].Amount, 64)
	if err != nil || userBalance < constants.MinBal {
		return userBalance / math.Pow(10, 8), false
	}

	return userBalance / math.Pow(10, 8), true
}