package templates

const (
	GreetingMsg = `Hey %s, welcome to the Sentinel Socks5 Proxy Bot for Telegram.

Please select a blockchain network for payments to this bot.`
	NodeAttachedAlready = "you already have a node assigned to your username. Please use /mynode to access it"

	CheckWalletOptionsError = "error while fetching user wallet address. in case you have not attached your wallet address, please share your wallet address again."
	Success                 = "Congratulations!! please click the button below to connect to the sentinel dVPN node and next time use /mynode to access this node"
	AskToSelectANode        = `Please select a node ID from the list below and reply in the format of
1 for Node 1, 2 for Node 2 and so on...`
	UserInfo = `Bandwidth Duration Left: <b>%0.0f days</b>
Ethereum Wallet Attached: <b>%s</b>`
	AskForEthWallet   = "Please share your ethereum wallet address that you want to use for transactions to this bot"
	AskForPayment     = "please send %s $SENTS to the following address and submit the transaction hash here: "
	AskForTMWallet    = "Please share your tendermint wallet address that you want to use for transactions to this bot"
	AskForBW          = "Please select how much bandwidth you need by clicking on one of the buttons below: "
	BWError           = "error while storing bandwidth price"
	NodeList          = "%s.) Location: %s\n User: %s \n Node wallet: %s"
	BWPeriods         = "you have opted for %s of unlimited bandwidth"
	Error             = "could not read user info"
	BWAttachmentError = "error occurred while adding user details for bandwidth requirements"
	ConnectMessage    = "please click on the button below to connect to Sentinel's SOCKS5 Proxy"
	NoEthNodes        = "no nodes available right now. please check again later or try our Tendermint network"
	NoTMNodes         = "no nodes available right now. please check again later or try our Ethereum network"
)
