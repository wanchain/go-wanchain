#!/bin/bash

echo -n ${KEYSTOREPW} > /dev/shm/pw.txt
./build/bin/faucet  \
-account.json ./data/keystore/UTC--2017-05-14T03-13-33.929385593Z--2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e \
-account.pass /dev/shm/pw.txt  -apiport 8080 \
-bootnodes enode://6b6e6ab0d3e25ce337d98af19fc7bdad26ea02eb4605f6033e370132cabbf9cd068dd703aa577d5c1b07f68e165ae22a31bcaa0e62f50bb74ac560923fc3a46b@127.0.0.1:30303 \
-ethport 30303  -network 7 \
-genesis ./genesis_example/genesis.json 
#sleep 10
#rm -rf /dev/shm/pw.txt

