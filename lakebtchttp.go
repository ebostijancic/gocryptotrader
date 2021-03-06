package main

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	LAKEBTC_API_URL          = "https://www.LakeBTC.com/api_v1/"
	LAKEBTC_API_VERSION      = "1"
	LAKEBTC_TICKER           = "ticker"
	LAKEBTC_ORDERBOOK        = "bcorderbook"
	LAKEBTC_ORDERBOOK_CNY    = "bcorderbook_cny"
	LAKEBTC_TRADES           = "bctrades"
	LAKEBTC_GET_ACCOUNT_INFO = "getAccountInfo"
	LAKEBTC_BUY_ORDER        = "buyOrder"
	LAKEBTC_SELL_ORDER       = "sellOrder"
	LAKEBTC_GET_ORDERS       = "getOrders"
	LAKEBTC_CANCEL_ORDER     = "cancelOrder"
	LAKEBTC_GET_TRADES       = "getTrades"
)

type LakeBTC struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	Email, APISecret        string
	TakerFee, MakerFee      float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type LakeBTCTicker struct {
	Last   float64
	Bid    float64
	Ask    float64
	High   float64
	Low    float64
	Volume float64
}

type LakeBTCOrderbook struct {
	Bids [][]float64 `json:"asks"`
	Asks [][]float64 `json:"bids"`
}

type LakeBTCTickerResponse struct {
	USD LakeBTCTicker
	CNY LakeBTCTicker
}

func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = false
	l.TakerFee = 0.2
	l.MakerFee = 0.15
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
}

func (l *LakeBTC) GetName() string {
	return l.Name
}

func (l *LakeBTC) SetEnabled(enabled bool) {
	l.Enabled = enabled
}

func (l *LakeBTC) IsEnabled() bool {
	return l.Enabled
}

func (l *LakeBTC) Setup(exch Exchanges) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *LakeBTC) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (l *LakeBTC) Start() {
	go l.Run()
}

func (l *LakeBTC) SetAPIKeys(apiKey, apiSecret string) {
	l.Email = apiKey
	l.APISecret = apiSecret
}

func (l *LakeBTC) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	} else {
		return l.TakerFee
	}
}

func (l *LakeBTC) Run() {
	if l.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", l.GetName(), IsEnabled(l.Websocket), LAKEBTC_WEBSOCKET_URL)
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	if l.Websocket {
		go l.WebsocketClient()
	}

	for l.Enabled {
		ticker := l.GetTicker()
		for _, x := range l.EnabledPairs {
			if x == "BTCUSD" {
				log.Printf("LakeBTC BTC USD: Last %f High %f Low %f Volume %f\n", ticker.USD.Last, ticker.USD.High, ticker.USD.Low, ticker.USD.Volume)
				AddExchangeInfo(l.GetName(), x[0:3], x[3:], ticker.USD.Last, ticker.USD.Volume)
			} else if x == "BTCCNY" {
				log.Printf("LakeBTC BTC CNY: Last %f High %f Low %f Volume %f\n", ticker.CNY.Last, ticker.CNY.High, ticker.CNY.Low, ticker.CNY.Volume)
				AddExchangeInfo(l.GetName(), x[0:3], x[3:], ticker.CNY.Last, ticker.CNY.Volume)
			}
		}
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *LakeBTC) GetTicker() LakeBTCTickerResponse {
	response := LakeBTCTickerResponse{}
	err := SendHTTPGetRequest(LAKEBTC_API_URL+LAKEBTC_TICKER, true, &response)
	if err != nil {
		log.Println(err)
		return response
	}
	return response
}

func (l *LakeBTC) GetTickerPrice(currency string) TickerPrice {
	var tickerPrice TickerPrice
	ticker := l.GetTicker()

	if currency == "USD" {
		tickerPrice.Ask = ticker.USD.Ask
		tickerPrice.Bid = ticker.USD.Bid
	} else if currency == "CNY" {
		tickerPrice.Ask = ticker.CNY.Ask
		tickerPrice.Bid = ticker.CNY.Bid
	}

	tickerPrice.CryptoCurrency = currency

	return tickerPrice
}

func (l *LakeBTC) GetOrderBook(currency string) bool {
	req := LAKEBTC_ORDERBOOK
	if currency == "CNY" {
		req = LAKEBTC_ORDERBOOK_CNY
	}

	err := SendHTTPGetRequest(LAKEBTC_API_URL+req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (l *LakeBTC) GetTradeHistory() bool {
	err := SendHTTPGetRequest(LAKEBTC_API_URL+LAKEBTC_TRADES, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (l *LakeBTC) GetAccountInfo() {
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ACCOUNT_INFO, "")

	if err != nil {
		log.Println(err)
	}
}

//TODO Get current holdings from LakeBTC
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the LakeBTC exchange
func (e *LakeBTC) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}

func (l *LakeBTC) Trade(orderType int, amount, price float64, currency string) {
	params := strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(amount, 'f', -1, 64) + "," + currency
	err := errors.New("")

	if orderType == 0 {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_BUY_ORDER, params)
	} else {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_SELL_ORDER, params)
	}

	if err != nil {
		log.Println(err)
	}
}

func (l *LakeBTC) GetOrders() {
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ORDERS, "")
	if err != nil {
		log.Println(err)
	}
}

func (l *LakeBTC) CancelOrder(orderID int64) {
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_CANCEL_ORDER, params)
	if err != nil {
		log.Println(err)
	}
}

func (l *LakeBTC) GetTrades(timestamp time.Time) {
	params := ""

	if !timestamp.IsZero() {
		params = strconv.FormatInt(timestamp.Unix(), 10)
	}

	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_TRADES, params)
	if err != nil {
		log.Println(err)
	}
}

func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	v := url.Values{}
	v.Set("tnonce", nonce)
	v.Set("accesskey", l.Email)
	v.Set("requestmethod", "POST")
	v.Set("id", nonce)
	v.Set("method", method)
	v.Set("params", params)

	encoded := v.Encode()
	hmac := GetHMAC(HASH_SHA256, []byte(encoded), []byte(l.APISecret))

	if l.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", LAKEBTC_API_URL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Json-Rpc-Tonce"] = nonce
	headers["Authorization: Basic"] = Base64Encode([]byte(l.Email + ":" + HexEncodeToString(hmac)))
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", LAKEBTC_API_URL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	if l.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	return nil
}
