package abci

import (
	"encoding/binary"
	"fmt"

	// "log"
	"strconv"
	"time"

	abcicode "github.com/tendermint/tendermint/abci/example/code"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	haechiNode "github.com/tendermint/tendermint/csconsensus/haechi/shard/validator"
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

var _ abcitypes.Application = (*HaechiShardApplication)(nil)

type HaechiShardApplication struct {
	abcitypes.BaseApplication
	// mu   sync.Mutex
	Node *haechiNode.ValidatorInterface
}

func NewHaechiShardApplication(node *haechiNode.ValidatorInterface) *HaechiShardApplication {
	return &HaechiShardApplication{
		Node: node,
	}
}

func (HaechiShardApplication) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

func (HaechiShardApplication) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

func (app *HaechiShardApplication) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	return abcitypes.ResponseCheckTx{Code: abcicode.CodeTypeOK, GasWanted: 1}
}

func (app *HaechiShardApplication) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	return abcitypes.ResponseBeginBlock{}
}

func (app *HaechiShardApplication) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	_, tx_json := haechiNode.ResolveTx(req.Tx)
	var err1, err2 error
	var events []abcitypes.Event
	var event_type string
	// fmt.Println("shard receives tx type " + strconv.Itoa(int(tx_json.Tx_type)))
	temp_tx := hctypes.TransactionType{
		From_shard: tx_json.From_shard,
		To_shard:   tx_json.To_shard,
		Tx_type:    tx_json.Tx_type,
		From:       tx_json.From,
		To:         tx_json.To,
		Value:      tx_json.Value,
		Data:       tx_json.Data,
		Nonce:      tx_json.Nonce,
	}
	if tx_json.Tx_type == haechiNode.IntraShard_TX {
		// Attacking test
		if string(temp_tx.From) == "VCTM" || string(temp_tx.From) == "ATTA" {
			fmt.Printf("Attacking trace from %s, transaction id %d \n", string(temp_tx.From), temp_tx.Value)
		}
		event_type = "intra-shard transaction"
		err1 = app.Node.BCState.Database.Set(prefixKey(tx_json.From), []byte("0"))
		err2 = app.Node.BCState.Database.Set(prefixKey(tx_json.To), []byte("0"))
		app.Node.BCState.Size++
	} else if tx_json.Tx_type == haechiNode.InterShard_TX_Verify {
		app.Node.BCState.Index++
		event_type = "inter-shard transaction"
		err1 = app.Node.BCState.Database.Set(prefixKey(tx_json.From), []byte("lock"))
		app.Node.Current_cl += string(req.Tx)
		app.Node.Current_cl += ",blockheight="
		app.Node.Current_cl += strconv.Itoa(int(app.Node.BCState.Height))
		app.Node.Current_cl += ",index="
		app.Node.Current_cl += strconv.Itoa(int(app.Node.BCState.Index))
		app.Node.Current_cl += ">"
		// Attacking test
		if app.Node.Byzantine == 1 && string(temp_tx.From) == "VCTM" { // construct front-running tx: new_tx1
			if string(temp_tx.To) == "INTR" { // front-running attack with intra-shard txs
				temp_tx.From_shard = temp_tx.To_shard
				temp_tx.Tx_type = haechiNode.InterShard_TX_Verify

			} else if string(temp_tx.To) == "CROS" { // front-running attack with cross-shard txs
				temp_tx.From_shard = app.Node.Attack_shard
				temp_tx.Tx_type = haechiNode.InterShard_TX_Verify
			}
			temp_tx.Nonce = uint32(app.Node.Node_id)
			temp_tx.From = []byte("ATTA")
			_, attack_tx := haechiNode.Deserilization(temp_tx)
			go app.Node.DeliverAttackTx(attack_tx, temp_tx.To_shard) // sending front-running tx
		}
	} else if tx_json.Tx_type == haechiNode.CrossShard_Call_List {
		// fmt.Println("shard receives call list")
		go app.Node.HandleCallList(req.Tx)
	} else if tx_json.Tx_type == haechiNode.InterShard_TX_Commit {
		event_type = "inter-shard commit transaction"
		err1 = app.Node.BCState.Database.Set(prefixKey(tx_json.From), []byte("0"))
		// Trace: cross-shard tx confirmation latency
		if string(tx_json.From) == "CROS" {
			fmt.Println("cross-shard trace, nonce is", tx_json.Nonce)
			fmt.Println("cross-shard trace, end time is", time.Now())
		}
		tx_json.Tx_type = haechiNode.InterShard_TX_Update
		_, new_tx := haechiNode.Deserilization(tx_json)
		if app.Node.Leader {
			go app.Node.DeliverUpdateTx(new_tx, tx_json.To_shard)
			// app.Node.DeliverUpdateTx(new_tx, tx_json.To_shard)
		}
	} else if tx_json.Tx_type == haechiNode.InterShard_TX_Update {
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

func (app *HaechiShardApplication) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	return abcitypes.ResponseEndBlock{}
}

func (app *HaechiShardApplication) Commit() abcitypes.ResponseCommit {
	// tln("commit tx, current time is: " + time.Now().String())
	current_timestamp := time.Now().Unix()
	if app.Node.Leader && app.Node.Current_cl != "" {
		cl := app.Node.Current_cl
		app.Node.Current_cl = ""
		go app.Node.DeliverCrossLink(current_timestamp, cl)
	}
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, int64(app.Node.BCState.Size))
	app.Node.BCState.AppHash = appHash
	app.Node.BCState.Height++
	app.Node.BCState.Index = 0
	return abcitypes.ResponseCommit{Data: []byte{}}
}

func (app *HaechiShardApplication) Query(reqQuery abcitypes.RequestQuery) (resQuery abcitypes.ResponseQuery) {
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
		resQuery.Height = int64(app.Node.BCState.Height)

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
	resQuery.Height = int64(app.Node.BCState.Height)

	return resQuery
}
