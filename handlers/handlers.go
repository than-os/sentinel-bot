package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/than-os/sentinel-bot/buttons"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/services"
	"github.com/than-os/sentinel-bot/services/proxy"
	"github.com/than-os/sentinel-bot/services/tendermint"
	"github.com/than-os/sentinel-bot/templates"
	"gopkg.in/telegram-bot-api.v4"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func Greet(b *tgbotapi.BotAPI, u tgbotapi.Update) {
	greet := fmt.Sprintf(templates.GreetingMsg, u.Message.From.UserName)

	c := tgbotapi.NewMessage(u.Message.Chat.ID, greet)
	c.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard:        buttons.ReplyButtons(constants.EthNetwork, constants.TenderMintNetwork),
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}
	_, _ = b.Send(c)
}

func GetNodes() (models.Nodes, error) {
	var N models.Nodes
	var body []models.TONNode
	resp, err := http.Get("https://ton.sentinelgroup.io/all")
	if err != nil {
		return N, err
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return N, err
	}
	defer resp.Body.Close()

	for _, node := range body {
		if node.Type == "tendermint" {
			N.TMNodes = append(N.TMNodes, node)
		} else {
			N.EthNodes = append(N.EthNodes, node)
		}
	}
	return N, err
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
		go services.AskForEthWallet(b, u, db, nodes.EthNodes)
	case constants.TenderMintNetwork:
		go services.AskForTendermintWallet(b, u, db, nodes.TMNodes)
	case isEthAddr(u):
		go services.HandleWallet(b, u, db)
	case tendermint.IsValidTMAccount(u):
		go tendermint.HandleWallet(b, u, db)
	case constants.TenD, constants.OneM, constants.ThreeM:
		go HandleBW(b, u, db, nodes)
	case isNodeID(u):
		go HandleNodeId(b, u, db, nodes)
	case isTxn(u):
		go services.HandleTxHash(b, u, db, nodes.EthNodes)
	case tendermint.IsTMTxnHash(u):
		go tendermint.HandleTMTxnHash(b, u, db, nodes.TMNodes)
	default:
		if !u.Message.IsCommand() {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "invalid option")
			_, _ = b.Send(c)
		}
	}
}

func ShowEthWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	kv, err := db.Read(constants.EthAddr, u.Message.From.UserName)
	if err != nil {

		c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while finding user wallet")
		_, _ = b.Send(c)
		return
	}

	c := tgbotapi.NewMessage(u.Message.Chat.ID, kv.Value)
	_, _ = b.Send(c)
}

func ShowMyNode(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	kv, err := db.Read(constants.AssignedNodeURI, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while finding node url")
		_, _ = b.Send(c)
		return
	}
	btnOpts := models.InlineButtonOptions{
		Label: "Proxy Node",
		URL:   kv.Value,
	}
	c := tgbotapi.NewMessage(u.Message.Chat.ID, constants.ConnectMessage)
	c.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: buttons.InlineButtons(btnOpts),
	}
	_, _ = b.Send(c)
}

func Restart(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	kv, err := db.Read(constants.IPAddr, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "not found")
		_, _ = b.Send(c)
		return
	}
	err = proxy.DeleteUser(u.Message.From.UserName, kv.Value)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "user not deleted")
		_, _ = b.Send(c)
	}
	err = db.RemoveETHUser(u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "not found")
		_, _ = b.Send(c)
		return
	}
	err = db.RemoveTMUser(u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "not found")
		_, _ = b.Send(c)
		return
	}
	greet := fmt.Sprintf(templates.GreetingMsg, u.Message.From.UserName)
	c := tgbotapi.NewMessage(u.Message.Chat.ID, greet)
	c.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard: buttons.ReplyButtons(constants.EthNetwork, constants.TenderMintNetwork),
	}
	_, _ = b.Send(c)
}

func ShowMyInfo(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {
	bw, err := db.Read(constants.Timestamp, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "not found")
		_, _ = b.Send(c)
		return
	}
	wallet, err := db.Read(constants.EthAddr, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "not found")
		_, _ = b.Send(c)
		return
	}

	d, _ := time.Parse(time.RFC3339, bw.Value)
	days := math.Ceil(d.Sub(time.Now()).Hours()) / 24

	msg := fmt.Sprintf(templates.UserInfo, days, wallet.Value)
	c := tgbotapi.NewMessage(u.Message.Chat.ID, msg)
	c.ParseMode = "HTML"
	_, _ = b.Send(c)
}

func HandleNodeId(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes models.Nodes) {
	network, err := db.Read(constants.BlockchainNetwork, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, constants.BWAttachmentError)
		_, _ = b.Send(c)
	}
	if network.Value == constants.TenderMintNetwork {
		tendermint.HandleTMNodeID(b, u, db, nodes.TMNodes)
	}

	if network.Value == constants.EthNetwork {
		services.HandleNodeID(b, u, db, nodes.EthNodes)

	}

}

func HandleBW(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes models.Nodes) {
	network, err := db.Read(constants.BlockchainNetwork, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, constants.BWAttachmentError)
		_, _ = b.Send(c)
	}

	if network.Value == constants.TenderMintNetwork {
		tendermint.HandleBWTM(b, u, db, nodes.TMNodes)
	}

	if network.Value == constants.EthNetwork {
		services.HandleEthBW(b, u, db, nodes.EthNodes)
	}

}
