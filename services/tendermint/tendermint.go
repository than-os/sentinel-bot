package tendermint

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/than-os/sentinel-bot/buttons"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/services/proxy"
	"github.com/than-os/sentinel-bot/templates"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
		color.Red("error0: %s", err.Error())
		return txnResp
	}
	if err = json.NewDecoder(resp.Body).Decode(&txnResp); err != nil {
		color.Red("error: %s", err.Error())
		return txnResp
	}
	color.Cyan("resp: %s", txnResp)
	return txnResp
}

func HandleTMTxnHash(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	resp, err := db.Read(constants.NodeTM, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not get user info")
		_, _ = b.Send(c)
		return
	}

	respToStr := fmt.Sprintf("%s", resp.Value)
	strToInt, err := strconv.Atoi(respToStr)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "ASCII to INT conversion error")
		_, _ = b.Send(c)
		return
	}

	idx := strToInt - 1
	if IsValidTMTxn(u, db) {
		uri := "https://t.me/socks?server=" + nodes[idx].IPAddr + "&port=" + strconv.Itoa(nodes[idx].Port) + "&user=" + nodes[idx].Username + "&pass=" + nodes[idx].Password

		err := db.Insert(constants.IPAddrTM, u.Message.From.UserName, nodes[idx].IPAddr)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error in adding user details")
			_, _ = b.Send(c)
			return
		}
		err = db.Insert(constants.AssignedNodeURITM, u.Message.From.UserName, uri)

		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error in adding user details")
			_, _ = b.Send(c)
			return
		}
		err = db.Insert(constants.IsAuthTM, u.Message.From.UserName, "true")
		if err != nil {
			tgbotapi.NewMessage(u.Message.Chat.ID, "error while adding user to auth group. please try again")
			return
		}
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "Thanks for submitting the TX-HASH. We're validating it")
		_, _ = b.Send(c)
		c = tgbotapi.NewMessage(u.Message.Chat.ID, "creating new user for "+u.Message.From.UserName+"...")
		_, _ = b.Send(c)

		node := nodes[idx]
		err = proxy.AddUser(node.IPAddr, u.Message.From.UserName, db, constants.PasswordTM)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "Error while creating SOCKS5 user for "+u.Message.From.UserName)
			_, _ = b.Send(c)
			return
		}
		pass, err := db.Read(constants.PasswordTM, u.Message.From.UserName)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while getting user pass")
			_, _ = b.Send(c)

			return
		}
		uri = "https://t.me/socks?server=" + node.IPAddr + "&port=" + strconv.Itoa(node.Port) + "&user=" + u.Message.From.UserName + "&pass=" + fmt.Sprintf("%s", pass.Value)
		err = db.Insert(constants.IPAddrTM, u.Message.From.UserName, nodes[idx].IPAddr)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error in adding user details")
			_, _ = b.Send(c)

			return
		}
		err = db.Insert(constants.AssignedNodeURITM, u.Message.From.UserName, uri)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while adding user details")
			_, _ = b.Send(c)

			return
		}
		btnOpts := []models.InlineButtonOptions{
			{
				Label: nodes[idx].Username,
				URL: uri,
			},
		}
		c = tgbotapi.NewMessage(u.Message.Chat.ID, constants.Success)
		c.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: buttons.InlineButtons(btnOpts),
		}
		_, _ = b.Send(c)

		return
	}
	c := tgbotapi.NewMessage(u.Message.Chat.ID, "invalid TXN Hash. Please try again")
	_, _ = b.Send(c)

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
	replyButtons := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.KeyboardButton{Text: "10 Days"},
		},
		{
			tgbotapi.KeyboardButton{Text: "30 Days"},
		},
		{
			tgbotapi.KeyboardButton{Text: "90 Days"},
		},
	}
	ok := IsValidTMAccount(u)
	if ok != "" {
		err := db.Insert(constants.WalletTM, u.Message.From.UserName, u.Message.Text)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while storing user eth address")
			_, _ = b.Send(c)
			return
		}
		c1 := tgbotapi.NewMessage(u.Message.Chat.ID, "Attached Tendermint wallet to user successfully")
		c2 := tgbotapi.NewMessage(u.Message.Chat.ID, `Please select how much bandwidth you need by clicking on one of the buttons below: `)
		_, _ = b.Send(c1)
		c2.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard:        replyButtons,
			OneTimeKeyboard: true,
			ResizeKeyboard:  true,
		}
		_, _ = b.Send(c2)
		err = db.Insert(constants.WalletTM, u.Message.From.UserName, u.Message.Text)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not store your wallet")
			_, _ = b.Send(c)
		}

		return
	}
	c := tgbotapi.NewMessage(u.Message.Chat.ID, "internal bot error")
	_, _ = b.Send(c)
	return
}

