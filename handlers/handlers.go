package handlers

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/helpers"
	"github.com/than-os/sentinel-bot/services/ethereum"
	"github.com/than-os/sentinel-bot/services/proxy"
	"github.com/than-os/sentinel-bot/services/tendermint"
	"github.com/than-os/sentinel-bot/templates"
	"gopkg.in/telegram-bot-api.v4"
	"math"
	"regexp"
	"strconv"
	"time"
)

func Greet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	helpers.SetState(b, u, constants.EthState, constants.EthState0, db)
	helpers.SetState(b, u, constants.TMState, constants.TMState0, db)
	greet := fmt.Sprintf(templates.GreetingMsg, u.Message.From.UserName)
	btnOpts := []string{constants.EthNetwork, constants.TenderMintNetwork}
	opts := models.ButtonHelper{
		Type: constants.ReplyButton, Labels: btnOpts,
	}
	helpers.Send(b, u, greet, opts)
}

func AboutSentinel(b *tgbotapi.BotAPI, u tgbotapi.Update) {
	helpers.Send(b, u, templates.AboutSentinel)
}

func isEthAddr(u tgbotapi.Update) string {
	r, _ := regexp.Compile(constants.EthRegex)
	ok := common.IsHexAddress(u.Message.Text)

	if ok && r.MatchString(u.Message.Text) {
		return u.Message.Text
	}

	return ""
}

func isNodeID(u tgbotapi.Update) string {
	_, err := strconv.Atoi(u.Message.Text)
	if err != nil {
		return ""
	}

	return u.Message.Text
}

func isTxn(u tgbotapi.Update) string {
	_, err := hexutil.Decode(u.Message.Text)
	if err != nil {
		return ""
	}
	return u.Message.Text
}

func MainHandler(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes models.Nodes) {

	switch u.Message.Text {

	case constants.EthNetwork:
		go ethereum.AskForEthWallet(b, u, db, nodes.EthNodes)
	case constants.TenderMintNetwork:
		go tendermint.AskForTendermintWallet(b, u, db, nodes.TMNodes)
	case isEthAddr(u):
		go ethereum.HandleWallet(b, u, db)
	case tendermint.IsValidTMAccount(u):
		go tendermint.HandleWallet(b, u, db)
	case constants.TenD, constants.OneM, constants.ThreeM:
		go HandleBW(b, u, db, nodes)
	case isNodeID(u):
		go HandleNodeId(b, u, db, nodes)
	case isTxn(u):
		go ethereum.HandleTxHash(b, u, db, nodes.EthNodes)
	case tendermint.IsTMTxnHash(u):
		go tendermint.HandleTMTxnHash(b, u, db, nodes.TMNodes)
	default:
		if !u.Message.IsCommand() {
			helpers.Send(b, u, templates.InvalidOption)
		}
	}
}

func ShowEthWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	kv, err := db.Read(constants.EthAddr, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}

	helpers.Send(b, u, kv.Value)
}

func ShowMyNode(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	kv, err := db.Read(constants.AssignedNodeURI, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	btnOpts := []models.InlineButtonOptions{
		{Label: "Proxy Node", URL: kv.Value},
	}
	opts := models.ButtonHelper{
		Type: constants.InlineButton, InlineKeyboardOpts: btnOpts,
	}
	helpers.Send(b, u, templates.ConnectMessage, opts)

}

func Restart(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	kv, err := db.Read(constants.IPAddr, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	err = proxy.DeleteUser(u.Message.From.UserName, kv.Value)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	err = db.RemoveETHUser(u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	err = db.RemoveTMUser(u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	greet := fmt.Sprintf(templates.GreetingMsg, u.Message.From.UserName)
	btnOpts := []string{constants.EthNetwork, constants.TenderMintNetwork}
	opts := models.ButtonHelper{
		Type:   constants.ReplyButton,
		Labels: btnOpts,
	}
	helpers.Send(b, u, greet, opts)
}

func ShowMyInfo(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	bw, err := db.Read(constants.Timestamp, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}
	wallet, err := db.Read(constants.EthAddr, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.Error)
		return
	}

	d, _ := time.Parse(time.RFC3339, bw.Value)
	days := math.Ceil(d.Sub(time.Now()).Hours()) / 24
	msg := fmt.Sprintf(templates.UserInfo, days, wallet.Value)
	helpers.Send(b, u, msg)
}

func HandleNodeId(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes models.Nodes) {

	network, err := db.Read(constants.BlockchainNetwork, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.NoNetworkSelected)
		return
	}
	if network.Value == constants.TenderMintNetwork {
		TMState := helpers.GetState(b, u, constants.TMState, db)
		//color.Green("******* STATE NODE ID = %d *******", TMState)
		if TMState < constants.TMState3 {
			helpers.Send(b, u, templates.FollowSequence)
			return
		}
		tendermint.HandleTMNodeID(b, u, db, nodes.TMNodes)
		helpers.SetState(b, u, constants.TMState, constants.TMState4, db)
	}

	if network.Value == constants.EthNetwork {
		EthState := helpers.GetState(b, u, constants.EthState, db)
		//color.Green("******* STATE NODE ID = %d *******", EthState)
		if EthState <= constants.EthState1 {
			helpers.Send(b, u, templates.FollowSequence)
			return
		}
		ethereum.HandleNodeID(b, u, db, nodes.EthNodes)
		helpers.SetState(b, u, constants.EthState, constants.EthState3, db)

	}

}

func HandleBW(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes models.Nodes) {
	network, err := db.Read(constants.BlockchainNetwork, u.Message.From.UserName)
	if err != nil {
		helpers.Send(b, u, templates.BWAttachmentError)
	}

	if network.Value == constants.TenderMintNetwork {
		state := helpers.GetState(b, u, constants.TMState, db)
		//color.Green("******* STATE HANDLE BW = %d *******", state)

		if state < constants.TMState2 {
			helpers.Send(b, u, templates.FollowSequence)
			return
		}
		tendermint.HandleBWTM(b, u, db, nodes.TMNodes)
		helpers.SetState(b, u, constants.TMState, constants.TMState3, db)
	}

	if network.Value == constants.EthNetwork {
		EthState := helpers.GetState(b, u, constants.EthState, db)
		//color.Green("******* STATE HANDLE BW = %d *******", EthState)
		if EthState <= constants.EthState0 {
			helpers.Send(b, u, templates.FollowSequence)
			return
		}
		ethereum.HandleEthBW(b, u, db, nodes.EthNodes)
		helpers.SetState(b, u, constants.EthState, constants.TMState2, db)
	}

}
