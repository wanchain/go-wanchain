// Copyright 2018 Wanchain Foundation Ltd
// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// faucet is a Ether faucet backed by a light client.
package fullFaucet

//go:generate go-bindata -nometadata -o website.go faucet.html

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/eth"
	"github.com/wanchain/go-wanchain/ethclient"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
	"html/template"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var (
	genesisFlag = flag.String("genesis", "", "Genesis json file to seed the chain with")
	apiPortFlag = flag.Int("apiport", 8080, "Listener port for the HTTP API connection")
	ethPortFlag = flag.Int("ethport", 17717, "Listener port for the devp2p connection")
	bootFlag    = flag.String("bootnodes", "", "Comma separated bootnode enode URLs to seed with")
	netFlag     = flag.Uint64("network", 0, "Network ID to use for the Ethereum protocol")
	statsFlag   = flag.String("ethstats", "", "Ethstats network monitoring auth string")

	netnameFlag = flag.String("faucet.name", "", "Network name to assign to the faucet")
	payoutFlag  = flag.Int("faucet.amount", 1, "Number of Ethers to pay out per user request")
	minutesFlag = flag.Int("faucet.minutes", 1440, "Number of minutes to wait between funding rounds")
	tiersFlag   = flag.Int("faucet.tiers", 3, "Number of funding tiers to enable (x3 time, x2.5 funds)")

	accJSONFlag = flag.String("account.json", "", "Key json file to fund user requests with")
	accPassFlag = flag.String("account.pass", "", "Decryption password to access faucet funds")

	githubUser  = flag.String("github.user", "", "GitHub user to authenticate with for Gist access")
	githubToken = flag.String("github.token", "", "GitHub personal token to access Gists with")

	captchaToken  = flag.String("captcha.token", "", "Recaptcha site key to authenticate client side")
	captchaSecret = flag.String("captcha.secret", "", "Recaptcha secret key to authenticate server side")

	noauthFlag = flag.Bool("noauth", false, "Enables funding requests without authentication")
	logFlag    = flag.Int("loglevel", 3, "Log level to use for Ethereum and the faucet")
)

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

func  (f *faucet) SendTransaction(singedTx *types.Transaction) error {

	ctx := context.Background()

	var txHash common.Hash
	data, err := rlp.EncodeToBytes(singedTx)
	if err != nil {
		return err
	}

	err = f.client.C.CallContext(ctx, &txHash, "eth_sendRawTransaction", common.ToHex(data))
	if nil != err {
		log.Error("send pos tx fail", "err", err)
		return err
	}

	log.Info("send pos tx success", "txHash", txHash)
	if err == nil {
		log.Info("send pos tx success", "txHash", txHash)
	} else {
		log.Info("send pos tx failed", "err", err)
	}

	return err

}

