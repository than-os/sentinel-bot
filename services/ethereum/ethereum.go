package ethereum

import (
	"encoding/json"
	"fmt"
	"github.com/than-os/sentinel-bot/buttons"
	"github.com/than-os/sentinel-bot/services"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/services/proxy"
	"github.com/than-os/sentinel-bot/templates"
	"gopkg.in/telegram-bot-api.v4"
)

func HandleWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB) {

	ok := common.IsHexAddress(u.Message.Text)
	if ok {
		err := db.Insert(constants.EthAddr, u.Message.From.UserName, u.Message.Text)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while storing user eth address")
			_, _ = b.Send(c)
			return
		}
		services.Send(b, u, "Attached the ETH wallet to user successfully")
		opts := models.ButtonHelper{
			Type:   constants.ReplyButton,
			Labels: []string{constants.TenD, constants.OneM, constants.ThreeM},
		}
		services.Send(b, u, templates.AskForBW, opts)

		err = db.Insert(constants.EthAddr, u.Message.From.UserName, u.Message.Text)
		if err != nil {
			services.Send(b, u, "could not store your wallet")
			return
		}

		return
	}

	services.Send(b, u, "internal bot error")
	return
}

func HandleEthBW(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {

	resp, err := db.Read(constants.Bandwidth, u.Message.From.UserName)

	if err != nil {
		err := db.Insert(constants.Bandwidth, u.Message.From.UserName, u.Message.Text[:2])
		if err != nil {
			return
		}
		switch u.Message.Text {
		case constants.TenD:
			subscriptionPeriod(b, u, db, constants.TenDays, constants.NodeBasePrice, constants.TenD)
		case constants.OneM:
			subscriptionPeriod(b, u, db, constants.Month, constants.NodeMonthPrice, constants.OneM)
		case constants.ThreeM:
			subscriptionPeriod(b, u, db, constants.ThreeMonths, constants.NodeThreeMonthPrice, constants.ThreeM)
		}

		services.Send(b, u, templates.AskToSelectANode)
		for idx, node := range nodes {
			geo, err := proxy.GetGeoLocation(node.IPAddr)

			if err != nil {
				services.Send(b, u, err.Error())
				return
			}
			msg := fmt.Sprintf(templates.NodeList, strconv.Itoa(idx+1), geo.Country, node.Username, node.WalletAddress)
			services.Send(b, u, msg)
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
	buttonOptions := models.InlineButtonOptions{
		Label: "Sentinel Proxy Node", URL: uri,
	}
	c.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: buttons.InlineButtons(buttonOptions),
	}
	_, _ = b.Send(c)
}

func AskForEthWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {

	if len(nodes) == 0 {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, constants.NoEthNodes)
		c.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard:        buttons.ReplyButtons(constants.TenderMintNetwork),
			ResizeKeyboard:  true,
			Selective:       true,
			OneTimeKeyboard: true,
		}
		_, _ = b.Send(c)
		return
	}

	err := db.Insert(constants.BlockchainNetwork, u.Message.From.UserName, constants.EthNetwork)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "internal bot error")
		_, _ = b.Send(c)
	}
	c := tgbotapi.NewMessage(u.Message.Chat.ID, templates.AskForEthWallet)
	_, _ = b.Send(c)
}

func AskForTendermintWallet(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {

	if len(nodes) == 0 {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, constants.NoTMNodes)
		c.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
			Keyboard: buttons.ReplyButtons(constants.EthNetwork),
		}
		_, _ = b.Send(c)
		return
	}

	err := db.Insert(constants.BlockchainNetwork, u.Message.From.UserName, constants.TenderMintNetwork)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "internal bot error")
		_, _ = b.Send(c)
	}
	c := tgbotapi.NewMessage(u.Message.Chat.ID, templates.AskForTMWallet)
	_, _ = b.Send(c)
}

