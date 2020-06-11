#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     

echo "build gwan..."

mkdir -p ~/wandev/data/keystore
mkdir -p ~/wandev/bin

cp ./bootnode/UTC--2017-05-14T03-13-33.929385593Z--2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e ~/wandev/data/keystore

echo -n 'wanglu' >  ~/wandev/pwdfile

make clean

make gwan

cp ./build/bin/gwan ~/wandev/bin/

cd ~/wandev/

echo "run gwan"

./bin/gwan -verbosity 3 --plutodev --nodiscover --datadir ./data --etherbase 0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e --unlock 0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e --password ./pwdfile --rpc --rpcaddr 0.0.0.0 --mine --minerthreads 1 --rpcport 8888 --port 18546 --rpcapi eth,personal,net,admin,wan,txpool,pos $@
