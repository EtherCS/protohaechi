package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"time"
)

// ./build/ahluser -shards 4 -size 30 -beaconport 10057 -beaconip "127.0.0.1" -shardports "20057" -shardips "127.0.0.1" -parallel 300 -duration 30
var shardNum, shardSize, batchSize, beaconPort, concurrentNum, reqDuration uint

var crossRate float64
var shardPorts, beaconIp, shardIps string

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
}

func main() {
	flag.Parse()
	// initialize ip
	shard_ports_temp := []byte(shardPorts)
	shard_ports := bytes.Split(shard_ports_temp, []byte(","))
	shard_ips_temp := []byte(shardIps)
	shard_ips := bytes.Split(shard_ips_temp, []byte(","))
	var ports_value64 []uint64
	var beaconports_value64 []uint64
	for _, shard_port := range shard_ports {
		temp_port, _ := strconv.ParseUint(string(shard_port), 10, 64)
		// each node in a shard
		for i := 0; i < int(shardSize); i++ {
			// fmt.Println("AHL: listening port", temp_port+100*uint64(i))
			ports_value64 = append(ports_value64, temp_port+100*uint64(i))
			beaconports_value64 = append(beaconports_value64, 10057+100*uint64(i))
		}

	}
	fmt.Println("AHL: client size is", concurrentNum)
	fmt.Println("AHL: the size of nodes receiving requests is", len(ports_value64))
	for p := 0; p < int(concurrentNum); p++ {
		for i, _ := range shard_ips {
			for j, _ := range ports_value64 {
				// fmt.Println("AHL: sending txs to", string(shard_ips[i])+":"+fmt.Sprint(ports_value64[j]))
				go send_request(uint(ports_value64[j]), string(shard_ips[i]), uint(beaconports_value64[j]), beaconIp, batchSize, uint(i), shardNum, crossRate)
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
			http.Get(fmt.Sprintf("http://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=EFGH,to=WXYZ,value=10,data=NONE,nonce=%v,txid=%v\"", b_ip, b_port, from_id, get_rand(int64(s_num)), 1, get_rand(math.MaxInt64), get_rand(20000)))
		}
		for i := uint(0); i < tx_num-ctx_num; i++ {
			http.Get(fmt.Sprintf("http://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=EFGH,to=WXYZ,value=10,data=NONE,nonce=%v,txid=%v\"", s_ip, s_port, from_id, get_rand(int64(s_num)), 0, get_rand(math.MaxInt64), i))
		}
	}
}

func tcp_send_request(conn net.Conn, s_port uint, s_ip string, b_port uint, b_ip string, tx_num uint, from_id uint, s_num uint, cross_rate float64, done chan string) {
	ctx_num := uint(float64(tx_num) * cross_rate)
	for {

		var e error
		for i := uint(0); i < tx_num-ctx_num; i++ {
			_, e = conn.Write([]byte(fmt.Sprintf("tcp://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=EFGH,to=WXYZ,value=10,data=NONE,nonce=%v,txid=%v\"", s_ip, s_port, from_id, get_rand(int64(s_num)), 0, get_rand(math.MaxInt64), i) + "\n"))

			if e != nil {
				break
			}
		}
		for i := uint(0); i < ctx_num; i++ {
			_, e = conn.Write([]byte(fmt.Sprintf("tcp://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=EFGH,to=WXYZ,value=10,data=NONE,nonce=%v,txid=%v\"", b_ip, b_port, from_id, get_rand(int64(s_num)), 1, get_rand(math.MaxInt64), get_rand(20000)) + "\n"))

			if e != nil {
				break
			}
		}
		if e != nil {
			fmt.Println("Error to send message because of ", e.Error())
			break
		}
	}
	done <- "done for " + s_ip
}
