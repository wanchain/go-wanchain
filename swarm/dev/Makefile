.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/wanchain/go-wanchain/cmd/swarm
	go build -o bin/geth     github.com/wanchain/go-wanchain/cmd/geth
	go build -o bin/bootnode github.com/wanchain/go-wanchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/wanchain/go-wanchain/swarm/...
