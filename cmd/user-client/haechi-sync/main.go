package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

// ./build/haechiuser -shards 4 -size 30 -beaconport 10057 -beaconip "127.0.0.1" -shardports "20057" -shardips "127.0.0.1,127.0.0.1,127.0.0.1,127.0.0.1" -leaderports "20057,30057,40057,50057" -parallel 10 -duration 180

var shardNum, shardSize, batchSize, beaconPort, concurrentNum, reqDuration uint

// var requestRate int64
var crossRate float64
var shardPorts, beaconIp, shardIps, leaderPorts, publicIps string

func init() {
	flag.UintVar(&shardNum, "shards", 2, "the number of shards")
	flag.UintVar(&shardSize, "size", 4, "shard size")
	flag.UintVar(&batchSize, "batch", 10, "the batch size of one request")
	flag.UintVar(&concurrentNum, "parallel", 100, "concurrent number for sending requests")
	flag.UintVar(&reqDuration, "duration", 120, "duration of sending request")
	flag.Float64Var(&crossRate, "ratio", 0.9, "the ratio of cross-shard txs")

	flag.UintVar(&beaconPort, "beaconport", 10057, "beacon chain port")
	flag.StringVar(&shardPorts, "shardports", "20057,21057", "shards chain port")
	flag.StringVar(&beaconIp, "beaconip", "127.0.0.1", "beacon chain ip")
	flag.StringVar(&shardIps, "shardips", "127.0.0.1,127.0.0.1", "shards chain ip")
	flag.StringVar(&leaderPorts, "leaderports", "20057,30057,40057,50057", "shards chain leader port")
	flag.StringVar(&publicIps, "publicips", "127.0.0.1", "public ips")
}

func main() {
	flag.Parse()
	// initialize ip
	shard_ports_temp := []byte(shardPorts)
	shard_ports := bytes.Split(shard_ports_temp, []byte(","))
	shard_ips_temp := []byte(shardIps)
	shard_ips := bytes.Split(shard_ips_temp, []byte(","))
	public_ips_temp := []byte(publicIps)
	public_ips := bytes.Split(public_ips_temp, []byte(","))
	leader_ports_temp := []byte(leaderPorts)
	leader_ports := bytes.Split(leader_ports_temp, []byte(","))
	var ports_value64 []uint64
	var leader_ports_value64 []uint64
	// for shard activation
	for _, leader_port := range leader_ports {
		temp_leader_port, _ := strconv.ParseUint(string(leader_port), 10, 64)
		leader_ports_value64 = append(leader_ports_value64, temp_leader_port)
	}
	for _, shard_port := range shard_ports {
		temp_port, _ := strconv.ParseUint(string(shard_port), 10, 64)
		// each node in a shard
		for i := 0; i < int(shardSize); i++ {
			// fmt.Println("Byshard: listening port", temp_port+100*uint64(i))
			ports_value64 = append(ports_value64, temp_port+100*uint64(i))
		}
	}
	fmt.Println("Haechi: client size is", concurrentNum)
	fmt.Println("Haechi: the size of nodes receiving requests is", len(ports_value64))
	for p := 0; p < int(concurrentNum); p++ {
		// for i, _ := range shard_ips {
		for j, _ := range ports_value64 {
			go send_request(uint(ports_value64[j]), string(shard_ips[0]), beaconPort, beaconIp, batchSize, uint(0), shardNum, crossRate)
		}
		// }
		// time.Sleep(time.Duration(requestRate) * time.Millisecond)
	}
	// activate other shards
	for i, _ := range public_ips {
		for j, l_p := range leader_ports_value64 {
			// fmt.Println("Haechi: send txs to ip", string(public_ips[i])+":"+fmt.Sprint(l_p))
			if uint(i) <= shardNum/4 {
				fmt.Println("Haechi: send txs to ip", string(public_ips[i])+":"+fmt.Sprint(l_p))
				go send_request(uint(l_p), string(public_ips[i]), beaconPort, beaconIp, batchSize, uint(i*4+j), shardNum, crossRate)
			}
		}

	}
	time.Sleep(time.Duration(reqDuration) * time.Second)
}

func get_rand(upperBond int64) string {
	maxInt := new(big.Int).SetInt64(upperBond)
	i, err := rand.Int(rand.Reader, maxInt)
	if err != nil {
		fmt.Printf("Can't generate random value: %v, %v", i, err)
	}
	outputRand := fmt.Sprintf("%v", i)
	return outputRand
}

func send_request(s_port uint, s_ip string, b_port uint, b_ip string, tx_num uint, from_id uint, s_num uint, cross_rate float64) {
	ctx_num := uint(float64(tx_num) * cross_rate)
	for {
		for i := uint(0); i < ctx_num; i++ {
			http.Get(fmt.Sprintf("http://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=EFGH,to=WXYZ,value=10,data=NONE,nonce=%v\"", s_ip, s_port, from_id, get_rand(int64(s_num)), 1, get_rand(math.MaxInt32)))
		}
		for i := uint(0); i < tx_num-ctx_num; i++ {
			http.Get(fmt.Sprintf("http://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=ABCD,to=EFGH,value=10,data=NONE,nonce=%v\"", s_ip, s_port, from_id, get_rand(int64(s_num)), 0, get_rand(math.MaxInt32)))
		}
	}
}
