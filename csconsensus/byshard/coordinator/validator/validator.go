package validator

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"net/http"
	"strconv"

	hctypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

/*
A transaction is defined as a json including:
{
	tx_type uint32
	from	[20]byte
	to		[20]byte
	value	uint32
	data	[20]byte
}
e.g., sent by user as: "type=0,from=ABCD,to=DCBA,value=0,data=NONE"
*/
const (
	Addr_Length uint8 = 4
	Data_Length uint8 = 4
)
const (
	Type_Num              uint8 = 5
	IntraShard_TX         uint8 = 0
	InterShard_TX_Verify  uint8 = 1
	InterShard_TX_Execute uint8 = 2
	InterShard_TX_Commit  uint8 = 3
	InterShard_TX_Update  uint8 = 4
)

// var (
// 	stateKey = []byte("stateKey")
// )

type BlockchainState struct {
	Database dbm.DB
	Size     int64
	Height   int64
	AppHash  []byte
}

func NewBlockchainState(name string, dir string) *BlockchainState {
	var bcstate BlockchainState
	var err error
	// bcstate, _ := loadState(dbm.DB.(name, dir))
	bcstate.Database, err = dbm.NewDB(name, dbm.GoLevelDBBackend, dir)
	bcstate.Height = 0
	bcstate.Size = 0
	if err != nil {
		log.Fatalf("Create database error: %v", err)
	}
	return &bcstate
}

type ValidatorInterface struct {
	BCState             *BlockchainState
	shard_num           uint8
	Shard_id            uint8
	Leader              bool
	input_addr          hctypes.HaechiAddress
	output_shards_addrs []hctypes.HaechiAddress
	Byzantine           uint8 // 0: honest; 1: byzantine
	Attack_shard        uint8 // the attacker's shard
	Node_id             uint8 // used to construct different front-running tx
}

func NewValidatorInterface(bcstate *BlockchainState, shard_num uint8, shard_id uint8, leader bool, in_addr hctypes.HaechiAddress, out_addrs []hctypes.HaechiAddress, attacks ...uint8) *ValidatorInterface {
	var new_validator ValidatorInterface
	new_validator.BCState = bcstate
	new_validator.shard_num = shard_num
	new_validator.Shard_id = shard_id
	new_validator.Leader = leader
	new_validator.input_addr = in_addr
	new_validator.output_shards_addrs = make([]hctypes.HaechiAddress, shard_num)
	for i := uint8(0); i < shard_num; i++ {
		new_validator.output_shards_addrs[i].Ip = out_addrs[i].Ip
		new_validator.output_shards_addrs[i].Port = out_addrs[i].Port
	}
	if len(attacks) == 1 {
		new_validator.Byzantine = attacks[0]
	} else if len(attacks) == 2 { // attacks[0]: isByzantine; attacks[1]: attack shard id
		new_validator.Byzantine = attacks[0]    // default: honest
		new_validator.Attack_shard = attacks[1] // default: itself
	} else if len(attacks) == 3 { // attacks[0]: isByzantine; attacks[1]: attack shard id; attacks[2]: attacker id
		new_validator.Byzantine = attacks[0]    // default: honest
		new_validator.Attack_shard = attacks[1] // default: itself
		new_validator.Node_id = attacks[2]
	} else {
		new_validator.Byzantine = 0
		new_validator.Attack_shard = new_validator.Shard_id
		new_validator.Node_id = 0
	}
	return &new_validator
}

func (nw *ValidatorInterface) DeliverExecutionTx(tx []byte, shardid uint8) {
	tx_str := string(tx)
	receiver_addr := net.JoinHostPort(nw.output_shards_addrs[shardid].Ip.String(), strconv.Itoa(int(nw.output_shards_addrs[shardid].Port)))
	request := receiver_addr
	request += "/broadcast_tx_commit?tx=\""
	request += tx_str
	request += "\""
	http.Get("http://" + request)
	// fmt.Println("Byshard: sending request", "http://"+request)
	// _, err := http.Get("http://" + request)
	// if err != nil {
	// 	fmt.Println("Error: deliver execution tx error when request a curl", err)
	// }
}

func (nw *ValidatorInterface) DeliverCommitTx(tx []byte, shardid uint8) {
	tx_str := string(tx)
	sender_addr := net.JoinHostPort(nw.output_shards_addrs[shardid].Ip.String(), strconv.Itoa(int(nw.output_shards_addrs[shardid].Port)))
	request := sender_addr
	request += "/broadcast_tx_commit?tx=\""
	request += tx_str
	request += "\""
	http.Get("http://" + request)
	// fmt.Println("Byshard: sending request", "http://"+request)
	// _, err := http.Get("http://" + request)
	// if err != nil {
	// 	fmt.Println("Error: deliver commit tx error when request a curl", err)
	// }
}

