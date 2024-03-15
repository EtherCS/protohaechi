source ~/.bashrc
export GOPATH=$HOME/go
GOSRC=$GOPATH/src
ROOT=$GOSRC/github.com/EtherCS/haechi

mkdir -p build

go build -o build/ahlbc $ROOT/cmd/ahl/beacon
go build -o build/ahlshard $ROOT/cmd/ahl/shard
go build -o build/ahllatency $ROOT/cmd/latency-client/ahl
go build -o build/ahlclient $ROOT/cmd/ahl/client
go build -o build/ahluser $ROOT/cmd/user-client/ahl
go build -o build/ahlattack $ROOT/cmd/attack-client/ahl

go build -o build/byshard $ROOT/cmd/byshard/coordinator
go build -o build/byshardclient $ROOT/cmd/byshard/client
go build -o build/byshardlatency $ROOT/cmd/latency-client/byshard
go build -o build/bysharduser $ROOT/cmd/user-client/byshard
go build -o build/byshardattack $ROOT/cmd/attack-client/byshard

go build -o build/haechibc $ROOT/cmd/haechi/beacon
go build -o build/haechishard $ROOT/cmd/haechi/shard
go build -o build/haechiclient $ROOT/cmd/haechi/client
go build -o build/haechilatency $ROOT/cmd/latency-client/haechi
go build -o build/haechiuser $ROOT/cmd/user-client/haechi
go build -o build/haechiattack $ROOT/cmd/attack-client/haechi

go build -o build/haechisyncbc $ROOT/cmd/haechi-sync/beacon
go build -o build/haechisyncshard $ROOT/cmd/haechi-sync/shard
go build -o build/haechisyncclient $ROOT/cmd/haechi-sync/client
go build -o build/haechisynclatency $ROOT/cmd/latency-client/haechi-sync
go build -o build/haechisyncuser $ROOT/cmd/user-client/haechi-sync
go build -o build/haechisyncattack $ROOT/cmd/attack-client/haechi-sync

# go build -o build/haechiclient_haechi_2shard $ROOT/cmd/haechi/client_haechi_2shard
# go build -o build/haechiclient_haechi_4shard $ROOT/cmd/haechi/client_haechi_4shard
# go build -o build/haechiclient_haechi_6shard $ROOT/cmd/haechi/client_haechi_6shard
# go build -o build/haechiclient_haechi_8shard $ROOT/cmd/haechi/client_haechi_8shard
# go build -o build/haechiclient_haechi_10shard $ROOT/cmd/haechi/client_haechi_10shard
# go build -o build/haechiclient_haechi_14shard $ROOT/cmd/haechi/client_haechi_14shard
# go build -o build/haechiclient_haechi_16shard $ROOT/cmd/haechi/client_haechi_16shard

chmod +x build/*
