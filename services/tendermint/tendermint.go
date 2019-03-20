package tendermint

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/helpers"
	"github.com/than-os/sentinel-bot/services/proxy"
	"github.com/than-os/sentinel-bot/templates"
	"gopkg.in/telegram-bot-api.v4"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func AskForTendermintWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	if len(nodes) == 0 {
		btnOpts := []string{constants.EthNetwork}
		opts := models.ButtonHelper{Type: constants.ReplyButton, Labels: btnOpts}
		helpers.Send(b, u, templates.NoTMNodes, opts)
		return
	}

	err := db.Insert(constants.BlockchainNetwork, u.Message.From.UserName, constants.TenderMintNetwork)
	if err != nil {
		helpers.Send(b, u, "internal bot error")
		return
	}

	helpers.Send(b, u, templates.AskForTMWallet)
	helpers.SetState(b, u, constants.TMState, constants.TMState0, db)
}

func IsValidTMAccount(u tgbotapi.Update) string {
	ok := strings.HasPrefix(u.Message.Text, constants.TMPrefix)
	l := len(u.Message.Text)

	if ok && l == constants.TMWalletLength {
		return u.Message.Text
	}

	return ""
}

func IsTMTxnHash(u tgbotapi.Update) string {
	ok := common.IsHexAddress(u.Message.Text)
	// sample tx hash = 158AAFD03A6493B922216A7F5AAC8FA0865F7643
	if ok && len(u.Message.Text) == constants.TMHashLength {
		return u.Message.Text
	}

	return ""
}

func getTMTxn(hash string) models.TMTxn {
	var txnResp models.TMTxn
	url := fmt.Sprintf(constants.TMTxnURL, hash)
	resp, err := http.Get(url)
	if err != nil {
		return txnResp
	}
	if err = json.NewDecoder(resp.Body).Decode(&txnResp); err != nil {
		return txnResp
	}
	return txnResp
}

func HandleTMTxnHash(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	state := helpers.GetState(b, u, constants.TMState, db)
	color.Green("******* STATE BW = %d *******", state)

	if state <= constants.TMState2 {
		helpers.Send(b,u, templates.FollowSequence)
		return
	}

	resp, err := db.Read(constants.NodeTM, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not get user info")
		_, _ = b.Send(c)
		return
	}

	respToStr := fmt.Sprintf("%s", resp.Value)
	strToInt, err := strconv.Atoi(respToStr)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}

	i := strToInt - 1
	if IsValidTMTxn(u, db) {
		url := fmt.Sprintf(constants.ProxyURL, nodes[i].IPAddr, strconv.Itoa(nodes[i].Port), nodes[i].Username, nodes[i].Password)

		values := []models.KV{
			{Key: constants.IPAddrTM, Value: nodes[i].IPAddr},
			{Key: constants.AssignedNodeURITM, Value: url},
			{Key: constants.IsAuthTM, Value: "true"},
		}
		err := db.MultiWriter(values, u.Message.From.UserName)
		if err != nil {
			helpers.Send(b, u, templates.Error)
			return
		}

		helpers.Send(b, u, "Thanks for submitting the TX-HASH. We're validating it")
		helpers.Send(b, u, "creating new user for "+u.Message.From.UserName+"...")

		node := nodes[i]
		err = proxy.AddUser(node.IPAddr, u.Message.From.UserName, constants.PasswordTM, db)
		if err != nil {
			helpers.Send(b, u, templates.Error)
			return
		}
		pass, err := db.Read(constants.PasswordTM, u.Message.From.UserName)
		if err != nil {
			helpers.Send(b, u, templates.Error)
			return
		}
		url = fmt.Sprintf(constants.ProxyURL, nodes[i].IPAddr, strconv.Itoa(nodes[i].Port), u.Message.From.UserName, pass.Value)

		kv := []models.KV{
			{
				Key:   constants.IPAddrTM,
				Value: nodes[i].IPAddr,
			},
			{
				Key:   constants.AssignedNodeURITM,
				Value: url,
			},
		}

		err = db.MultiWriter(kv, u.Message.From.UserName)
		if err != nil {
			helpers.Send(b, u, templates.Error)
			return
		}

		btnOpts := []models.InlineButtonOptions{
			{Label: nodes[i].Username, URL: url},
		}
		opts := models.ButtonHelper{
			Type:               constants.InlineButton,
			InlineKeyboardOpts: btnOpts,
		}
		helpers.Send(b, u, templates.Success, opts)
		go helpers.SetState(b, u, constants.TMState, constants.TMState4, db)
		return
	}

	helpers.Send(b, u, "invalid txn hash. please try again")
}