func HandleTxHash(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	resp, err := db.Read(constants.Node, u.Message.From.UserName)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not get user info")
		_, _ = b.Send(c)
		return
	}
	UserWallet, err := db.Read(constants.EthAddr, u.Message.From.UserName)
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
	if FindTxByHash(u.Message.Text, UserWallet.Value, u, db) {
		uri := "https://t.me/socks?server=" + nodes[idx].IPAddr + "&port=" + strconv.Itoa(nodes[idx].Port) + "&user=" + nodes[idx].Username + "&pass=" + nodes[idx].Password

		err := db.Insert(constants.IPAddr, u.Message.From.UserName, nodes[idx].IPAddr)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error in adding user details")
			_, _ = b.Send(c)
			return
		}
		err = db.Insert(constants.AssignedNodeURI, u.Message.From.UserName, uri)

		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error in adding user details")
			_, _ = b.Send(c)
			return
		}
		err = db.Insert(constants.IsAuth, u.Message.From.UserName, "true")
		if err != nil {
			tgbotapi.NewMessage(u.Message.Chat.ID, "error while adding user to auth group. please try again")
			return
		}
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "Thanks for submitting the TX-HASH. We're validating it")
		_, _ = b.Send(c)
		c = tgbotapi.NewMessage(u.Message.Chat.ID, "creating new user for "+u.Message.From.UserName+"...")
		_, _ = b.Send(c)

		node := nodes[idx]
		err = proxy.AddUser(node.IPAddr, u.Message.From.UserName, db, constants.Password)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "Error while creating SOCKS5 user for "+u.Message.From.UserName)
			_, _ = b.Send(c)
			return
		}
		pass, err := db.Read(constants.Password, u.Message.From.UserName)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while getting user pass")
			_, _ = b.Send(c)

			return
		}
		uri = "https://t.me/socks?server=" + node.IPAddr + "&port=" + strconv.Itoa(node.Port) + "&user=" + u.Message.From.UserName + "&pass=" + fmt.Sprintf("%s", pass.Value)
		err = db.Insert(constants.IPAddr, u.Message.From.UserName, nodes[idx].IPAddr)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error in adding user details")
			_, _ = b.Send(c)

			return
		}
		err = db.Insert(constants.AssignedNodeURI, u.Message.From.UserName, uri)
		if err != nil {
			c := tgbotapi.NewMessage(u.Message.Chat.ID, "error while adding user details")
			_, _ = b.Send(c)

			return
		}
		buttonOptions := models.InlineButtonOptions{
			Label: nodes[idx].Username, URL: uri,
		}
		c = tgbotapi.NewMessage(u.Message.Chat.ID, constants.Success)
		c.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: buttons.InlineButtons(buttonOptions),
		}
		_, _ = b.Send(c)

		return
	}
	c := tgbotapi.NewMessage(u.Message.Chat.ID, "invalid TXN Hash. Please try again")
	_, _ = b.Send(c)

}

func FindTxByHash(txHash, walletAddr string, u tgbotapi.Update, db ldb.BotDB) bool {

	wallet := "0x" + constants.ZFill + strings.TrimLeft(walletAddr, "0x")
	uri := constants.TestSentURI1 + wallet + constants.TestSendURI2 + wallet
	resp, err := http.Get(uri)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var body models.TxReceiptList
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}

	var val bool
	user, err := db.Read(constants.EthAddr, u.Message.From.UserName)
	if err != nil {
		return false
	}
	node, err := db.Read(constants.NodeWallet, u.Message.From.UserName)
	if err != nil {
		return false
	}
	// userWallet := fmt.Sprintf("%s", w.Value)
	for _, txReceipt := range body.Results {

		if txReceipt.TransactionHash == txHash {
			// nodeWallet := "0xceb5bc384012f0eebee119d82a24925c47714fe3"
			d, e := db.Read(constants.Timestamp, u.Message.From.UserName)
			if e != nil {
				return false
			}
			duration, err := time.Parse(time.RFC3339, fmt.Sprintf("%s", d.Value))
			if err != nil {
				return false
			}
			diff := math.Ceil(duration.Sub(time.Now()).Hours() / 24)

			okWallet := strings.EqualFold(txReceipt.Topics[1], "0x"+constants.ZFill+strings.TrimLeft(user.Value, "0x"))

			okRecipient := strings.EqualFold(txReceipt.Topics[2], "0x"+constants.ZFill+strings.TrimLeft(node.Value, "0x"))
			okAmount := false
			if diff == 10 {
				okAmount = hex2int(txReceipt.Data) == uint64(1000000000)
			} else if diff == 30 {
				okAmount = hex2int(txReceipt.Data) == uint64(3000000000)
			} else {
				okAmount = hex2int(txReceipt.Data) == uint64(8000000000)
			}
			color.Red("comparison: %v%v%v%v", hex2int(txReceipt.Data), okAmount, okRecipient, okWallet)
			if okWallet && okRecipient && okAmount {
				val = true
			}
		}
	}
	return val
}

func hex2int(hexStr string) uint64 {
	// remove 0x suffix if found in the input string
	cleaned := strings.Replace(hexStr, "0x", "", -1)

	// base 16 for hexadecimal
	result, _ := strconv.ParseUint(cleaned, 16, 64)
	return uint64(result)
}

func HandleNodeID(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, nodes []models.TONNode) {
	NodeId := u.Message.Text
	idx, _ := strconv.Atoi(NodeId)
	if idx > len(nodes) {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "invalid node id")
		_, _ = b.Send(c)
		return
	}
	err := db.Insert(constants.Node, u.Message.From.UserName, NodeId)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not store user info")
		_, _ = b.Send(c)
		return
	}
	err = db.Insert(constants.NodeWallet, u.Message.From.UserName, nodes[idx-1].WalletAddress)
	if err != nil {
		c := tgbotapi.NewMessage(u.Message.Chat.ID, "could not store node wallet address for payments")
		_, _ = b.Send(c)
		return
	}

	kv, err := db.Read(constants.NodePrice, u.Message.From.UserName)
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

func subscriptionPeriod(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, t time.Duration, price, period string) {
	//t := constants.TenDays
	pairs := []models.KV{
		{
			Key: constants.Timestamp, Value: time.Now().Add(t).Format(time.RFC3339),
		},
		{
			Key: constants.NodePrice, Value: price,
		},
	}
	err := db.MultiWriter(pairs, u.Message.From.UserName)
	if err != nil {
		services.Send(b, u, templates.BWError)
		return
	}
	msg := fmt.Sprintf(templates.BWPeriods, period)
	services.Send(b, u, msg)
}