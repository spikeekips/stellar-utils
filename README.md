# Basic Utilities For Stellar Network

This simple utilities was written for stellar network. This tools are usually using for maintaining BOSCoin.

## NOTICE

* The default unit of amount must be 'Lumen'(`XLM`), not `stroop`. For more information, please see [Stellar Assets](https://www.stellar.org/developers/guides/concepts/assets.html#one-stroop-multiple-stroops) page.
* The transaction fee can be set manually. The deefault unit of fee must be `stroop`.
* Before making transaction, you must create the proper keypair and create account in network, following the stellar manners.
* The belowed usages, will consider
    * The sender account is already created, it's secret seed is `SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4`.
    * The stellar official testnet, 'https://horizon-testnet.stellar.org' will be used for horizon.


## Preparation
At first, check whether golang is installed. This was tested in golang 1.9.2(darwin/amd64).

```
$ go version
go version go1.9.2 darwin/amd64
```

```
$ mkdir -p stellar-utils/{bin,src}
$ cd stellar-utils
$ export GOPATH=$(pwd)
$ export PATH=$(pwd)/bin:$PATH

$ mkdir -p src/github.com/spikeekips
$ cd src/github.com/spikeekips
$ git clone https://github.com/spikeekips/stellar-utils.git
```


## `stellar-check-account`: Get account information

This is the totally same with the `$ curl <horizon url>/accounts/<account public address>`, but one thing different is, you can do with the secret seed.


```
$ cd stellar-check-account
$ go get
$ go install
```

```
$ stellar-check-account -h
stellar-check-account [options] <public address>
  -horizon string
    	horizon server address
  -verbose
    	verbose

$ stellar-check-account  -horizon https://horizon-testnet.stellar.org -verbose  SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4
{
  "_links": {
...
  },
  "account_id": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
  "balances": [
    {
      "asset_type": "native",
      "balance": "709.9951400"
    }
  ],
  "data": {},
  "flags": {
    "auth_required": false,
    "auth_revocable": false
  },
  "id": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
  "paging_token": "",
  "sequence": "117",
  "signers": [
    {
      "key": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
      "public_key": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
      "type": "ed25519_public_key",
      "weight": 1
    }
  ],
  "subentry_count": 0,
  "thresholds": {
    "high_threshold": 0,
    "low_threshold": 0,
    "med_threshold": 0
  }
}
```

## `stellar-keypair`: Generate and check keypair

This command is the replacement of the official tool, `stellar-core --genseed`. This can also extract the public address from secret seed or networkPassphrase.

```
$ cd stellar-keypair
$ go get
$ go install
```

```
$ stellar-keypair -h
Usage of ./stellar-keypair:
  -short
    	short format, "<secret seed> <public address>"
```

```
$ stellar-keypair
       Secret Seed: SDXVSYBZEQ3NAQ36EP7I257WR77S2ORBI7LH2IT6DYW75Y2JY35XJNPL
    Public Address: GCJADEFKRXZPQK6EJ6OPL224GN6A4I4HBW7KXZIICK7SKKMPCCCOZCIF
```
This will just generate public address and secret seed. With `-short` flag, it will be simpler.


### With Network Passphrase

To get the base secret seed and public address from the network passphrase,

```
$ stellar-keypair "Test SDF Network ; September 2015"
       Secret Seed: SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4
    Public Address: GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H
Network Passphrase: 'Test SDF Network ; September 2015'
```

*info*

> To get the 'networkPassphrase' of the network, you can check the horizon response.
```
$ curl -s https://horizon-testnet.stellar.org
...
  "horizon_version": "snapshot-snapshots-3-gbc25555",
  "core_version": "v9.1.0",
  "history_latest_ledger": 6878240,
  "history_elder_ledger": 1,
  "core_latest_ledger": 6878240,
  "network_passphrase": "Test SDF Network ; September 2015", // <--
  "protocol_version": 9
...
```

## `stellar-create-account`: Create accounts

```
$ cd stellar-create-account
$ go get
$ go install
```

```
$ stellar-create-account -h
  -csv string
    	account csv file
  -fee uint
    	transaction fee (default 10000)
  -horizon string
    	horizon server address
  -verbose
    	verbose
```

By default, this will generate keypair(public address and secret seed for new account) automatically.
```
$ stellar-create-account -verbose --horizon https://horizon-testnet.stellar.org SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4 0.1
(O) Successfully new account is created. GCT255XT7UKN3G43V6EOIT7IPBXBU2M6HRHS7GKZYAJTT6YA7RUVCQB5(SB5IIZ4HZHZ7TSNCGG4EBMXF5LHX7ULT2Z3LQEPCAE43GJK7RAJFFMIZ), 0.1
```

Check the newly created account, 'GCT255XT7UKN3G43V6EOIT7IPBXBU2M6HRHS7GKZYAJTT6YA7RUVCQB5' in network.
```
$ curl https://horizon-testnet.stellar.org/accounts/GCT255XT7UKN3G43V6EOIT7IPBXBU2M6HRHS7GKZYAJTT6YA7RUVCQB5; echo
...
  "balances": [
    {
      "balance": "0.1000000",
      "asset_type": "native"
    }
  ],
...
````

If you already have the specific public address and secret seed,
```
$ stellar-create-account -verbose --horizon https://horizon-testnet.stellar.org SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4 0.1 GAZXF5KYKGTLIXOVYTCVRXR7YDG2244LLAXL5EY7HNJBB2OUAQ6PKZN3
```

### Create Multiple Accounts

```
$ cd stellar-create-account-bulk
$ go get
$ go install
```

Create new csv file like this,
```
GAVQV73MD4ERVVIQVU2C7ZQH22OVHOTV2YKDW4K5RDBILE62EIF456PD,1001.0000000
GDZUBXYMYGEECX22EBPKEH3BI3HCR7DQRU7H5WRYVK7KHVS2VTFXTDLZ,2002.0000000
GBOODJHZKSID5W2YARNHD2WIFBFR7U6OGHX53DDYZFAHBBWQ2Y3CBIC3,3003.0000000
GD3OPPKZEEY2LSYEITBHUBD4ST5TAFWU3WY7XIOD6POZGZRZS5TVUDQQ,4004.0000000
```

This csv file is saved in `/tmp/accounts.csv`. Just add new option, `-csv` with the csv file name.
```
$ stellar-create-account -verbose -horizon https://horizon-testnet.stellar.org -csv /tmp/accounts.csv SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4
```

## `stellar-create-account-bulk`: Create accounts in bulk

Create new csv file like this,
```
GAVQV73MD4ERVVIQVU2C7ZQH22OVHOTV2YKDW4K5RDBILE62EIF456PD,1001.0000000
GDZUBXYMYGEECX22EBPKEH3BI3HCR7DQRU7H5WRYVK7KHVS2VTFXTDLZ,2002.0000000
GBOODJHZKSID5W2YARNHD2WIFBFR7U6OGHX53DDYZFAHBBWQ2Y3CBIC3,3003.0000000
GD3OPPKZEEY2LSYEITBHUBD4ST5TAFWU3WY7XIOD6POZGZRZS5TVUDQQ,4004.0000000
```

This csv file is saved in `/tmp/accounts.csv`. Just add new option, `-csv` with the csv file name.
```
$ stellar-create-account-bulk -verbose -horizon https://horizon-testnet.stellar.org SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4 /tmp/accounts.csv
```

## `stellar-payment`: Seend payment

```
$ cd stellar-payment
$ go get
$ go install
```

```
$ stellar-payment -verbose -horizon https://horizon-testnet.stellar.org SDHOAMBNLGCE2MV5ZKIVZAQD3VCLGP53P3OBSBI6UN5L5XZI5TKHFQL4 GAVQV73MD4ERVVIQVU2C7ZQH22OVHOTV2YKDW4K5RDBILE62EIF456PD 0.01
  sender:            0.0110000:    499886986.6900000 ->    499886986.6790000
receiver:            0.0100000:         1001.0000000 ->         1001.0100000
```