func (nw *ValidatorInterface) DeliverUpdateTx(tx []byte, shardid uint8) {
	// output_addr := net.JoinHostPort(nw.ip_output_shard.String(), strconv.Itoa(int(nw.port_output_shard)))
	tx_str := string(tx)
	receiver_addr := net.JoinHostPort(nw.output_shards_addrs[shardid].Ip.String(), strconv.Itoa(int(nw.output_shards_addrs[shardid].Port)))
	request := receiver_addr
	request += "/broadcast_tx_commit?tx=\""
	request += tx_str
	request += "\""
	http.Get("http://" + request)
	// fmt.Println("Byshard: sending request", "http://"+request)
	// _, err := http.Get("http://" + request)
	// if err != nil {
	// 	fmt.Println("Error: deliver update tx error when request a curl")
	// }
}

func (nw *ValidatorInterface) DeliverAttackTx(tx []byte, shardid uint8) {
	tx_str := string(tx)
	receiver_addr := net.JoinHostPort(nw.output_shards_addrs[shardid].Ip.String(), strconv.Itoa(int(nw.output_shards_addrs[shardid].Port)))
	request := receiver_addr
	request += "/broadcast_tx_commit?tx=\""
	request += tx_str
	request += "\""
	http.Get("http://" + request)
	// _, _ := http.Get("http://" + request)
	// if err != nil {
	// 	fmt.Println("Error: deliver execution tx error when request a curl")
	// }
}

func Serilization(tx []byte) (uint32, hctypes.TransactionType) {
	var tx_json hctypes.TransactionType
	// tx_json.Tx_type = 4
	txElements := bytes.Split(tx, []byte(","))
	if len(txElements) == 0 {
		return 1, tx_json
	}
	for _, txElement := range txElements {
		kv := bytes.Split(txElement, []byte("="))
		switch {
		case string(kv[0]) == "fromid":
			temp_type64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			tx_json.From_shard = uint8(temp_type64)
		case string(kv[0]) == "toid":
			temp_type64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			tx_json.To_shard = uint8(temp_type64)
		case string(kv[0]) == "type":
			temp_type64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			tx_json.Tx_type = uint8(temp_type64)
			if tx_json.Tx_type == 5 {
				return 0, tx_json
			}
		case string(kv[0]) == "from":
			tx_json.From = make([]byte, Addr_Length)
			copy(tx_json.From, kv[1])
		case string(kv[0]) == "to":
			tx_json.To = make([]byte, Addr_Length)
			copy(tx_json.To, kv[1])
		case string(kv[0]) == "value":
			// temp_value := string(kv[1])
			temp_value64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			tx_json.Value = uint32(temp_value64)
		case string(kv[0]) == "data":
			tx_json.Data = make([]byte, Data_Length)
			copy(tx_json.Data, kv[1])
		case string(kv[0]) == "nonce":
			temp_value64, _ := strconv.ParseUint(string(kv[1]), 10, 64)
			tx_json.Nonce = uint32(temp_value64)
		}
	}
	return 0, tx_json
}

func Deserilization(tx hctypes.TransactionType) (uint32, []byte) {
	var tempStr string = ""
	tempStr += "fromid="
	tempStr += strconv.Itoa(int(tx.From_shard))
	tempStr += ",toid="
	tempStr += strconv.Itoa(int(tx.To_shard))
	tempStr += ",type="
	tempStr += strconv.Itoa(int(tx.Tx_type))
	tempStr += ",from="
	tempStr += string(tx.From)
	tempStr += ",to="
	tempStr += string(tx.To)
	tempStr += ",value="
	tempStr += strconv.Itoa(int(tx.Value))
	tempStr += ",data="
	tempStr += string(tx.Data)
	tempStr += ",nonce="
	tempStr += strconv.Itoa(int(tx.Nonce))
	tx_byte := []byte(tempStr)
	return 0, tx_byte
}

func ResolveTx(_tx []byte) (uint32, hctypes.TransactionType) {
	isValid, tx := Serilization(_tx)
	if isValid != 0 {
		return 1, tx
	}
	if tx.Tx_type < Type_Num {
		return 0, tx
	} else {
		return 1, tx
	}
	// txType := IntraShard_TX
	// return txType
}

func GetTxType(_tx []byte) uint8 {
	_, tx := Serilization(_tx)
	return tx.Tx_type
}

func AddOperation(add []byte, value uint32) []byte {
	temp_add := binary.BigEndian.Uint32(add)
	temp_add += value
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, temp_add)
	return buf.Bytes()
}

func SubOperation(minuend []byte, value uint32) []byte {
	temp_minuend := binary.BigEndian.Uint32(minuend)
	if temp_minuend >= value {
		temp_minuend -= value
	}
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, temp_minuend)
	return buf.Bytes()
}
