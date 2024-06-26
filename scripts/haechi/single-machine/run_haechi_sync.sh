source ~/.bashrc
export GOPATH=$HOME/go
GOSRC=$GOPATH/src
TEST_SCENE="haechisync"
TM_HOME="$HOME/.haechiOrder"
WORKSPACE="$GOSRC/github.com/EtherCS/protohaechi"
CURRENT_DATE=`date +"%Y-%m-%d-%H-%M"`
LOG_DIR="$WORKSPACE/tmplog/$TEST_SCENE-$CURRENT_DATE"
DURATION=120

SHARD_NUM=2
SHARD_SIZE=2
BEACON_PORT=10057
BEACON_IP="127.0.0.1"
SHARD_PORTS="20057,21057"
SHARD_IPS="127.0.0.1,127.0.0.1"

while getopts ":n:m:p:i:s:x:d:" opt
do 
    case $opt in
    n) # shard number
        echo "shard number is $OPTARG"
        SHARD_NUM=$OPTARG
        ;;
    m) # shard size
        echo "shard size is $OPTARG"
        SHARD_SIZE=$OPTARG
        ;;  
    p) # beacon port
        echo "beaconport is $OPTARG"
        BEACON_PORT=$OPTARG
        ;;  
    i) # beacon ip
        echo "beaconip is $OPTARG"
        BEACON_IP=$OPTARG
        ;;  
    s) # shard ports
        echo "shardports is $OPTARG"
        SHARD_PORTS=$OPTARG
        ;;  
    x) # shard ips
        echo "shardips is $OPTARG"  
        SHARD_IPS=$OPTARG
        ;;
    d) # executing duration
        echo "duration is $OPTARG"  
        DURATION=$OPTARG
        ;;  
    ?)  
        echo "unknown: $OPTARG"
        ;;
    esac
done

rm -rf $TM_HOME/*
mkdir -p $TM_HOME
mkdir -p $LOG_DIR

cp -r configs/haechi/* $TM_HOME
echo "configs generated"

pkill -9 haechisyncbc
pkill -9 haechisyncshard

# run beacon node
for ((i=0;i<$SHARD_SIZE;i++))
do
    if [ $i -eq 0 ]; then
        echo "running beacon leader"
        ./build/haechisyncbc -home $TM_HOME/beacon/node$i -leader "true" -shards $SHARD_NUM -beaconport $BEACON_PORT -shardports $SHARD_PORTS -beaconip $BEACON_IP -shardips $SHARD_IPS &> $LOG_DIR/beaconnode$i.log &
    else
        echo "running beacon validator"
        ./build/haechisyncbc -home $TM_HOME/beacon/node$i -leader "false" -shards $SHARD_NUM -beaconport $BEACON_PORT -shardports $SHARD_PORTS -beaconip $BEACON_IP -shardips $SHARD_IPS &> $LOG_DIR/beaconnode$i.log &
    fi
    sleep 1
done

# run shard node
for ((j=0;j<$SHARD_NUM;j++))
do
    for ((k=0;k<$SHARD_SIZE;k++))
    do
        if [ $k -eq 0 ]; then
            echo "running shard$j leader"
            ./build/haechisyncshard -home $TM_HOME/shard$j/node$k -leader "true" -shards $SHARD_NUM -shardid $j -beaconport $BEACON_PORT -shardports $SHARD_PORTS -beaconip $BEACON_IP -shardips $SHARD_IPS &> $LOG_DIR/shard$j-node$k.log &
        else
            echo "running shard$j validator$k"
            ./build/haechisyncshard -home $TM_HOME/shard$j/node$k -leader "false" -shards $SHARD_NUM -shardid $j -beaconport $BEACON_PORT -shardports $SHARD_PORTS -beaconip $BEACON_IP -shardips $SHARD_IPS -byzantine 1 -attackid 1 -nodeid $j &> $LOG_DIR/shard$j-node$k.log &
        fi
    sleep 1
    done
done

echo "testnet launched"
echo "running for ${DURATION}s..."
sleep $DURATION
pkill -9 haechisyncbc
pkill -9 haechisyncshard
echo "all done"
