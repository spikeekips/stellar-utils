package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spikeekips/stellar-utils/boslib"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/keypair"
)

var log *logrus.Logger
var flags *flag.FlagSet

var flagHorizon string
var flagVerbose bool

var addresses []keypair.KP

func usage(err error) {
	if err != nil {
		log.Error(err)
	}
	flags.Usage()
	os.Exit(1)
}

func init() {
	log = logrus.New()
	log.Level = logrus.InfoLevel
	boslib.SetLevel(log.Level)

	flags = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Println(filepath.Base(os.Args[0]), "[options] <public address>")
		flags.PrintDefaults()
	}
	flags.BoolVar(&flagVerbose, "verbose", false, "verbose")
	flags.StringVar(&flagHorizon, "horizon", "", "horizon server address")

	flags.Parse(os.Args[1:])
	if flagVerbose {
		log.Level = logrus.DebugLevel
		boslib.SetLevel(log.Level)
	}

	if flags.NArg() < 1 {
		usage(errors.New("<public address> is missing"))
	}

	{
		var invalidAddress []string
		for _, a := range flags.Args() {
			var err error
			var address keypair.KP
			if address, err = keypair.Parse(a); err != nil {
				invalidAddress = append(invalidAddress, a)
				continue
			}

			addresses = append(addresses, address)
		}

		if len(invalidAddress) > 0 {
			usage(fmt.Errorf("found invalid public address or secret seed: %s", strings.Join(invalidAddress, ", ")))
		}
	}
}

func main() {
	for _, address := range addresses {
		u, _ := url.Parse(flagHorizon)
		u.Path = path.Join(u.Path, "accounts", address.Address())
		response, err := http.Get(u.String())
		if err != nil {
			usage(fmt.Errorf("failed to connect to horizon, '%s': %v", u.String(), err))
		}

		if response.StatusCode == 404 {
			log.Errorf("account, '%s' does not exist", address.Address())
			continue
		}

		if response.StatusCode != 200 {
			log.Errorf("failed to get response from horizon, '%s': %v", flagHorizon, response.StatusCode)
			continue
		}

		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		var skel map[string]interface{}
		json.Unmarshal(body, &skel)

		s, _ := json.MarshalIndent(skel, "", "  ")
		fmt.Println(string(s))
	}
}
