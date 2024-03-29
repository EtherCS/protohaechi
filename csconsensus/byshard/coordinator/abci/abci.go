package abci

import (
	"encoding/binary"
	"fmt"
	"time"

	// "log"
	// "time"

	// "log"
	"strconv"

	abcicode "github.com/tendermint/tendermint/abci/example/code"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	byshardNode "github.com/tendermint/tendermint/csconsensus/byshard/coordinator/validator"
	hctypes "github.com/tendermint/tendermint/types"
)

var (
	// stateKey        = []byte("stateKey")
	kvPairPrefixKey = []byte("kvPairKey:")

	ProtocolVersion uint64 = 0x1
)

func prefixKey(key []byte) []byte {
	return append(kvPairPrefixKey, key...)
}

var _ abcitypes.Application = (*byshardApplication)(nil)

type byshardApplication struct {
	abcitypes.BaseApplication
	// mu   sync.Mutex
	Node *byshardNode.ValidatorInterface
	// intraTxBatch *badger.Txn
}

func NewbyshardApplication(node *byshardNode.ValidatorInterface) *byshardApplication {
	return &byshardApplication{
		Node: node,
	}
}

func (byshardApplication) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

func (byshardApplication) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

func (app *byshardApplication) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	return abcitypes.ResponseCheckTx{Code: abcicode.CodeTypeOK, GasWanted: 1}
}

func (app *byshardApplication) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	return abcitypes.ResponseBeginBlock{}
}

