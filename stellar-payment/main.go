package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/spikeekips/stellar-utils/boslib"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/keypair"
)

type Balance struct {
	Amount    float64
	AssetType string
}

var log *logrus.Logger

var flags *flag.FlagSet
var flagVerbose bool
var flagHorizon string
var flagSecretSeed string
var flagReceiverAddress string
var flagAmount float64
var flagFee uint64

var networkPassphrase string
var secretSeedKP keypair.KP
var receiverKP keypair.KP
var sender_balances_before []Balance
var receiver_balances_before []Balance

func usage(err error) {
	if err != nil {
		log.Error(err)
	}
	flags.Usage()
	os.Exit(1)
}

func checkAddressExists(horizonUrl, address string) (exists bool, err error) {
	exists = false

	// check whether account exists or not
	u, _ := url.Parse(horizonUrl)
	u.Path = path.Join(u.Path, "accounts", address)
	response, err := http.Get(u.String())
	if err != nil {
		err = fmt.Errorf("failed to connect to horizon, '%s': %v", u.String(), err)
		return
	}

	return response.StatusCode == 200, nil
}

func checkAccountBalanceInfo(horizonUrl, address string) (balances []Balance, err error) {
	// check whether account exists or not
	u, _ := url.Parse(horizonUrl)
	u.Path = path.Join(u.Path, "accounts", address)
	response, err := http.Get(u.String())
	if err != nil {
		err = fmt.Errorf("failed to connect to horizon, '%s': %v", u.String(), err)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var info map[string]interface{}
	json.Unmarshal(body, &info)

	var o []interface{}
	var result interface{}
	var ok bool
	if result, ok = info["balances"]; !ok {
		err = fmt.Errorf("invalid `balances` received: %v", info)
		return
	} else if o, ok = result.([]interface{}); !ok {
		err = fmt.Errorf("invalid `balances` items received: %v", result)
		return
	}

	for _, bt := range o {
		var balance float64
		var assetType string

		var bta map[string]interface{}
		if bta, ok = bt.(map[string]interface{}); !ok {
			err = fmt.Errorf("invalid item received: %v", bt)
			return
		}

		var f float64
		if f, err = strconv.ParseFloat(bta["balance"].(string), 64); err != nil {
			err = fmt.Errorf("invalid `balance` received: %v", bta)
			return
		} else {
			balance = f
		}

		if assetType, ok = bta["asset_type"].(string); !ok {
			err = fmt.Errorf("invalid `asset_type` received: %v", bta)
			return
		}

		balances = append(balances, Balance{Amount: balance, AssetType: assetType})
	}

	if len(balances) < 1 {
		err = fmt.Errorf("empty balance: %v", body)
		return
	}

	return balances, nil
}

func init() {
	log = logrus.New()
	log.Level = logrus.ErrorLevel

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options] <sender's secret seed> <receiver's public address> <amount>")
		flags.PrintDefaults()
	}

	flags.StringVar(&flagHorizon, "horizon", "", "horizon server address")
	flags.BoolVar(&flagVerbose, "verbose", false, "verbose")
	flags.Uint64Var(&flagFee, "fee", boslib.DefaultFee, "transaction fee")

	flags.Parse(os.Args[1:])

	if flagVerbose {
		log.Level = logrus.DebugLevel
	}

	log.Debugf("arguments: %v", os.Args)

	if flags.NArg() < 3 {
		usage(fmt.Errorf("insufficient arguments"))
	}

	flagSecretSeed = strings.TrimSpace(flags.Arg(0))
	flagReceiverAddress = strings.TrimSpace(flags.Arg(1))

	log.Debugf("given              flagHorizon: %T:%4d: %v", flagHorizon, utf8.RuneCountInString(flagHorizon), flagHorizon)
	log.Debugf("given           flagSecretSeed: %T:%4d: %v", flagSecretSeed, utf8.RuneCountInString(flagSecretSeed), flagSecretSeed)
	log.Debugf("given      flagRecieverAddress: %T:%4d: %v",
		flagReceiverAddress, utf8.RuneCountInString(flagReceiverAddress), flagReceiverAddress)
	log.Debugf("given               flagAmount: %T:%4d: %v",
		flagAmount, utf8.RuneCountInString(fmt.Sprintf("%v", flagAmount)), flagAmount)
	log.Debugf("given                  flagFee: %T:%4d: %v",
		flagFee, utf8.RuneCountInString(fmt.Sprintf("%v", flagFee)), flagFee)

	// amount
	{
		var err error
		flagAmount, err = strconv.ParseFloat(flags.Arg(2), 64)
		if err != nil {
			usage(err)
		}
	}

	// horizon
	{
		var err error

		flagHorizon = strings.TrimSpace(flagHorizon)
		if len(flagHorizon) < 1 {
			usage(fmt.Errorf("--horizon must be given"))
		}

		response, err := http.Get(flagHorizon)
		if err != nil {
			usage(fmt.Errorf("failed to connect to horizon, '%s': %v", flagHorizon, err))
		}
		if response.StatusCode != 200 {
			usage(fmt.Errorf("failed to connect to horizon, '%s': %v", flagHorizon, response.StatusCode))
		}

		if h, found := response.Header["Content-Type"]; !found {
			usage(fmt.Errorf("wrong horizon, '%s': 'Content-Type' is missing", flagHorizon))
		} else if strings.Split(h[0], ";")[0] != "application/hal+json" {
			usage(fmt.Errorf("wrong horizon, '%s': 'Content-Type' is not 'application/hal+json'", flagHorizon))
		}

		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		var apiSkel map[string]interface{}
		json.Unmarshal(body, &apiSkel)

		// set networkPassphrase
		if network_passphrase, found := apiSkel["network_passphrase"]; !found {
			usage(fmt.Errorf("wrong horizon, '%s': 'network_passphrase' is missing in response", flagHorizon))
		} else {
			networkPassphrase = network_passphrase.(string)
		}

		if err != nil {
			usage(fmt.Errorf("wrong horizon, '%s': %v", flagHorizon, err))
		}
	}

	// secret seed
	{
		var err error
		if secretSeedKP, err = keypair.Parse(flagSecretSeed); err != nil {
			usage(fmt.Errorf("invalid <secret seed>: %v", err))
		} else if string([]rune(flagSecretSeed)[0]) != "S" {
			usage(fmt.Errorf("not <secret seed>, this is public address"))
		}

		if exists, err := checkAddressExists(flagHorizon, secretSeedKP.Address()); err != nil {
			usage(err)
		} else if !exists {
			usage(fmt.Errorf(
				"invalid <secret seed>, '%s'; the account is not found in network",
				flagSecretSeed,
			))
		}
	}

	// account's public key
	{
		var err error
		if receiverKP, err = keypair.Parse(flagReceiverAddress); err != nil {
			usage(fmt.Errorf("malformed <receiver's public address>: %v", err))
		}

		if exists, err := checkAddressExists(flagHorizon, receiverKP.Address()); err != nil {
			usage(err)
		} else if !exists {
			err = fmt.Errorf(
				"invalid <receiver's address>, '%s'; the account is not found in network",
				flagReceiverAddress,
			)
			usage(err)
		}
	}

	// check accounts balance
	{
		var err error
		{
			if sender_balances_before, err = checkAccountBalanceInfo(flagHorizon, secretSeedKP.Address()); err != nil {
				usage(err)
			}
		}
		{
			if receiver_balances_before, err = checkAccountBalanceInfo(flagHorizon, receiverKP.Address()); err != nil {
				usage(err)
			}
		}
	}

	log.Debugf("parsed              flagHorizon: %T:%4d: %v", flagHorizon, utf8.RuneCountInString(flagHorizon), flagHorizon)
	log.Debugf("parsed    networkPassphrase: %T:%4d: %v",
		networkPassphrase, utf8.RuneCountInString(networkPassphrase), networkPassphrase)
	log.Debugf("parsed           flagSecretSeed: %T:%4d: %v",
		flagSecretSeed, utf8.RuneCountInString(flagSecretSeed), flagSecretSeed)
	log.Debugf("parsed               flagAmount: %T:%4d: %v",
		flagAmount, utf8.RuneCountInString(fmt.Sprintf("%v", flagAmount)), flagAmount)
	log.Debugf("parsed      flagRecieverAddress: %T:%4d: %v",
		flagReceiverAddress, utf8.RuneCountInString(flagReceiverAddress), flagReceiverAddress)
	log.Debugf("                sender balances: %20.7f", sender_balances_before[0].Amount)
	log.Debugf("              receiver balances: %20.7f", receiver_balances_before[0].Amount)
}