func FaucetStart(amount uint64,ethereum *eth.Ethereum,url string) {
	// Parse the flags and set up the logger to print everything requested

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*logFlag), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	if amount == 0 {
		amount = 100
	}

	*payoutFlag = int(amount)
	// Construct the payout tiers
	amounts := make([]string, *tiersFlag)
	periods := make([]string, *tiersFlag)
	for i := 0; i < *tiersFlag; i++ {
		// Calculate the amount for the next tier and format it
		wanAmount := float64(amount) * math.Pow(2.5, float64(i))
		amounts[i] = fmt.Sprintf("%s Wans", strconv.FormatFloat(wanAmount, 'f', -1, 64))
		if amount == 1 {
			amounts[i] = strings.TrimSuffix(amounts[i], "s")
		}
		// Calculate the period for the next tier and format it
		period := *minutesFlag * int(math.Pow(3, float64(i)))
		periods[i] = fmt.Sprintf("%d mins", period)
		if period%60 == 0 {
			period /= 60
			periods[i] = fmt.Sprintf("%d hours", period)

			if period%24 == 0 {
				period /= 24
				periods[i] = fmt.Sprintf("%d days", period)
			}
		}
		if period == 1 {
			periods[i] = strings.TrimSuffix(periods[i], "s")
		}
	}
	// Load up and render the faucet website
	tmpl, err := Asset("faucet.html")
	if err != nil {
		log.Crit("Failed to load the faucet template", "err", err)
	}
	website := new(bytes.Buffer)
	err = template.Must(template.New("").Parse(string(tmpl))).Execute(website, map[string]interface{}{
		"Network":   *netnameFlag,
		"Amounts":   amounts,
		"Periods":   periods,
		"Recaptcha": *captchaToken,
		"NoAuth":    *noauthFlag,
	})

	if err != nil {
		log.Crit("Failed to render the faucet template", "err", err)
	}


	faucet := &faucet{}

	client,err := ethclient.Dial(url)
	if err != nil {
		log.Crit("Failed to render the faucet template", "err", err)
	}
	faucet.client = client


	faucet.timeouts = make(map[string]time.Time)
	faucet.index =  website.Bytes()
	faucet.eth = ethereum
	faucet.update = make(chan struct{}, 1)

	for _, wallet := range faucet.eth.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			baseAddr,err := ethereum.Etherbase()
			if err != nil {
				return
			}

			if account.Address == baseAddr {
				faucet.account = account
				faucet.wallet = wallet
				break
			}
		}
	}


	if err := faucet.listenAndServe(*apiPortFlag); err != nil {
		log.Crit("Failed to launch faucet API", "err", err)
	}

	fmt.Println("launched server...")
}

// request represents an accepted funding request.
type request struct {
	Avatar  string             `json:"avatar"`  // Avatar URL to make the UI nicer
	Account common.Address     `json:"account"` // Ethereum address being funded
	Time    time.Time          `json:"time"`    // Timestamp when the request was accepted
	Tx      *types.Transaction `json:"tx"`      // Transaction funding the account
}

// faucet represents a crypto faucet backed by an Ethereum light client.
type faucet struct {

	index  []byte              // Index page to serve up on the web

	conns    []*websocket.Conn    // Currently live websocket connections
	timeouts map[string]time.Time // History of users and their funding timeouts
	reqs     []*request           // Currently pending funding requests
	update   chan struct{}        // Channel to signal request updates

	lock sync.RWMutex // Lock protecting the faucet's internals

	client    *ethclient.Client   // Client connection to the Ethereum chain
	eth		  *eth.Ethereum

	//keystore *keystore.KeyStore
	wallet  accounts.Wallet

	account  accounts.Account   // Account funding user faucet requests

	nonce    uint64             // Current pending nonce of the faucet
	price    *big.Int           // Current gas price to issue funds with

}

// listenAndServe registers the HTTP handlers for the faucet and boots it up
// for service user funding requests.
func (f *faucet) listenAndServe(port int) error {
	go f.loop()

	http.HandleFunc("/", f.webHandler)
	http.Handle("/api", websocket.Handler(f.apiHandler))

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// webHandler handles all non-api requests, simply flattening and returning the
// faucet website.
func (f *faucet) webHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(f.index)
}

