package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	// "time"
)

// ./build/ahllatency -shardport "20157" -shardip "18.188.221.188"

var shardPort, shardIp, beaconPort, beaconIp, testInfo string
var victimShard, targetShard, attackNum, isIntra uint

func init() {
	flag.StringVar(&shardPort, "shardport", "22057", "shards chain port")
	flag.StringVar(&shardIp, "shardip", "127.0.0.1", "shards chain ip")
	flag.StringVar(&beaconPort, "beaconport", "10057", "beacon chain port")
	flag.StringVar(&beaconIp, "beaconip", "127.0.0.1", "beacon chain ip")
	flag.StringVar(&testInfo, "info", "test", "testing information")
	flag.UintVar(&victimShard, "victimid", 1, "victim shard id") // sender shard
	flag.UintVar(&targetShard, "targetid", 1, "target shard id") // contract shard
	flag.UintVar(&attackNum, "attacknum", 10, "the number of attacking tx")
	flag.UintVar(&isIntra, "intra", 1, "attack with intra-shard tx") // 1: intra-shard front-running tx; 0: cross-shard front-running tx
}

func main() {
	flag.Parse()
	fmt.Println("test info:", testInfo)
	nonce := get_rand(100)
	if isIntra == 1 { // attack with intra-shard front-running txs
		for i := uint(0); i < attackNum; i++ { // value: trace victim transactions
			http.Get(fmt.Sprintf("http://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=VCTM,to=INTR,value=%v,data=NONE,nonce=%v,txid=%v\"", beaconIp, beaconPort, victimShard, targetShard, 1, i, nonce, get_rand(20000)))
		}
	} else { // attack with intra-shard front-running txs
		for i := uint(0); i < attackNum; i++ { // value: trace victim transactions
			http.Get(fmt.Sprintf("http://%v:%v/broadcast_tx_commit?tx=\"fromid=%v,toid=%v,type=%v,from=VCTM,to=CROS,value=%v,data=NONE,nonce=%v,txid=%v\"", beaconIp, beaconPort, victimShard, targetShard, 1, i, nonce, get_rand(20000)))
		}
	}

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