func main() {
	resp, err := boslib.SendPayment(
		flagHorizon,
		flagSecretSeed,
		receiverKP.Address(),
		fmt.Sprintf("%0.7f", flagAmount),
		networkPassphrase,
		0,
		flagFee,
	)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	log.Debugf("transaction posted in ledger: %v", resp.Ledger)

	var sender_balances_after []Balance
	var receiver_balances_after []Balance
	{
		var err error
		{
			if sender_balances_after, err = checkAccountBalanceInfo(flagHorizon, secretSeedKP.Address()); err != nil {
				usage(err)
			}
		}
		{
			if receiver_balances_after, err = checkAccountBalanceInfo(flagHorizon, receiverKP.Address()); err != nil {
				usage(err)
			}
		}
	}

	t := template.Must(template.New("").Parse(strings.TrimSpace(`
{{ .amount }} sent from {{ .from_address }} to {{ .to_address }} successfully

  sender: {{ .senderDiff }}: {{ .senderBefore }} -> {{ .senderAfter }}
receiver: {{ .receiverDiff }}: {{ .receiverBefore }} -> {{ .receiverAfter }}
		`) + "\n"))
	t.Execute(os.Stdout, map[string]interface{}{
		"to_address":     receiverKP.Address(),
		"from_address":   secretSeedKP.Address(),
		"amount":         fmt.Sprintf("%0.7f", flagAmount),
		"senderBefore":   fmt.Sprintf("%20.7f", sender_balances_before[0].Amount),
		"senderAfter":    fmt.Sprintf("%20.7f", sender_balances_after[0].Amount),
		"senderDiff":     fmt.Sprintf("%20.7f", sender_balances_before[0].Amount-sender_balances_after[0].Amount),
		"receiverBefore": fmt.Sprintf("%20.7f", receiver_balances_before[0].Amount),
		"receiverAfter":  fmt.Sprintf("%20.7f", receiver_balances_after[0].Amount),
		"receiverDiff":   fmt.Sprintf("%20.7f", receiver_balances_after[0].Amount-receiver_balances_before[0].Amount),
	})

	os.Exit(0)
}