// apiHandler handles requests for Ether grants and transaction statuses.
func (f *faucet) apiHandler(conn *websocket.Conn) {
	// Start tracking the connection and drop at the end
	defer conn.Close()

	f.lock.Lock()
	f.conns = append(f.conns, conn)
	f.lock.Unlock()

	defer func() {
		//f.lock.Lock()
		for i, c := range f.conns {
			if c == conn {
				f.conns = append(f.conns[:i], f.conns[i+1:]...)
				break
			}
		}
		//f.lock.Unlock()
	}()
	// Gather the initial stats from the network to report
	var (
		balance *big.Int
		nonce   uint64
		err     error
		price   *big.Int
	)

	curblk := f.eth.BlockChain().CurrentBlock()

	for {

		statdb,err := f.eth.BlockChain().State()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		ethbase,err := f.eth.Etherbase()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		balance = statdb.GetBalance(ethbase)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		nonce = statdb.GetNonce(ethbase)

		ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)

		price, err = f.eth.ApiBackend.SuggestPrice(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		cancel()

		// If stats retrieval failed, wait a bit and retry
		if err != nil {
			if err = sendError(conn, errors.New("Faucet offline: "+err.Error())); err != nil {
				log.Warn("Failed to send faucet error to client", "err", err)
				return
			}
			time.Sleep(3 * time.Second)
			continue
		}
		// Initial stats reported successfully, proceed with user interaction
		break
	}

	// Send over the initial stats and the latest header
	if err = send(conn, map[string]interface{}{
		"funds":    balance.Div(balance, ether),
		"funded":   nonce,
		"peers":    1,
		"requests": f.reqs,
	}, 3*time.Second); err != nil {
		log.Warn("Failed to send initial stats to client", "err", err)
		return
	}

	if err = send(conn,curblk.Header(), 3*time.Second); err != nil {
		log.Warn("Failed to send initial header to client", "err", err)
		return
	}

	// Keep reading requests from the websocket until the connection breaks
	for {
		// Fetch the next funding request and validate against github
		var msg struct {
			URL     string `json:"url"`
			Tier    uint   `json:"tier"`
			Captcha string `json:"captcha"`
		}

		if err = websocket.JSON.Receive(conn, &msg); err != nil {
			return
		}
		if !*noauthFlag && !strings.HasPrefix(msg.URL, "https://gist.github.com/") && !strings.HasPrefix(msg.URL, "https://twitter.com/") &&
			!strings.HasPrefix(msg.URL, "https://plus.google.com/") && !strings.HasPrefix(msg.URL, "https://www.facebook.com/") {
			if err = sendError(conn, errors.New("URL doesn't link to supported services")); err != nil {
				log.Warn("Failed to send URL error to client", "err", err)
				return
			}
			continue
		}

		if msg.Tier >= uint(*tiersFlag) {
			if err = sendError(conn, errors.New("Invalid funding tier requested")); err != nil {
				log.Warn("Failed to send tier error to client", "err", err)
				return
			}
			continue
		}

		log.Info("Faucet funds requested", "url", msg.URL, "tier", msg.Tier)

		// If captcha verifications are enabled, make sure we're not dealing with a robot
		if *captchaToken != "" {
			form := url.Values{}
			form.Add("secret", *captchaSecret)
			form.Add("response", msg.Captcha)

			res, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", form)
			if err != nil {
				if err = sendError(conn, err); err != nil {
					log.Warn("Failed to send captcha post error to client", "err", err)
					return
				}
				continue
			}
			var result struct {
				Success bool            `json:"success"`
				Errors  json.RawMessage `json:"error-codes"`
			}
			err = json.NewDecoder(res.Body).Decode(&result)
			res.Body.Close()
			if err != nil {
				if err = sendError(conn, err); err != nil {
					log.Warn("Failed to send captcha decode error to client", "err", err)
					return
				}
				continue
			}
			if !result.Success {
				log.Warn("Captcha verification failed", "err", string(result.Errors))
				if err = sendError(conn, errors.New("Beep-bop, you're a robot!")); err != nil {
					log.Warn("Failed to send captcha failure to client", "err", err)
					return
				}
				continue
			}
		}
		// Retrieve the Ethereum address to fund, the requesting user and a profile picture
		var (
			username string
			avatar   string
			address  common.Address
		)
		switch {
		case strings.HasPrefix(msg.URL, "https://gist.github.com/"):
			if err = sendError(conn, errors.New("GitHub authentication discontinued at the official request of GitHub")); err != nil {
				log.Warn("Failed to send GitHub deprecation to client", "err", err)
				return
			}
			continue
		case strings.HasPrefix(msg.URL, "https://twitter.com/"):
			username, avatar, address, err = authTwitter(msg.URL)
		case strings.HasPrefix(msg.URL, "https://plus.google.com/"):
			username, avatar, address, err = authGooglePlus(msg.URL)
		case strings.HasPrefix(msg.URL, "https://www.facebook.com/"):
			username, avatar, address, err = authFacebook(msg.URL)
		case *noauthFlag:
			username, avatar, address, err = authNoAuth(msg.URL)
		default:
			err = errors.New("Something funky happened, please open an issue at https://github.com/ethereum/go-ethereum/issues")
		}
		if err != nil {
			if err = sendError(conn, err); err != nil {
				log.Warn("Failed to send prefix error to client", "err", err)
				return
			}
			continue
		}
		log.Info("Faucet request valid", "url", msg.URL, "tier", msg.Tier, "user", username, "address", address)

		// Ensure the user didn't request funds too recently
		f.lock.Lock()
		var (
			fund    bool
			timeout time.Time
		)
		if timeout = f.timeouts[username]; time.Now().After(timeout) {

			statdb,err := f.eth.BlockChain().State()
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}

			nonce = statdb.GetNonce(f.account.Address)

			//for test,remove later
			//address := common.HexToAddress("0x3ecb7c39cf5d7b885b3e3f9704f6bb20a35db077")

			// User wasn't funded recently, create the funding transaction
			amount := new(big.Int).Mul(big.NewInt(int64(*payoutFlag)), ether)
			amount = new(big.Int).Mul(amount, new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(msg.Tier)), nil))
			amount = new(big.Int).Div(amount, new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(msg.Tier)), nil))

			tx := types.NewTransaction(nonce, address, amount, big.NewInt(21000), price, nil)

			signed, err := f.wallet.SignTx(f.account, tx, f.eth.BlockChain().Config().ChainId)
			if err != nil {
				f.lock.Unlock()
				if err = sendError(conn, err); err != nil {
					log.Warn("Failed to send transaction creation error to client", "err", err)
					return
				}
				continue
			}


			// Submit the transaction and mark as funded if successful
			if err := f.SendTransaction(signed); err != nil {
				f.lock.Unlock()
				if err = sendError(conn, err); err != nil {
					log.Warn("Failed to send transaction transmission error to client", "err", err)
					return
				}
				continue
			}

			f.reqs = append(f.reqs, &request{
				Avatar:  avatar,
				Account: address,
				Time:    time.Now(),
				Tx:      signed,
			})

			f.timeouts[username] = time.Now().Add(time.Duration(*minutesFlag*int(math.Pow(3, float64(msg.Tier)))) * time.Minute)
			fund = true
		}
		f.lock.Unlock()

		// Send an error if too frequent funding, othewise a success
		if !fund {
			if err = sendError(conn, fmt.Errorf("%s left until next allowance", common.PrettyDuration(timeout.Sub(time.Now())))); err != nil { // nolint: gosimple
				log.Warn("Failed to send funding error to client", "err", err)
				return
			}
			continue
		}

		if err = sendSuccess(conn, fmt.Sprintf("Funding request accepted for %s into %s", username, address.Hex())); err != nil {
			log.Warn("Failed to send funding success to client", "err", err)
			return
		}
		select {
			case f.update <- struct{}{}:
			default:
		}
	}
}

