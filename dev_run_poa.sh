#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     


echo "run geth in pluto testnet"
mkdir -p /wanchain/src/data_pluto
echo "wanglu" > /tmp/passwd.txt
cp -rf /wanchain/data/keystore /wanchain/src/data_pluto
#rm -rf /wanchain/src/data_pluto/geth/chaindata
#geth --datadir /wanchain/src/data_pluto init ./genesis_example/genesis_poa.json
networkid='--pluto'
/wanchain/src/build/bin/geth ${networkid} --nat none --verbosity 4 --gasprice '200000' --datadir /wanchain/src/data_pluto  \
	--port 30303  \
   	--rpc --rpcaddr 0.0.0.0 --rpcapi "eth,personal,net,admin" --rpccorsdomain '*' 