func IsValidTMTxn(u tgbotapi.Update, db ldb.BotDB) bool {

	username := u.Message.From.UserName
	hash := u.Message.Text
	txn := getTMTxn(hash)

	userWallet, err := db.Read(constants.WalletTM, username)
	color.Green("what is it3: %s", "")

	if err != nil {
		return false
	}

	recipientWallet, err := db.Read(constants.NodeWalletTM, username)
	color.Green("what is it3: %s", "")

	if err != nil {
		return false
	}

	amount, err := db.Read(constants.NodePriceTM, username)
	color.Green("what is it2: %s", "")
	if err != nil {
		return false
	}

	if len(txn.Tx.Value.Msg) > 0 {
		okWallet := txn.Tx.Value.Msg[0].Value.From == userWallet.Value
		okRecipient := txn.Tx.Value.Msg[0].Value.To == recipientWallet.Value
		okAmount := parseTxnAmount(txn.Tx.Value.Msg[0].Value.Coins[0].Amount) == amount.Value

		color.Green("what is it1 ? %s %s %s ", okAmount, okWallet, okRecipient)

		if okWallet && okRecipient && okAmount {
			return true
		}
	}
	return false
}

func HandleWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	TMState := helpers.GetState(b, u, constants.TMState, db)
	color.Green("******* STATE HANDLE WALLET = %d *******", TMState)
	if TMState == constants.TMState0 {
		helpers.Send(b,u, templates.FollowSequence)
		return
	}
	helpers.SetState(b, u, constants.TMState, constants.TMState1, db)
	if IsValidTMAccount(u) != "" {
		err := db.Insert(constants.WalletTM, u.Message.From.UserName, u.Message.Text)
		if err != nil {
			helpers.Send(b, u, templates.Error)
			return
		}

		btnOpts := []string{constants.TenD, constants.OneM, constants.ThreeM}
		opts := models.ButtonHelper{
			Type:   constants.ReplyButton,
			Labels: btnOpts,
		}
		helpers.Send(b, u, "Attached Tendermint wallet to user successfully")
		helpers.Send(b, u, `Please select how much bandwidth you need by clicking on one of the buttons below: `, opts)
		return
	}
	helpers.Send(b, u, templates.Error)
	return
}

func HandleBWTM(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	resp, err := db.Read(constants.BandwidthTM, u.Message.From.UserName)

	if err != nil {
		err := db.Insert(constants.BandwidthTM, u.Message.From.UserName, u.Message.Text[:2])
		if err != nil {
			helpers.Send(b, u, templates.Error)
			return
		}
		switch u.Message.Text {
		case constants.TenD:
			helpers.SubscriptionPeriod(b, u, db,
				constants.TenDays, constants.TenderMintNetwork, constants.NodeBasePrice, constants.TenD,
			)
		case constants.OneM:
			helpers.SubscriptionPeriod(b, u, db,
				constants.Month, constants.TenderMintNetwork, constants.NodeMonthPrice, constants.OneM,
			)
		case constants.ThreeM:
			helpers.SubscriptionPeriod(b, u, db,
				constants.ThreeMonths, constants.TenderMintNetwork, constants.NodeThreeMonthPrice, constants.ThreeM,
			)
		}

		helpers.Send(b, u, templates.AskToSelectANode)
		for idx, node := range nodes {
			geo, err := proxy.GetGeoLocation(node.IPAddr)

			if err != nil {
				helpers.Send(b, u, err.Error())
				return
			}
			msg := fmt.Sprintf(templates.NodeList, strconv.Itoa(idx+1), geo.Country, node.Username, node.WalletAddress)
			helpers.Send(b, u, msg)
		}
		return
	}

	nodeIdx, err := strconv.ParseInt(resp.Value[0:2], 10, 64)
	if err != nil {
		helpers.Send(b, u, err.Error())
		return
	}

	var n models.TONNode
	for i := 0; i < len(nodes); i++ {
		if i == int(nodeIdx) {
			n = nodes[i]
			return
		}
	}
	url := fmt.Sprintf(constants.ProxyURL, n.IPAddr, strconv.Itoa(n.Port), n.Username, n.Password)
	btnOpts := []models.InlineButtonOptions{
		{
			Label: "Sentinel Proxy Node",
			URL:   url,
		},
	}
	msg := fmt.Sprintf("you have already selected : Node %s", resp.Value)
	opts := models.ButtonHelper{
		Type:               constants.InlineButton,
		InlineKeyboardOpts: btnOpts,
	}
	helpers.Send(b, u, msg, opts)
}

func HandleTMNodeID(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	NodeId := u.Message.Text
	idx, _ := strconv.Atoi(NodeId)
	if idx > len(nodes) {
		helpers.Send(b, u, templates.Error)
		return
	}

	values := []models.KV{
		{Key: constants.NodeTM, Value: NodeId},
		{Key: constants.NodeWalletTM, Value: nodes[idx-1].WalletAddress},
	}
	err := db.MultiWriter(values, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}

	kv, err := db.Read(constants.NodePriceTM, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	msg := fmt.Sprintf(templates.AskForPayment, kv.Value)

	helpers.Send(b, u, msg)
	helpers.Send(b, u, nodes[idx-1].WalletAddress)
}

func parseTxnAmount(amount string) string {
	f, e := strconv.ParseFloat(amount, 64)
	if e != nil {
		return ""
	}
	return fmt.Sprintf("%0.0f", f*math.Pow(10, -8))
}
