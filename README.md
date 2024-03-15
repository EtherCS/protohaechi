# Haechi
This repo provides an implementation of [Haechi](https://arxiv.org/pdf/2306.06299.pdf) in Go. The codebase forks from [Tendermint](https://github.com/tendermint/tendermint/tree/main) project. To obtain the measured metrics easily, we keep the source codes of Tendermint and inject minimum codes for code tracing. For quick revision and modification, we list the file directories of major implementation, configuration and scripts below:

- Different cross-shard protocols: [./csconsensus](https://github.com/EtherCS/protohaechi/tree/main/csconsensus).
- Clients: [./cmd](https://github.com/EtherCS/protohaechi/tree/main/cmd).
- Configuration files: [./configs](https://github.com/EtherCS/protohaechi/tree/main/configs).
- Scripts: [./scripts/haechi](https://github.com/EtherCS/protohaechi/tree/main/scripts/haechi).

> NOTE: If you have any question, feel free to contact antfinancialyujian@gmail.com.
> 
# Table of Contents

1 - [Minimum requirement](#minimum-requirements)

2 - [Installation instructions](#installation-instructions)

3 - [Testing instructions](#testing-instructions)

4 - [Experimental results](#experimental-results)

5 - [Parameter description](#parameter-description)

## Minimum requirements

| Requirement       | Note |
|-------------------|--------------------|
| go                 | 1.16      |

## Installation instructions

> NOTE: The path should be set correctly because our scripts rely on a relative path.

First, create the repo path. Under your *$HOME*, execute

```bash
$ mkdir go && cd go && mkdir src && cd src && mkdir github.com && cd github.com && mkdir EtherCS && cd EtherCS
```

Next, under the path *$HOME/go/src/github.com/EtherCS*, clone the repo and download the dependencies:

```bash
$ git clone git@github.com:EtherCS/protohaechi.git
$ cd protohaechi
$ go mod tidy
```

These command may take a long time the first time you run them.
## Testing instructions
Under the repo path (i.e., *$GOPATH/src/github.com/EtherCS/protohaechi*), follow the two steps:

### Step 1. Build

```bash
$ make build_test
```

### Step 2. Test

1. AHL

- run two shards with four nodes in a terminal

```bash
$ ./scripts/haechi/single-machine/run_ahl.sh
```

- run client in another terminal to send transactions

```bash
$ ./build/ahlclient
```

- run a latency client in another terminal for testing latency
  
```bash
$ ./build/ahllatency -beaconport "10057" -beaconip "127.0.0.1" -shardport "20157" -shardip "127.0.0.1" -info "ahl: latency test"
```

2. Byshard

- run two shards with four nodes in a terminal

```bash
$ ./scripts/haechi/single-machine/run_byshard.sh
```

- run client in another terminal to send transactions

```bash
$ ./build/byshardclient
```

- run a latency client in another terminal for testing latency
```bash
$ ./build/byshardlatency  -beaconport "10057" -beaconip "127.0.0.1" -shardport "20157" -shardip "127.0.0.1" -info "byshard: latency test"
```

3. Haechi-sync

- run two shards with four nodes in a terminal

```bash
$ ./scripts/haechi/single-machine/run_haechi_sync.sh
```

- run client in another terminal to send transactions

```bash
$ ./build/haechisyncclient
```

- run a latency client in another terminal for testing latency
```bash
$ ./build/haechisynclatency  -beaconport "10057" -beaconip "127.0.0.1" -shardport "20157" -shardip "127.0.0.1" -info "haechi-sync: latency test"
```

4. Haechi

- run two shards with four nodes in a terminal

```bash
$ ./scripts/haechi/single-machine/run_haechi.sh
```

- run client in another terminal to send transactions

```bash
$ ./build/haechiclient
```

- run a latency client in another terminal for testing latency
```bash
$ ./build/haechilatency -beaconport "10057" -beaconip "127.0.0.1" -shardport "20157" -shardip "127.0.0.1" -info "haechi: latency test"
```
## Experimental results

After running all tests under different parameters, experimental results can be found in the log files ***./tmplog*** with a specific file name. A sample log is:
```
Haechi: start a block at time 2023-06-06 11:04:38.350497 -0400 EDT m=+46.775439376
2023-06-06T11:04:38-04:00 INFO received proposal module=consensus proposal={"Type":32,"block_id":{"hash":"2DCA1844D7FA7BD768787C8C32E9360D3E43C16B833FAA428C3352DE3CA4BDDB","parts":{"hash":"630199F85CC73C592A14A53AD9406695309472128FF32ABAF8D36613E3CC3148","total":1}},"height":13,"pol_round":-1,"round":0,"signature":"VICrY6DNRFU/ku+HHtwUgxz6aH22WXvH5x3VF6jPaULXAvqNG+BLAva7kNs9tzxe3KRKHwMSm9m6/cM90vPhAw==","timestamp":"2023-06-06T15:04:38.430812Z"}
2023-06-06T11:04:38-04:00 INFO received complete proposal block hash=2DCA1844D7FA7BD768787C8C32E9360D3E43C16B833FAA428C3352DE3CA4BDDB height=13 module=consensus
2023-06-06T11:04:40-04:00 INFO finalizing commit of block hash=2DCA1844D7FA7BD768787C8C32E9360D3E43C16B833FAA428C3352DE3CA4BDDB height=13 module=consensus num_txs=100 root=
2023-06-06T11:04:44-04:00 INFO executed block height=13 module=state num_invalid_txs=0 num_valid_txs=100
2023-06-06T11:04:44-04:00 INFO committed state app_hash= height=13 module=state num_txs=100
Haechi: commit block at time 2023-06-06 11:04:44.293434 -0400 EDT m=+52.718476153
```

## Parameter description

#### node parameter description

| node parameters | description        | default               |
|-----------------|--------------------|-----------------------|
| n               | shard number       | 2                     |
| m               | shard size         | 2                     |
| p               | beacon chain port  | 10057                 |
| i               | beacon chain ip    | "127.0.0.1"           |
| s               | shard chain points | "20057,21057"         |
| x               | shard chain ips    | "127.0.0.1,127.0.0.1" |

#### client parameter description

| client parameters | description               | default               |
|-------------------|---------------------------|-----------------------|
| shards            | shard number              | 2                     |
| size              | shard size                | 2                     |
| beaconport        | beacon chain port         | 10057                 |
| beaconip          | beacon chain ip           | "127.0.0.1"           |
| shardports        | shard chain points        | "20057,21057"         |
| shardips          | shard chain ips           | "127.0.0.2,127.0.0.3" |
| batch             | batch size per request    | 10                    |
| ratio             | cross-shard txs ratio     | 0.8                   |
| parallel          | concurrent request number | 100                   |
| duration          | execution duration, s     | 120                   |
<!-- | rate              | sleeping time, ms         | 100                   | -->
*Note: the default maximum number of connections of a node is 900*

#### script parameter description

| script parameters | description        | default                       |
|-------------------|--------------------|-------------------------------|
| h                 | config path        | configs/EC2-test/shard/30node |
| n                 | shard number       | 2                             |
| m                 | shard size         | 2                             |
| p                 | beacon chain port  | 10057                         |
| i                 | beacon chain ip    | "127.0.0.1"                   |
| s                 | shard chain points | "20057,21057"                 |
| x                 | shard chain ips    | "127.0.0.1,127.0.0.1"         |
| d                 | testing time       | 120                           |