// loop keeps waiting for interesting events and pushes them out to connected
// websockets.
func (f *faucet) loop() {
	// Wait for chain events and push them to clients
	heads := make(chan core.ChainHeadEvent, 16)
	sub := f.eth.ApiBackend.SubscribeChainHeadEvent(heads)
	defer sub.Unsubscribe()

	for {
		select {
		case blk := <-heads:
			// New chain head arrived, query the current stats and stream to clients
			var (
				balance *big.Int
				nonce   uint64
				price   *big.Int
				err     error
			)
			ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)

			statdb,err := f.eth.BlockChain().State()
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}

			balance = statdb.GetBalance(f.account.Address)

			if err == nil {
				nonce = statdb.GetNonce(f.account.Address)
				if err == nil {
					price, err = f.eth.ApiBackend.SuggestPrice(ctx)
				}
			}
			cancel()

			// If querying the data failed, try for the next block
			if err != nil {
				log.Warn("Failed to update faucet state", "block", blk.Block.Number, "hash", blk.Block.Hash(), "err", err)
				continue
			} else {
				log.Info("Updated faucet state", "block",  blk.Block.Number, "hash",  blk.Block.Hash(), "balance", balance, "nonce", nonce, "price", price)
			}
			// Faucet state retrieved, update locally and send to clients
			balance = new(big.Int).Div(balance, ether)

			f.lock.Lock()
			f.price, f.nonce = price, nonce
			for len(f.reqs) > 0 && f.reqs[0].Tx.Nonce() < f.nonce {
				f.reqs = f.reqs[1:]
			}
			f.lock.Unlock()

			f.lock.RLock()
			for _, conn := range f.conns {
				if err := send(conn, map[string]interface{}{
					"funds":    balance,
					"funded":   f.nonce,
					"peers":    1,//f.stack.Server().PeerCount(),
					"requests": f.reqs,
				}, time.Second); err != nil {
					log.Warn("Failed to send stats to client", "err", err)
					conn.Close()
					continue
				}
				if err := send(conn, blk.Block.Header(), 10*time.Second); err != nil {
					log.Warn("Failed to send header to client", "err", err)
					conn.Close()
				}
			}
			f.lock.RUnlock()

		case <-f.update:
			// Pending requests updated, stream to clients
			f.lock.RLock()
			for _, conn := range f.conns {
				if err := send(conn, map[string]interface{}{"requests": f.reqs}, time.Second); err != nil {
					log.Warn("Failed to send requests to client", "err", err)
					conn.Close()
				}
			}
			f.lock.RUnlock()
		}
	}
}

