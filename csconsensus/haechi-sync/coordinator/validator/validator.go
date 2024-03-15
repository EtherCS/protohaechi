package validator

import (
	"bytes"
	"log"
	"math"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	aq "github.com/emirpasic/gods/queues/arrayqueue"
	hctypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

type BlockchainState struct {
	Database dbm.DB
	Size     int64
	Height   int64
	AppHash  []byte
}

func NewBlockchainState(name string, dir string) *BlockchainState {
	var bcstate BlockchainState
	var err error
	bcstate.Database, err = dbm.NewDB(name, dbm.GoLevelDBBackend, dir)
	bcstate.Height = 1
	bcstate.Size = 0
	if err != nil {
		log.Fatalf("Create database error: %v", err)
	}
	return &bcstate
}

/*
In the current implementation, we decompose transactions from each CrossLink to ShardCLMsgs for simplicity
*/
type ShardCrosslinkMsg struct {
	CL *aq.Queue // queue used to store CrossLink's transactions
}

var (
	paraGuard sync.Mutex
)

type ValidatorInterface struct {
	BCState              *BlockchainState
	ShardCLMsgs          []ShardCrosslinkMsg // ShardCLMsgs[i].CL contains a list of transactions from consecutive CrossLinks
	ShardCLPool          []ShardCrosslinkMsg
	ShardLastBlockHeight []uint32
	ValidTSRange         [2]int64
	ShardBlockLastTS     []int64
	Leader               bool
	input_addr           hctypes.HaechiAddress
	output_shards_addrs  []hctypes.HaechiAddress
	shard_num            uint8
	currentCCLs          hctypes.CrossShardCallLists
	min_next_TS          int64
	Start_Order          bool
}

func NewValidatorInterface(bcstate *BlockchainState, shard_num uint8, leader bool, in_addr hctypes.HaechiAddress, out_addrs []hctypes.HaechiAddress) *ValidatorInterface {
	var new_validator ValidatorInterface
	new_validator.BCState = bcstate
	new_validator.shard_num = shard_num
	new_validator.ShardCLMsgs = make([]ShardCrosslinkMsg, shard_num)
	new_validator.ShardCLPool = make([]ShardCrosslinkMsg, shard_num)

	for i := uint8(0); i < shard_num; i++ {
		new_validator.ShardCLMsgs[i].CL = aq.New()
		new_validator.ShardCLPool[i].CL = aq.New()
	}
	new_validator.ShardBlockLastTS = make([]int64, shard_num)
	new_validator.ShardLastBlockHeight = make([]uint32, shard_num)
	for i := uint8(0); i < shard_num; i++ {
		new_validator.ShardBlockLastTS = append(new_validator.ShardBlockLastTS, 0)
		new_validator.ShardLastBlockHeight = append(new_validator.ShardLastBlockHeight, 0)
	}
	new_validator.ShardBlockLastTS = new_validator.ShardBlockLastTS[1:]
	new_validator.ShardLastBlockHeight = new_validator.ShardLastBlockHeight[1:]
	new_validator.ValidTSRange[0] = time.Now().Unix()
	new_validator.ValidTSRange[1] = math.MaxInt64
	new_validator.Leader = leader
	new_validator.input_addr = in_addr
	new_validator.output_shards_addrs = make([]hctypes.HaechiAddress, shard_num)
	new_validator.currentCCLs = make(hctypes.CrossShardCallLists, shard_num)
	for i := uint8(0); i < shard_num; i++ {
		new_validator.output_shards_addrs[i].Ip = out_addrs[i].Ip
		new_validator.output_shards_addrs[i].Port = out_addrs[i].Port
		new_validator.currentCCLs[i].Call_txs = aq.New()
	}
	new_validator.min_next_TS = math.MaxInt64
	new_validator.Start_Order = true
	return &new_validator
}

// TODO: concurrency control
func (nw *ValidatorInterface) DeliverCallLists() {
	// fmt.Println(fmt.Sprintf("DeliverCallLists start FormCCLs at %v", time.Now()))
	paraGuard.Lock()
	nw.FormCCLs()
	// fmt.Println(fmt.Sprintf("DeliverCallLists end FormCCLs at %v", time.Now()))
	// fmt.Println(fmt.Sprintf("DeliverCallLists start DeliverCallList at %v", time.Now()))
	for i := uint8(0); i < nw.shard_num; i++ {
		go nw.DeliverCallList(i)
		// nw.DeliverCallList(i)
	}
	// fmt.Println(fmt.Sprintf("DeliverCallLists end DeliverCallList at %v", time.Now()))
	nw.Start_Order = true
	defer paraGuard.Unlock()
}

func (nw *ValidatorInterface) FormCCLs() {
	nw.UpdateTimestampRange()
	var cls_size int
	for i := uint(0); i < uint(nw.shard_num); i++ {
		if nw.ShardCLMsgs[i].CL.Empty() {
			continue
		}
		cls_size = nw.ShardCLMsgs[i].CL.Size()
		for j := uint(0); j < uint(cls_size); j++ {
			cl_temp, _ := nw.ShardCLMsgs[i].CL.Dequeue()
			cl := cl_temp.(hctypes.CrossLinkTransaction)
			if cl.Block_timestamp > nw.ValidTSRange[1]+100 { // allow deviation
				// advanced cross link
				// temp_output := fmt.Sprintf("Haechi trace BTS is %v, validTS is %v", cl.Block_timestamp, nw.ValidTSRange[1])
				// fmt.Println(temp_output)
				nw.ShardCLMsgs[i].CL.Enqueue(cl)
			} else {
				nw.currentCCLs[cl.To_shard].Call_txs.Enqueue(cl)
			}
			// nw.currentCCLs[cl.To_shard].Call_txs.Enqueue(cl)
		}
	}
	// fmt.Println(fmt.Sprintf("DeliverCallLists start GlobalOrdering at %v", time.Now()))
	nw.GlobalOrdering()
	// fmt.Println(fmt.Sprintf("DeliverCallLists end GlobalOrdering at %v", time.Now()))
}

func (nw *ValidatorInterface) GlobalOrdering() {
	for shard_id := uint(0); shard_id < uint(nw.shard_num); shard_id++ {
		ccl_size := nw.currentCCLs[shard_id].Call_txs.Size()
		if ccl_size == 0 {
			continue
		}
		temp_cls := make([]hctypes.CrossLinkTransaction, ccl_size)
		for k := uint(0); k < uint(ccl_size); k++ {
			temp_tx, _ := nw.currentCCLs[shard_id].Call_txs.Dequeue()
			cl_tx := temp_tx.(hctypes.CrossLinkTransaction)
			temp_cls[k] = cl_tx
		}
		sort.SliceStable(temp_cls, func(i, j int) bool {
			return temp_cls[i].Index < temp_cls[j].Index
		})
		sort.SliceStable(temp_cls, func(i, j int) bool {
			return temp_cls[i].Block_timestamp < temp_cls[j].Block_timestamp
		})
		nw.currentCCLs[shard_id].Call_txs.Clear()

		for j := uint(0); j < uint(len(temp_cls)); j++ {
			nw.currentCCLs[shard_id].Call_txs.Enqueue(temp_cls[j])
		}
	}
}

// TODO: 1) compress data transmissionk; 2) speed up the formation of CallList
func (nw *ValidatorInterface) DeliverCallList(shard_id uint8) {
	tx_string := ""
	for !nw.currentCCLs[shard_id].Call_txs.Empty() {
		temp_tx, _ := nw.currentCCLs[shard_id].Call_txs.Dequeue()
		cl_tx := temp_tx.(hctypes.CrossLinkTransaction)
		tx_string += "fromid="
		tx_string += strconv.Itoa(int(cl_tx.From_shard))
		tx_string += ",toid="
		tx_string += strconv.Itoa(int(cl_tx.To_shard))
		tx_string += ",type=5" // CrossShard_Call_List
		tx_string += ",from="
		tx_string += string(cl_tx.From)
		tx_string += ",to="
		tx_string += string(cl_tx.To)
		tx_string += ",value="
		tx_string += strconv.Itoa(int(cl_tx.Value))
		tx_string += ",data="
		tx_string += string(cl_tx.Data)
		tx_string += ",nonce="
		tx_string += strconv.Itoa(int(cl_tx.Nonce))
		tx_string += ">"
	}
	// fmt.Println(fmt.Sprintf("beacon chain send %v to shard id %v", tx_string, strconv.Itoa(int(shard_id))))
	if tx_string != "" {
		receiver_addr := net.JoinHostPort(nw.output_shards_addrs[shard_id].Ip.String(), strconv.Itoa(int(nw.output_shards_addrs[shard_id].Port)))
		request := receiver_addr
		request += "/broadcast_tx_commit?tx=\""
		request += tx_string
		request += "\""
		http.Get("http://" + request)
		// _, err := http.Get("http://" + request)
		// if err != nil {
		// 	fmt.Println(fmt.Sprintf("Error: tx error when request a curl %v", err))
		// }
	}
}

func (nw *ValidatorInterface) UpdateShardCrosslinkMsgs(request []byte) {
	// paraGuard.Lock()
	shardid := CheckFromShardId(request)
	blockheight := CheckBlockHeight(request)
	var blockts = CheckBlockTimestamp(request)
	// MicroBench: test time difference of shard's CrossLinks
	// temp_output := fmt.Sprintf("Haechi trace: receive CrossLink from %v with height %v, last height %v, at time %v, request %v", shardid, blockheight, nw.ShardLastBlockHeight[shardid], time.Now(), string(request))
	// fmt.Println(temp_output)
	// TODO: packet loss
	if blockheight == nw.ShardLastBlockHeight[shardid] && blockheight != 0 { // non-consecutive crosslink add to CLPool
		_, crosslinkTxs := hctypes.RequestToCrossLinkTxs(request)
		for _, crosslinkTx := range crosslinkTxs {
			nw.ShardCLPool[shardid].CL.Enqueue(crosslinkTx)
		}
	} else { // consecutive crosslink
		_, crosslinkTxs := hctypes.RequestToCrossLinkTxs(request)
		for _, crosslinkTx := range crosslinkTxs {
			nw.ShardCLMsgs[shardid].CL.Enqueue(crosslinkTx)
		}
		if !nw.ShardCLPool[shardid].CL.Empty() {
			temp_tx, _ := nw.ShardCLPool[shardid].CL.Peek()
			cl_tx := temp_tx.(hctypes.CrossLinkTransaction)
			for cl_tx.Block_height == blockheight+1 || cl_tx.Block_height == blockheight { // blockheight is the latest consecutive block height
				if cl_tx.Block_height == blockheight+1 {
					blockheight += 1
				}
				temp_new_tx, _ := nw.ShardCLPool[shardid].CL.Dequeue()
				nw.ShardCLMsgs[shardid].CL.Enqueue(temp_new_tx)
				temp_tx, _ = nw.ShardCLPool[shardid].CL.Peek()
				cl_tx = temp_tx.(hctypes.CrossLinkTransaction)
				blockts = cl_tx.Block_timestamp
			}
		}
		// fmt.Println("UpdateShardCrosslinkMsgs blockts " + strconv.Itoa(int(blockts)) + "last block ts " + strconv.Itoa(int(nw.ShardBlockLastTS[shardid])))
		nw.UpdateOrderParameters(blockts, shardid, blockheight)
	}
	// defer paraGuard.Unlock()
}

func (nw *ValidatorInterface) UpdateOrderParameters(blockts int64, shardid uint8, blockheight uint32) {
	nw.ShardBlockLastTS[shardid] = blockts

	nw.ShardLastBlockHeight[shardid] = blockheight

	if nw.ShardBlockLastTS[shardid] < nw.min_next_TS {
		nw.min_next_TS = nw.ShardBlockLastTS[shardid]
	}
}

func (nw *ValidatorInterface) UpdateTimestampRange() {
	nw.ValidTSRange[0] = nw.ValidTSRange[1]
	nw.ValidTSRange[1] = nw.min_next_TS

}

func (nw *ValidatorInterface) StartOrder() bool {
	start := true
	for _, ccl := range nw.ShardCLMsgs {
		if ccl.CL.Empty() {
			start = false
			return start
		}
	}
	return start
}

func CheckFromShardId(tx []byte) uint8 {
	var fromshardid uint8
	tx_elements := bytes.Split(tx, []byte(","))
	for _, tx_element := range tx_elements {
		kv := bytes.Split(tx_element, []byte("="))
		if string(kv[0]) == "fromid" {
			temp_type64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			fromshardid = uint8(temp_type64)
			break
		}
	}
	return fromshardid
}

func CheckBlockTimestamp(tx []byte) int64 {
	var bts int64
	txs := bytes.Split(tx, []byte(">"))
	tx_elements := bytes.Split(txs[0], []byte(","))
	for _, tx_element := range tx_elements {
		kv := bytes.Split(tx_element, []byte("="))
		if string(kv[0]) == "blocktimestamp" {
			temp_type64, _ := strconv.ParseInt(string(kv[1]), 10, 64)
			bts = temp_type64
			break
		}
	}

	return bts
}

func CheckBlockHeight(tx []byte) uint32 {
	var blockheight uint32
	tx_elements := bytes.Split(tx, []byte(","))
	for _, tx_element := range tx_elements {
		kv := bytes.Split(tx_element, []byte("="))
		if string(kv[0]) == "blockheight" {
			temp_type64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			blockheight = uint32(temp_type64)
			break
		}
	}
	return blockheight
}