func (app *byshardApplication) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	_, tx_json := byshardNode.ResolveTx(req.Tx)
	var err1, err2 error
	var events []abcitypes.Event
	var event_type string
	new_tx := hctypes.TransactionType{
		From_shard: tx_json.From_shard,
		To_shard:   tx_json.To_shard,
		Tx_type:    tx_json.Tx_type,
		From:       tx_json.From,
		To:         tx_json.To,
		Value:      tx_json.Value,
		Data:       tx_json.Data,
		Nonce:      tx_json.Nonce,
	}
	if tx_json.Tx_type == byshardNode.IntraShard_TX {
		// Attacking test
		if string(new_tx.From) == "VCTM" || string(new_tx.From) == "ATTA" {
			fmt.Printf("Attacking trace from %s, transaction id %d \n", string(new_tx.From), new_tx.Value)
		}
		event_type = "intra-shard transaction"
		err1 = app.Node.BCState.Database.Set(prefixKey(tx_json.From), []byte("0"))
		err2 = app.Node.BCState.Database.Set(prefixKey(tx_json.To), []byte("0"))
		app.Node.BCState.Size++
	} else if tx_json.Tx_type == byshardNode.InterShard_TX_Verify {
		// fmt.Println("Byshard: this is InterShard_TX_Verify transaction")
		event_type = "inter-shard verification transaction"
		err1 = app.Node.BCState.Database.Set(prefixKey(tx_json.From), []byte("lock"))
		new_tx.Tx_type = byshardNode.InterShard_TX_Execute
		_, exec_tx := byshardNode.Deserilization(new_tx)
		if app.Node.Leader {
			go app.Node.DeliverExecutionTx(exec_tx, new_tx.To_shard)
		}
		// Attacking test
		if app.Node.Byzantine == 1 && string(new_tx.From) == "VCTM" { // construct front-running tx: new_tx1
			if string(new_tx.To) == "INTR" { // front-running attack with intra-shard txs
				new_tx.From_shard = new_tx.To_shard
				new_tx.Tx_type = byshardNode.IntraShard_TX

			} else if string(new_tx.To) == "CROS" { // front-running attack with cross-shard txs
				new_tx.From_shard = app.Node.Attack_shard
				new_tx.Tx_type = byshardNode.InterShard_TX_Verify
			}
			new_tx.Nonce = uint32(app.Node.Node_id)
			new_tx.From = []byte("ATTA")
			_, attack_tx := byshardNode.Deserilization(new_tx)
			go app.Node.DeliverAttackTx(attack_tx, new_tx.To_shard) // sending front-running tx
		}
	} else if tx_json.Tx_type == byshardNode.InterShard_TX_Execute {
		// fmt.Println("Byshard: this is InterShard_TX_Execute transaction")
		event_type = "inter-shard execution transaction"
		err2 = app.Node.BCState.Database.Set(prefixKey(tx_json.To), []byte("lock"))
		new_tx.Tx_type = byshardNode.InterShard_TX_Commit
		_, commit_tx := byshardNode.Deserilization(new_tx)
		if app.Node.Leader {
			go app.Node.DeliverCommitTx(commit_tx, new_tx.From_shard)
		}
		// Attacking test
		if string(new_tx.From) == "VCTM" || string(new_tx.From) == "ATTA" {
			fmt.Printf("Attacking trace from %s, transaction id %d \n", string(new_tx.From), new_tx.Value)
		}
	} else if tx_json.Tx_type == byshardNode.InterShard_TX_Commit {
		// fmt.Println("Byshard: this is InterShard_TX_Commit transaction")
		event_type = "inter-shard commit transaction"
		err1 = app.Node.BCState.Database.Set(prefixKey(tx_json.From), []byte("0"))
		// Trace: cross-shard tx confirmation latency
		if string(tx_json.From) == "CROS" {
			fmt.Println("cross-shard trace, nonce is", tx_json.Nonce)
			fmt.Println("cross-shard trace, end time is", time.Now())
		}
		new_tx.Tx_type = byshardNode.InterShard_TX_Update
		_, update_tx := byshardNode.Deserilization(new_tx)
		if app.Node.Leader {
			go app.Node.DeliverUpdateTx(update_tx, new_tx.To_shard)
		}
	} else if tx_json.Tx_type == byshardNode.InterShard_TX_Update {
		// fmt.Println("Byshard: this is InterShard_TX_Update transaction")
		event_type = "inter-shard update transaction"
		err2 = app.Node.BCState.Database.Set(prefixKey(tx_json.To), []byte("0"))
		app.Node.BCState.Size++
	}
	if err1 != nil || err2 != nil {
		panic(err1)
	}
	events = []abcitypes.Event{
		{
			Type: event_type,
			Attributes: []abcitypes.EventAttribute{
				{Key: "from", Value: string(tx_json.From), Index: true},
				{Key: "to", Value: string(tx_json.To), Index: true},
				{Key: "value", Value: strconv.Itoa(int(tx_json.Value)), Index: true},
				{Key: "data", Value: string(tx_json.Data), Index: true},
			},
		},
	}
	return abcitypes.ResponseDeliverTx{Code: abcicode.CodeTypeOK, Events: events}
}

func (app *byshardApplication) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	return abcitypes.ResponseEndBlock{}
}

func (app *byshardApplication) Commit() abcitypes.ResponseCommit {
	// tln("commit tx, current time is: " + time.Now().String())
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.Node.BCState.Size)
	app.Node.BCState.AppHash = appHash
	app.Node.BCState.Height++
	return abcitypes.ResponseCommit{Data: []byte{}}
}

func (app *byshardApplication) Query(reqQuery abcitypes.RequestQuery) (resQuery abcitypes.ResponseQuery) {
	if reqQuery.Prove {
		value, err := app.Node.BCState.Database.Get(prefixKey(reqQuery.Data))
		if err != nil {
			panic(err)
		}
		if value == nil {
			resQuery.Log = "does not exist"
		} else {
			resQuery.Log = "exists"
		}
		resQuery.Index = -1 // TODO make Proof return index
		resQuery.Key = reqQuery.Data
		resQuery.Value = value
		resQuery.Height = app.Node.BCState.Height

		return
	}

	resQuery.Key = reqQuery.Data
	value, err := app.Node.BCState.Database.Get(prefixKey(reqQuery.Data))
	if err != nil {
		panic(err)
	}
	if value == nil {
		resQuery.Log = "does not exist"
	} else {
		resQuery.Log = "exists"
	}
	resQuery.Value = value
	resQuery.Height = app.Node.BCState.Height

	return resQuery
}