func HandleBWTM(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	resp, err := db.Read(constants.BandwidthTM, u.Message.From.UserName)

	if err != nil {
		log.Println("error in checkUserOptions", err.Error())

		err := db.Insert(constants.BandwidthTM, u.Message.From.UserName, u.Message.Text[:2])
		if err != nil {
			return
		}
		switch u.Message.Text {
		case constants.TenD:
			t := constants.TenDays
			err := db.Insert(constants.TimestampTM, u.Message.From.UserName, time.Now().Add(t).Format(time.RFC3339))
			if err != nil {
				c := tgbotapi.NewMessage(u.Message.Chat.ID, constants.BWAttachmentError)
				_, _ = b.Send(c)
				return
			}
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "you have opted for 10 days of unlimited bandwidth")
			_, _ = b.Send(c)
			err = db.Insert(constants.NodePriceTM, u.Message.From.UserName, constants.NodeBasePrice)
			if err != nil {
				c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while storing bandwidth price")
				_, _ = b.Send(c)
				return
			}
		case constants.OneM:
			t := constants.Month
			err := db.Insert(constants.TimestampTM, u.Message.From.UserName, time.Now().Add(t).Format(time.RFC3339))
			if err != nil {

			}
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "you have opted for 30 days of unlimited bandwidth")
			_, _ = b.Send(c)
			err = db.Insert(constants.NodePriceTM, u.Message.From.UserName, constants.NodeMonthPrice)
			if err != nil {
				c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while storing bandwidth price")
				_, _ = b.Send(c)
				return
			}
		case constants.ThreeM:
			t := constants.ThreeMonths
			err := db.Insert(constants.TimestampTM, u.Message.From.UserName, time.Now().Add(t).Format(time.RFC3339))
			if err != nil {
				_, _ = b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, err.Error()))
				return
			}
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "you have opted for 90 days of unlimited bandwidth")
			_, _ = b.Send(c)
			err = db.Insert(constants.NodePriceTM, u.Message.From.UserName, constants.NodeThreeMonthPrice)
			if err != nil {
				c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while storing bandwidth price")
				_, _ = b.Send(c)
				return
			}
		}
		c := tgbotapi.NewMessage(u.Message.Chat.ID, templates.AskToSelectANode)
		_, _ = b.Send(c)
		for idx, node := range nodes {
			geo, err := proxy.GetGeoLocation(node.IPAddr)

			if err != nil {
				c := tgbotapi.NewMessage(u.Message.Chat.ID, err.Error())
				_, _ = b.Send(c)
				return
			}
			c := tgbotapi.NewMessage(u.Message.Chat.ID, strconv.Itoa(idx+1)+". Location: "+geo.Country+"\n "+"User:"+node.Username+"\n "+"Node Wallet: "+node.WalletAddress)
			_, _ = b.Send(c)
		}
		return
	}

	nodeIdx, err := strconv.ParseInt(resp.Value[0:2], 10, 64)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, err.Error())
		_, _ = b.Send(c)
		return
	}

	var n models.TONNode
	for i := 0; i < len(nodes); i++ {
		if i == int(nodeIdx) {
			n = nodes[i]
			return
		}
	}
	uri := "https://t.me/socks?server=" + n.IPAddr + "&port=" + strconv.Itoa(n.Port) + "&user=" + n.Username + "&pass=" + n.Password
	c := tgbotapi.NewMessage(u.Message.Chat.ID, "you have already selected : Node "+fmt.Sprintf("%s", resp.Value))
	btnOpts := []models.InlineButtonOptions{
		{
			Label: "Sentinel Proxy Node",
			URL: uri,
		},
	}
	c.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: buttons.InlineButtons(btnOpts),
	}
	_, _ = b.Send(c)
}

func HandleTMNodeID(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	NodeId := u.Message.Text
	idx, _ := strconv.Atoi(NodeId)
	if idx > len(nodes) {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "invalid node id")
		_, _ = b.Send(c)
		return
	}
	err := db.Insert(constants.NodeTM, u.Message.From.UserName, NodeId)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not store user info")
		_, _ = b.Send(c)
		return
	}
	err = db.Insert(constants.NodeWalletTM, u.Message.From.UserName, nodes[idx-1].WalletAddress)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not store node wallet address for payments")
		_, _ = b.Send(c)
		return
	}

	kv, err := db.Read(constants.NodePriceTM, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not get node's price for bandwidth")
		_, _ = b.Send(c)
		return
	}
	msg := fmt.Sprintf(templates.AskForPayment, kv.Value)
	color.Red("crazy: %s\n%s", nodes, kv)
	c := tgbotapi.NewMessage(u.Message.Chat.ID, msg)
	_, _ = b.Send(c)
	c = tgbotapi.NewMessage(u.Message.Chat.ID, nodes[idx-1].WalletAddress) //should be node wallet address
	_, _ = b.Send(c)
}

func parseTxnAmount(amount string) string {
	//txn.Tx.Value.Msg[0].Value.Coins[0].Amount
	f, e := strconv.ParseFloat(amount, 64)
	if e != nil {
		return ""
	}
	return fmt.Sprintf("%0.0f", f*math.Pow(10, -8))
}
