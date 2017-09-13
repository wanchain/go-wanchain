#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     


echo "run geth in pluto bootnode testnet"
mkdir -p ./data_pluto/keystore
echo "wanglu" > ./passwd.txt
cp -rf ./UTC* ./data_pluto/keystore/
#rm -rf /wanchain/src/data_pluto/geth/chaindata
#geth --datadir /wanchain/src/data_pluto init ./genesis_example/genesis_poa.json

networkid='--pluto'
../build/bin/geth ${networkid}  --nodekey ./nodekey --verbosity 4 --gasprice '200000' --datadir ./data_pluto  \
	--mine --minerthreads 1 \
	--nat none \
	 --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password ./passwd.txt \
	 --targetgaslimit 900000000  --port 30303  \
	--etherbase '0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e'   --rpc --rpcaddr 0.0.0.0 --rpcapi "eth,personal,net,admin" --rpccorsdomain '*' 
