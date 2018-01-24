package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/spikeekips/stellar-utils/boslib"

	"github.com/sirupsen/logrus"
	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
)

var log *logrus.Logger
var flags *flag.FlagSet

var flagHorizon string
var flagVerbose bool
var flagFee uint64

var secretSeed string
var networkPassphrase string

func usage(err error) {
	if err != nil {
		log.Error(err)
	}
	flags.Usage()
	os.Exit(1)
}

func init() {
	log = logrus.New()
	log.SetLevel(logrus.InfoLevel)

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options] <sender's secret seed> ")
		flags.PrintDefaults()
	}

	flags.StringVar(&flagHorizon, "horizon", "", "horizon server address")
	flags.BoolVar(&flagVerbose, "verbose", false, "verbose")
	flags.Uint64Var(&flagFee, "fee", boslib.DefaultFee, "transaction fee")

	log.Debugf("arguments: %v", os.Args)
	flags.Parse(os.Args[1:])

	if flagVerbose {
		log.Level = logrus.DebugLevel
	}

	if flags.NArg() < 1 {
		usage(fmt.Errorf("insufficient arguments"))
	}

	secretSeed = strings.TrimSpace(flags.Arg(0))

	log.Debugf("given              flagHorizon: %T:%4d: %v", flagHorizon, utf8.RuneCountInString(flagHorizon), flagHorizon)
	log.Debugf("given               secretSeed: %T:%4d: %v", secretSeed, utf8.RuneCountInString(secretSeed), secretSeed)
	log.Debugf("given                  flagFee: %T:%4d: %v", flagFee, utf8.RuneCountInString(fmt.Sprintf("%v", flagFee)), flagFee)

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
		var secretSeedKP keypair.KP
		if secretSeedKP, err = keypair.Parse(secretSeed); err != nil {
			usage(fmt.Errorf("invalid <secret seed>: %v", err))
		} else if string([]rune(secretSeed)[0]) != "S" {
			usage(fmt.Errorf("not <secret seed>, this is public address"))
		}

		if exists, err := boslib.CheckAddressExists(flagHorizon, secretSeedKP.Address()); err != nil {
			usage(err)
		} else if !exists {
			usage(fmt.Errorf(
				"invalid <secret seed>, '%s'; the account is not found in network",
				secretSeed,
			))
		}
	}
}

func main() {
	nc := boslib.MakeNetwork(flagHorizon)

	var err error

	tx, err := b.Transaction(
		b.SourceAccount{secretSeed},
		b.Network{Passphrase: networkPassphrase},
		b.AutoSequence{nc},
		b.BaseFee{Amount: flagFee},
		b.Inflation(),
	)
	if err != nil {
		fmt.Printf("failed to make tranaction: %s\n", err)
		os.Exit(1)
	}

	tx.NetworkPassphrase = networkPassphrase

	txe, err := tx.Sign(secretSeed)
	if err != nil {
		fmt.Printf("failed to sign: %s\n", err)
		os.Exit(1)
	}

	var txeB64 string
	if txeB64, err = txe.Base64(); err != nil {
		fmt.Printf("failed to txeB64: %s\n", err)
		os.Exit(1)
	}

	var resp horizon.TransactionSuccess
	if resp, err = nc.SubmitTransaction(txeB64); err != nil {
		log.Debugf("failed to SubmitTransaction: %s", err)
		os.Exit(1)
	}

	log.Debugf("inflation occurred: %v", resp.Ledger)

	os.Exit(0)
}
