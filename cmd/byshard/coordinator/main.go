package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/viper"

	abciclient "github.com/tendermint/tendermint/abci/client"
	cfg "github.com/tendermint/tendermint/config"
	byshardapp "github.com/tendermint/tendermint/csconsensus/byshard/coordinator/abci"
	byshardnode "github.com/tendermint/tendermint/csconsensus/byshard/coordinator/validator"
	tmlog "github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/types"
	hctypes "github.com/tendermint/tendermint/types"
)

var homeDir, isLeader, shardPorts, beaconIp, shardIps string
var beaconPort, shardNum, shardid, attackShard, isByzantine, nodeID uint

// var isLeader bool

func init() {
	flag.StringVar(&homeDir, "home", "", "Path to the tendermint config directory (if empty, uses $HOME/.tendermint)")
	flag.StringVar(&isLeader, "leader", "false", "Is it a leader (default: false)")

	flag.UintVar(&shardNum, "shards", 2, "the number of shards")
	flag.UintVar(&shardid, "shardid", 0, "shard id")
	flag.UintVar(&beaconPort, "beaconport", 10057, "beacon chain port")
	flag.UintVar(&isByzantine, "byzantine", 0, "Is it byzantine (default: false)")
	flag.UintVar(&attackShard, "attackid", 0, "attack shard id")
	flag.UintVar(&nodeID, "nodeid", 0, "node id")

	flag.StringVar(&shardPorts, "shardports", "20057,21057", "shards chain port")
	flag.StringVar(&beaconIp, "beaconip", "127.0.0.1", "beacon chain ip")
	flag.StringVar(&shardIps, "shardips", "127.0.0.1,127.0.0.1", "shards chain ip")
}

func main() {
	flag.Parse()
	if homeDir == "" {
		homeDir = os.ExpandEnv("$HOME/.tendermint")
	}
	config := cfg.DefaultValidatorConfig()

	config.SetRoot(homeDir)

	viper.SetConfigFile(fmt.Sprintf("%s/%s", homeDir, "config/config.toml"))
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Reading config: %v", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		log.Fatalf("Decoding config: %v", err)
	}
	if err := config.ValidateBasic(); err != nil {
		log.Fatalf("Invalid configuration data: %v", err)
	}
	gf, err := types.GenesisDocFromFile(config.GenesisFile())
	if err != nil {
		log.Fatalf("Loading genesis document: %v", err)
	}

	dbPath := filepath.Join(homeDir, "leveldb")
	// fmt.Println("path database is: " + dbPath)
	db := byshardnode.NewBlockchainState("leveldb", dbPath)
	// db, err := dbm.NewGoLevelDBWithOpts
	// db := dbm.NewMemDB()
	// db, err := badger.Open(badger.DefaultOptions(dbPath))
	// if err != nil {
	// 	log.Fatalf("Opening database: %v", err)
	// }
	defer func() {
		if err := db.Database.Close(); err != nil {
			log.Fatalf("Closing database: %v", err)
		}
	}()
	var validatorInterface *byshardnode.ValidatorInterface
	in_ip_temp := hctypes.HaechiAddress{
		Ip:   hctypes.BytesToIp([]byte(beaconIp)),
		Port: uint16(beaconPort),
	}
	out_ips_temps := make([]hctypes.HaechiAddress, shardNum)
	out_ports_temp := []byte(shardPorts)
	out_ports := bytes.Split(out_ports_temp, []byte(","))
	out_ips_temp := []byte(shardIps)
	out_ips := bytes.Split(out_ips_temp, []byte(","))
	for i := 0; i < int(shardNum); i++ {
		temp_value64, _ := strconv.ParseUint(string(out_ports[i]), 10, 64)
		out_ips_temps[i] = hctypes.HaechiAddress{
			Ip:   hctypes.BytesToIp(out_ips[i]),
			Port: uint16(temp_value64),
		}
	}
	if isLeader == "true" {
		validatorInterface = byshardnode.NewValidatorInterface(db, uint8(shardNum), uint8(shardid), true, in_ip_temp, out_ips_temps, uint8(isByzantine), uint8(attackShard), uint8(nodeID))
	} else if isLeader == "false" {
		validatorInterface = byshardnode.NewValidatorInterface(db, uint8(shardNum), uint8(shardid), false, in_ip_temp, out_ips_temps, uint8(isByzantine), uint8(attackShard), uint8(nodeID))
	}

	app := byshardapp.NewbyshardApplication(validatorInterface)
	acc := abciclient.NewLocalCreator(app)

	logger := tmlog.MustNewDefaultLogger(tmlog.LogFormatPlain, tmlog.LogLevelInfo, false)
	node, err := nm.New(config, logger, acc, gf)
	if err != nil {
		log.Fatalf("Creating node: %v", err)
	}

	node.Start()
	defer func() {
		node.Stop()
		node.Wait()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
