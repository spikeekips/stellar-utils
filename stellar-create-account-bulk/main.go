package main

import (
	"encoding/csv"
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

	"github.com/spikeekips/stellar-utils/boslib"
	"github.com/stellar/go/keypair"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger
var flags *flag.FlagSet

var flagHorizon string
var flagNetworkPassphrase string
var flagVerbose bool
var flagSecretSeed string
var flagCSVFileName string
var flagFee uint64

func usage(err error) {
	if err != nil {
		log.Error(err)
	}
	flags.Usage()
	os.Exit(1)
}

var balanceData [][]boslib.Balance

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
	log.Level = logrus.InfoLevel
	boslib.SetLevel(log.Level)

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options] <secret seed> <csv>")
		flags.PrintDefaults()
	}
	flags.BoolVar(&flagVerbose, "verbose", false, "verbose")
	flags.StringVar(&flagHorizon, "horizon", "", "horizon server address")
	flags.Uint64Var(&flagFee, "fee", boslib.DefaultFee, "transaction fee")

	flags.Parse(os.Args[1:])
	if flagVerbose {
		log.Level = logrus.DebugLevel
		boslib.SetLevel(log.Level)
	}

	if flags.NArg() < 2 {
		usage(fmt.Errorf("insufficient arguments"))
	}

	flagSecretSeed = strings.TrimSpace(flags.Arg(0))

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

	flagCSVFileName = strings.TrimSpace(flags.Arg(1))
	{
		var err error
		var f *os.File

		if f, err = os.Open(flagCSVFileName); err != nil {
			usage(err)
		}

		var csvContent [][]string
		if csvContent, err = csv.NewReader(f).ReadAll(); err != nil {
			usage(err)
		}

		var bs []boslib.Balance
		for n, b := range csvContent {
			address := b[0]
			if _, err = strconv.ParseFloat(b[1], 64); err != nil {
				usage(err)
			}

			balance := b[1]

			if n != 0 && n%100 == 0 {
				balanceData = append(balanceData, bs)
				bs = []boslib.Balance{}
			}

			bs = append(bs, boslib.Balance{Amount: balance, Address: address})
		}

		if len(bs) > 0 {
			balanceData = append(balanceData, bs)
		}
	}
}

func main() {
	for _, b := range balanceData {
		_, err := boslib.CreateAccounts(
			flagHorizon,
			flagSecretSeed,
			flagNetworkPassphrase,
			0,
			flagFee,
			b...,
		)

		for _, i := range b {
			t := template.Must(template.New("").Parse("({{ if .err }}X{{ else }}O{{ end }}) {{ .b.Address }} : {{ .b.Amount }}\n"))
			t.Execute(os.Stdout, map[string]interface{}{
				"b":   i,
				"err": err != nil,
			})
		}
		if err != nil {
			log.Error(err)
		}
	}
}