// sends transmits a data packet to the remote end of the websocket, but also
// setting a write deadline to prevent waiting forever on the node.
func send(conn *websocket.Conn, value interface{}, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 6 * time.Second
	}
	conn.SetWriteDeadline(time.Now().Add(timeout))
	return websocket.JSON.Send(conn, value)
}

// sendError transmits an error to the remote end of the websocket, also setting
// the write deadline to 1 second to prevent waiting forever.
func sendError(conn *websocket.Conn, err error) error {
	return send(conn, map[string]string{"error": err.Error()}, time.Second)
}

// sendSuccess transmits a success message to the remote end of the websocket, also
// setting the write deadline to 1 second to prevent waiting forever.
func sendSuccess(conn *websocket.Conn, msg string) error {
	return send(conn, map[string]string{"success": msg}, time.Second)
}

// authGitHub tries to authenticate a faucet request using GitHub gists, returning
// the username, avatar URL and Ethereum address to fund on success.
func authGitHub(url string) (string, string, common.Address, error) {
	// Retrieve the gist from the GitHub Gist APIs
	parts := strings.Split(url, "/")
	req, _ := http.NewRequest("GET", "https://api.github.com/gists/"+parts[len(parts)-1], nil)
	if *githubUser != "" {
		req.SetBasicAuth(*githubUser, *githubToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", common.Address{}, err
	}
	var gist struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}
	err = json.NewDecoder(res.Body).Decode(&gist)
	res.Body.Close()
	if err != nil {
		return "", "", common.Address{}, err
	}
	if gist.Owner.Login == "" {
		return "", "", common.Address{}, errors.New("Anonymous Gists not allowed")
	}
	// Iterate over all the files and look for Ethereum addresses
	var address common.Address
	for _, file := range gist.Files {
		content := strings.TrimSpace(file.Content)
		if len(content) == 2+common.AddressLength*2 {
			address = common.HexToAddress(content)
		}
	}
	if address == (common.Address{}) {
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	// Validate the user's existence since the API is unhelpful here
	if res, err = http.Head("https://github.com/" + gist.Owner.Login); err != nil {
		return "", "", common.Address{}, err
	}
	res.Body.Close()

	if res.StatusCode != 200 {
		return "", "", common.Address{}, errors.New("Invalid user... boom!")
	}
	// Everything passed validation, return the gathered infos
	return gist.Owner.Login + "@github", fmt.Sprintf("https://github.com/%s.png?size=64", gist.Owner.Login), address, nil
}

// authTwitter tries to authenticate a faucet request using Twitter posts, returning
// the username, avatar URL and Ethereum address to fund on success.
func authTwitter(url string) (string, string, common.Address, error) {
	// Ensure the user specified a meaningful URL, no fancy nonsense
	parts := strings.Split(url, "/")
	if len(parts) < 4 || parts[len(parts)-2] != "status" {
		return "", "", common.Address{}, errors.New("Invalid Twitter status URL")
	}
	username := parts[len(parts)-3]

	// Twitter's API isn't really friendly with direct links. Still, we don't
	// want to do ask read permissions from users, so just load the public posts and
	// scrape it for the Ethereum address and profile URL.
	res, err := http.Get(url)
	if err != nil {
		return "", "", common.Address{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", common.Address{}, err
	}
	address := common.HexToAddress(string(regexp.MustCompile("0x[0-9a-fA-F]{40}").Find(body)))
	if address == (common.Address{}) {
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	var avatar string
	if parts = regexp.MustCompile("src=\"([^\"]+twimg.com/profile_images[^\"]+)\"").FindStringSubmatch(string(body)); len(parts) == 2 {
		avatar = parts[1]
	}
	return username + "@twitter", avatar, address, nil
}

// authGooglePlus tries to authenticate a faucet request using GooglePlus posts,
// returning the username, avatar URL and Ethereum address to fund on success.
func authGooglePlus(url string) (string, string, common.Address, error) {
	// Ensure the user specified a meaningful URL, no fancy nonsense
	parts := strings.Split(url, "/")
	if len(parts) < 4 || parts[len(parts)-2] != "posts" {
		return "", "", common.Address{}, errors.New("Invalid Google+ post URL")
	}
	username := parts[len(parts)-3]

	// Google's API isn't really friendly with direct links. Still, we don't
	// want to do ask read permissions from users, so just load the public posts and
	// scrape it for the Ethereum address and profile URL.
	res, err := http.Get(url)
	if err != nil {
		return "", "", common.Address{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", common.Address{}, err
	}
	address := common.HexToAddress(string(regexp.MustCompile("0x[0-9a-fA-F]{40}").Find(body)))
	if address == (common.Address{}) {
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	var avatar string
	if parts = regexp.MustCompile("src=\"([^\"]+googleusercontent.com[^\"]+photo.jpg)\"").FindStringSubmatch(string(body)); len(parts) == 2 {
		avatar = parts[1]
	}
	return username + "@google+", avatar, address, nil
}

// authFacebook tries to authenticate a faucet request using Facebook posts,
// returning the username, avatar URL and Ethereum address to fund on success.
func authFacebook(url string) (string, string, common.Address, error) {
	// Ensure the user specified a meaningful URL, no fancy nonsense
	parts := strings.Split(url, "/")
	if len(parts) < 4 || parts[len(parts)-2] != "posts" {
		return "", "", common.Address{}, errors.New("Invalid Facebook post URL")
	}
	username := parts[len(parts)-3]

	// Facebook's Graph API isn't really friendly with direct links. Still, we don't
	// want to do ask read permissions from users, so just load the public posts and
	// scrape it for the Ethereum address and profile URL.
	res, err := http.Get(url)
	if err != nil {
		return "", "", common.Address{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", common.Address{}, err
	}
	address := common.HexToAddress(string(regexp.MustCompile("0x[0-9a-fA-F]{40}").Find(body)))
	if address == (common.Address{}) {
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	var avatar string
	if parts = regexp.MustCompile("src=\"([^\"]+fbcdn.net[^\"]+)\"").FindStringSubmatch(string(body)); len(parts) == 2 {
		avatar = parts[1]
	}
	return username + "@facebook", avatar, address, nil
}

// authNoAuth tries to interpret a faucet request as a plain Ethereum address,
// without actually performing any remote authentication. This mode is prone to
// Byzantine attack, so only ever use for truly private networks.
func authNoAuth(url string) (string, string, common.Address, error) {
	address := common.HexToAddress(regexp.MustCompile("0x[0-9a-fA-F]{40}").FindString(url))
	if address == (common.Address{}) {
		return "", "", common.Address{}, errors.New("No Ethereum address found to fund")
	}
	return address.Hex() + "@noauth", "", address, nil
}



func test() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello world`))
	})
	err := http.ListenAndServe(":8080", nil) // <-今天讲的就是这个ListenAndServe是如何工作的
	fmt.Println(err)

}
