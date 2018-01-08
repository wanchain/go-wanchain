#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     


echo "run geth in testnet"
mkdir -p ./data_testnet
./build/bin/geth --testnet --txpool.nolocals --txpool.pricelimit 180000000000 --verbosity 4  --datadir ./data_testnet \
     --rpc --rpcaddr 0.0.0.0 --rpcapi "eth,personal,net,admin,wan" --rpccorsdomain '*' $@
