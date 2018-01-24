package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spikeekips/stellar-utils/boslib"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/keypair"
)

var log *logrus.Logger

var flags *flag.FlagSet
var flagVerbose bool
var flagHorizon string
var flagNetworkPassphrase string
var flagSecretSeed string
var flagCSVFile string
var flagBalance string
var flagAccountPublicAddress string
var flagAccountSecretSeed string
var flagFee uint64
var isRandom bool
var accountPublicAddress string
var accountData [][2]string // {{<public address>, <balance>}}

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

func init() {
	log = logrus.New()
	log.Level = logrus.ErrorLevel

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options] <secret seed> <balance> [<account's public address>]")
		flags.PrintDefaults()
	}

	flags.StringVar(&flagHorizon, "horizon", "", "horizon server address")
	flags.BoolVar(&flagVerbose, "verbose", false, "verbose")
	flags.Uint64Var(&flagFee, "fee", boslib.DefaultFee, "transaction fee")
	flags.StringVar(&flagCSVFile, "csv", "", "account csv file")

	flags.Parse(os.Args[1:])

	if flagVerbose {
		log.Level = logrus.DebugLevel
	}

	log.Debugf("arguments: %v", os.Args)

	if len(flagCSVFile) < 1 && flags.NArg() < 2 {
		usage(fmt.Errorf("insufficient arguments"))
	}

	flagSecretSeed = strings.TrimSpace(flags.Arg(0))
	flagAccountPublicAddress = strings.TrimSpace(flags.Arg(2))

	log.Debugf("given              flagHorizon: %T:%4d: %v", flagHorizon, utf8.RuneCountInString(flagHorizon), flagHorizon)
	log.Debugf("given           flagSecretSeed: %T:%4d: %v", flagSecretSeed, utf8.RuneCountInString(flagSecretSeed), flagSecretSeed)
	log.Debugf("given               flagCSVFile: %T:%4d: %v",
		flagCSVFile, utf8.RuneCountInString(fmt.Sprintf("%v", flagCSVFile)), flagCSVFile)
	log.Debugf("given               flagBalance: %T:%4d: %v",
		flagBalance, utf8.RuneCountInString(fmt.Sprintf("%v", flagBalance)), flagBalance)
	log.Debugf("given flagAccountPublicAddress: %T:%4d: %v",
		flagAccountPublicAddress, utf8.RuneCountInString(flagAccountPublicAddress), flagAccountPublicAddress)
	log.Debugf("given                  flagFee: %T:%4d: %v",
		flagFee, utf8.RuneCountInString(fmt.Sprintf("%v", flagFee)), flagFee)

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

		// set flagNetworkPassphrase
		if network_passphrase, found := apiSkel["network_passphrase"]; !found {
			usage(fmt.Errorf("wrong horizon, '%s': 'network_passphrase' is missing in response", flagHorizon))
		} else {
			flagNetworkPassphrase = network_passphrase.(string)
		}

		if err != nil {
			usage(fmt.Errorf("wrong horizon, '%s': %v", flagHorizon, err))
		}
	}

	{
		if len(flagCSVFile) > 0 {
			var err error
			var f *os.File
			if f, err = os.Open(flagCSVFile); err != nil {
				log.Debugf("'%s' is not account csv file", flagCSVFile)
			} else {
				bf := bufio.NewReader(f)
				line := 0
				var l string
				for {
					s, isPrefix, err := bf.ReadLine()
					if err != nil {
						break
					}
					if isPrefix {
						l = string(s)
						continue

					}
					line += 1
					l = string(s)

					a := strings.Split(l, ",")
					if len(a) < 2 {
						continue
					}
					_, err = strconv.ParseFloat(strings.TrimSpace(a[1]), 64)
					if err != nil {
						log.Errorf("invalid account data; `balance` must be integer at line, %d, '%s'", line, a[1])
						continue
					}

					address := strings.TrimSpace(a[0])
					// check duplciation
					for _, a := range accountData {
						if a[0] == address {
							usage(fmt.Errorf("<account's public address>, '%s' is duplicated at line, %d", address, line))
						}
					}

					if exists, err := checkAddressExists(flagHorizon, address); err != nil {
						usage(err)
					} else if exists {
						err = fmt.Errorf(
							"account, '%s' is already registered in horizon, '%s' at line, %d",
							address,
							flagHorizon,
							line,
						)
						usage(err)
					}

					if _, err = keypair.Parse(address); err != nil {
						usage(fmt.Errorf("invalid <account's public address>, '%s' at line, %d: %v", address, line, err))
					}
					accountData = append(
						accountData,
						[2]string{
							address,
							strings.TrimSpace(a[1]),
						},
					)
				}
			}
		}
	}

	// amount
	{
		if len(accountData) < 1 {
			flagBalance = flags.Arg(1)

			var err error
			var balance float64
			balance, err = strconv.ParseFloat(flagBalance, 64)
			if err != nil {
				usage(err)
			}
			if balance < boslib.MinimumBalance {
				usage(fmt.Errorf("<amount> must be higher than %0d", boslib.MinimumBalance))
			}
		}
	}

	// secret seed
	var secretSeedKP keypair.KP
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
				"invalid <secret seed>, '%s'; the account was not found in network",
				flagSecretSeed,
			))
		}
	}

	// account's public key
	{
		if len(accountData) < 1 {
			var err error
			if len(flagAccountPublicAddress) < 1 {
				log.Debugf("empty <account's public address> was given, so the keypair will be generated by random rules")
				kp, _ := keypair.Random()
				flagAccountSecretSeed = kp.Seed()
				flagAccountPublicAddress = kp.Address()

				isRandom = true
			} else {
				var receiverKP keypair.KP
				if receiverKP, err = keypair.Parse(flagAccountPublicAddress); err != nil {
					usage(fmt.Errorf("invalid <account's public address>: %v", err))
				}

				if exists, err := checkAddressExists(flagHorizon, receiverKP.Address()); err != nil {
					usage(err)
				} else if exists {
					err = fmt.Errorf(
						"account, '%s' is already registered in horizon, '%s'",
						receiverKP.Address(), flagHorizon,
					)
					usage(err)
				}
			}

			accountData = append(
				accountData,
				[2]string{
					flagAccountPublicAddress,
					flagBalance,
				},
			)
		}
	}

	log.Debugf("parsed              flagHorizon: %T:%4d: %v", flagHorizon, utf8.RuneCountInString(flagHorizon), flagHorizon)
	log.Debugf("parsed    flagNetworkPassphrase: %T:%4d: %v",
		flagNetworkPassphrase, utf8.RuneCountInString(flagNetworkPassphrase), flagNetworkPassphrase)
	log.Debugf("parsed           flagSecretSeed: %T:%4d: %v",
		flagSecretSeed, utf8.RuneCountInString(flagSecretSeed), flagSecretSeed)
	log.Debugf("parsed               flagBalance: %T:%4d: %v",
		flagBalance, utf8.RuneCountInString(fmt.Sprintf("%v", flagBalance)), flagBalance)
	log.Debugf("parsed flagAccountPublicAddress: %T:%4d: %v",
		flagAccountPublicAddress, utf8.RuneCountInString(flagAccountPublicAddress), flagAccountPublicAddress)
	log.Debugf("parsed    flagAccountSecretSeed: %T:%4d: %v",
		flagAccountSecretSeed, utf8.RuneCountInString(fmt.Sprintf("%v", flagAccountSecretSeed)), flagAccountSecretSeed)

	log.Debug("We will create new account:")
	for _, a := range accountData {
		b, _ := strconv.ParseFloat(a[1], 64)
		log.Debugf("'%s', '%10.7f'", a[0], b)
	}
}

func main() {
	for _, a := range accountData {
		address, balance := a[0], a[1]
		resp, err := boslib.CreateAccount(
			flagHorizon,
			flagSecretSeed,
			address,
			balance,
			flagNetworkPassphrase,
			0,
			flagFee,
		)
		if err != nil {
			fmt.Printf("(X) Failed to create account, '%s', '%s': %v\n", address, balance, err)
			os.Exit(1)
		}
		log.Debugf("transaction posted in ledger: %v", resp.Ledger)

		t := template.Must(template.New("").Parse(strings.TrimSpace(`
(O) Successfully {{ if .isRandom }}new {{ end }}account is created. {{ .address }}{{ if .isRandom }}({{ .seed }}){{ end }}, {{ .balance }}`) + "\n"))
		t.Execute(os.Stdout, map[string]interface{}{
			"address":  address,
			"seed":     flagAccountSecretSeed,
			"isRandom": isRandom,
			"balance":  balance,
		})
	}
}
