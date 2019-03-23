package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
)

type RegisterRequest struct {
	IPAddr     string `json:"ipAddr"`
	Port       int    `json:"port"`
	Username   string `json:"userName"`
	Password   string `json:"password"`
	WalletAddr string `json:"walletAddr"`
	Type       string `json:"type"`
}

type TMConfig struct {
	Account struct {
		Address string `json:"address"`
	}
	APIPort     int    `json:"api_port"`
	Description string `json:"description"`
	OpenVPN     struct {
		EncMethod string `json:"enc_method"`
		Port      int    `json:"port"`
	}
	PricePerGB float64 `json:"price_per_gb"`
	Register   struct {
		Hash  string `json:"hash"`
		Token string `json:"token"`
	}
}

type ConfigData struct {
	Token       string  `json:"token"`
	EncMethod   string  `json:"enc_method"`
	AccountAddr string  `json:"account_addr"`
	PricePerGB  float64 `json:"price_per_gb"`
	TONPrice    float64 `json:"tonPrice"`
}

type KeepAliveRequest struct {
	Status     string `json:"status"`
	NodeIpAddr string `json:"nodeIPAddr"`
}

type KeepAliveResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

const (
	EthConfig    = "/root/.sentinel/config.data"
	TMConfigPath = "/root/.sentinel/config"
	url          = "https://ton.sentinelgroup.io"
	//dir = "/home/thanos/Desktop/data.json"
)

func main() {
	time.Sleep(time.Minute)
	Register(os.Args[1:2][0])

	http.HandleFunc("/live", status)
	log.Fatal(http.ListenAndServe(":3030", nil))

}

func status(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)
	msg := `{ "status": "up" }`
	_, _ = w.Write([]byte(msg))
}

func ReadConfig() (wallet string) {
	fi, err := os.OpenFile(EthConfig, os.O_RDWR, 0666)
	if err != nil {
		log.Println(err)
		return
	}

	defer fi.Close()
	b, err := ioutil.ReadAll(fi)
	if err != nil {
		log.Println("error in reading file: ", err)
		return
	}

	var config ConfigData
	if err := json.Unmarshal(b, &config); err != nil {
		log.Println("err in marshaling the data: ", err)
		return
	}

	b, err = json.Marshal(config)
	if err != nil {
		log.Println(err)
		return
	}

	f, e := os.Create(EthConfig)
	if e != nil {
		log.Fatal(e)
	}

	_, e = f.Write(b)
	if e != nil {
		log.Println(e)
		return
	}

	defer f.Close()
	wallet = config.AccountAddr
	//config.TONPrice = 5
	log.Printf("here's the file: \n%s", config)
	return wallet
}

func keepAlive() {
	IPAddr := GetIP()
	color.Green("%s", "status up...")
	data := KeepAliveRequest{
		NodeIpAddr: IPAddr,
		Status:     "up",
	}
	b, e := json.Marshal(data)
	if e != nil {
		log.Println("error in marshal: ", e)
		return
	}
	resp2, err := http.Post(url+"/keep-alive", "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Println("error while submitting keep alive job")
	}
	var KeepAliveResp KeepAliveResponse
	if err := json.NewDecoder(resp2.Body).Decode(&KeepAliveResp); err != nil {
		log.Println("decoding error: ", err)
		return
	}
	defer resp2.Body.Close()
	var body KeepAliveResponse
	if err = json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		log.Println(err)
	}

}

func GetIP() string {
	type IP struct {
		IPAddr string `json:"ip"`
	}
	var ip IP
	resp, err := http.Get("https://ipleak.net/json/")
	if err != nil {
		log.Println("error while getting ip: ", err)
		return ""
	}
	if err := json.NewDecoder(resp.Body).Decode(&ip); err != nil {
		log.Println("error in decoding: ", err)
		return ""
	}

	return ip.IPAddr
}

func Register(networkType string) string {

	ipAddr := GetIP()
	var walletAddr string
	if networkType == "tendermint" {
		walletAddr = ReadTendermintWallet()
	} else {
		walletAddr = ReadConfig()
	}

	url := url + "/register"
	data := RegisterRequest{
		Port:       1080,
		Username:   "sentinel",
		Password:   `MemsIr[OkAj4"}`,
		Type:       networkType,
		IPAddr:     ipAddr,
		WalletAddr: walletAddr,
	}
	b, e := json.Marshal(data)
	if e != nil {
		log.Println("error in marshal: ", e)
		return ""
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Println("error in post request: ", err)
		return ""
	}
	defer resp.Body.Close()

	b, e = ioutil.ReadAll(resp.Body)
	if e != nil {
		log.Println("error in reading body: ", e)
		return ""
	}

	return fmt.Sprintf("%s", b)
}

func ReadTendermintWallet() string {
	var tmConfig TMConfig
	b, e := ioutil.ReadFile(TMConfigPath)
	if e != nil {
		return ""
	}

	if e = json.Unmarshal(b, &tmConfig); e != nil {
		return ""
	}

	return tmConfig.Account.Address
}
