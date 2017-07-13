#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     

echo "build geth..."

cd /wanchain/src
make clean
make geth
cp ./build/bin/geth /usr/local/bin

echo "run geth"
geth --verbosity 6 --datadir /wanchain/data --etherbase '0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e' --networkid 5201314 --mine --minerthreads 1 --nodiscover --rpc --rpcaddr 0.0.0.0
