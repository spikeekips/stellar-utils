package boslib

import (
	"net/http"
	"time"

	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/xdr"
)

var DefaultFee uint64 = 10000    // default value of boscoin, stroop
var MinimumBalance float64 = 0.1 // minimum balance for account, XLM

type FixedSequence struct {
	Seq xdr.SequenceNumber
}

func (m FixedSequence) MutateTransaction(o *b.TransactionBuilder) error {
	o.TX.SeqNum = m.Seq + 1
	return nil
}

func MakeNetwork(horizonUrl string) *horizon.Client {
	return &horizon.Client{
		URL:  horizonUrl,
		HTTP: &http.Client{Timeout: 1000000 * time.Second},
	}

}

type SigningError struct {
	err error
}

func (s *SigningError) Error() string {
	return s.err.Error()
}
