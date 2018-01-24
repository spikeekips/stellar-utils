package boslib

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/xdr"
)

func CheckAddressExists(horizonUrl, address string) (exists bool, err error) {
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

func LoadSequenceForAccount(horizonUrl, senderAddress string) (xdr.SequenceNumber, error) {
	nc := MakeNetwork(horizonUrl)
	return nc.SequenceForAccount(senderAddress)
}

func CreateAccount(horizonUrl, senderSeed, receiverAddress, amount, networkPassphrase string, seq xdr.SequenceNumber, fee uint64) (
	resp horizon.TransactionSuccess,
	err error,
) {
	nc := MakeNetwork(horizonUrl)

	var sp b.TransactionMutator
	if seq < 1 {
		sp = b.AutoSequence{nc}
	} else {
		sp = FixedSequence{seq}
	}

	tx, err := b.Transaction(
		b.BaseFee{Amount: fee},
		b.SourceAccount{senderSeed},
		sp,
		b.CreateAccount(
			b.Destination{receiverAddress},
			b.NativeAmount{amount},
		),
	)
	if err != nil {
		return
	}

	tx.NetworkPassphrase = networkPassphrase

	txe, err := tx.Sign(senderSeed)
	if err != nil {
		return
	}

	var txeB64 string
	if txeB64, err = txe.Base64(); err != nil {
		err = &SigningError{err: err}
		return
	}

	if resp, err = nc.SubmitTransaction(txeB64); err != nil {
		return
	}
	log.Debugf("transaction, 'create-account' posted in ledger: %v", resp.Ledger)

	return
}

type Balance struct {
	ID      string
	Address string
	Amount  string
}

func CreateAccounts(horizonUrl, senderSeed, networkPassphrase string, seq xdr.SequenceNumber, fee uint64, receiverAddress ...Balance) (
	resp horizon.TransactionSuccess,
	err error,
) {
	nc := MakeNetwork(horizonUrl)

	var sp b.TransactionMutator
	if seq < 1 {
		sp = b.AutoSequence{nc}
	} else {
		sp = FixedSequence{seq}
	}

	muts := []b.TransactionMutator{
		b.BaseFee{Amount: fee},
		b.SourceAccount{senderSeed},
		sp,
	}

	for _, i := range receiverAddress {
		muts = append(
			muts,
			b.CreateAccount(
				b.Destination{i.Address},
				b.NativeAmount{i.Amount},
			),
		)
	}

	tx, err := b.Transaction(muts...)
	if err != nil {
		return
	}

	tx.NetworkPassphrase = networkPassphrase

	txe, err := tx.Sign(senderSeed)
	if err != nil {
		return
	}

	var txeB64 string
	if txeB64, err = txe.Base64(); err != nil {
		err = &SigningError{err: err}
		return
	}

	if resp, err = nc.SubmitTransaction(txeB64); err != nil {
		return
	}
	log.Debugf("transaction, 'create-account' posted in ledger: %v", resp.Ledger)

	return
}
