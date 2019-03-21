package validations

import (
	"encoding/json"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/models"
	"net/http"
	"strconv"
)

func IsWalletHaveBalance(address string) bool {
	var body models.TMMsg
	resp, err := http.Get(constants.TMBalanceURL+address)
	if err != nil {
		return false
	}
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}

	userBalance, err := strconv.ParseInt(body.Value.Coins[0].Denom, 10, 64)
	if err != nil || userBalance < constants.MinBal {
		return false
	}

	return true
}
