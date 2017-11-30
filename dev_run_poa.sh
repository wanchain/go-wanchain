#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     


echo "run geth in pluto testnet"
mkdir -p ./data_pluto
echo "wanglu" > ./passwd.txt
if [ -d "DOCKER" ]; then
	cp -rf DOCKER/data/keystore ./data_pluto
else
	cp -rf ../data/keystore ./data_pluto
fi
#rm -rf ./data_pluto/geth/chaindata
#geth --datadir ./data_pluto init ./genesis_example/genesis_poa.json
networkid='--pluto'
./build/bin/geth ${networkid} --nat none --verbosity 4 --gasprice '200000' --datadir ./data_pluto  \
	 --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password ./passwd.txt \
	 --targetgaslimit 900000000  --port 17717  \
   	--rpc --rpcaddr 0.0.0.0 --rpcapi "eth,personal,net,admin" --rpccorsdomain '*' $@
