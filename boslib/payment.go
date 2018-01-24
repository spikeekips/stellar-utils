package boslib

import (
	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/xdr"
)

func SendPayment(horizonUrl, senderSeed, receiverAddress, amount, networkPassphrase string, seq xdr.SequenceNumber, fee uint64) (
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
		b.SourceAccount{senderSeed},
		sp,
		b.Payment(
			b.Destination{receiverAddress},
			b.NativeAmount{amount},
		),
		b.BaseFee{Amount: fee},
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
	log.Debugf("< transaction, 'payment' posted in ledger: %v", resp.Ledger)

	return resp, nil
}
