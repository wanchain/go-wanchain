#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     

echo "run gwan in pluto bootnode testnet"
make && \
rm -rf ~/.wanchain/internal/gwan && \
#build/bin/gwan --nodiscover --internal --port 18822 --syncmode full  --rpc --rpcaddr=0.0.0.0 –rpcapi=’eth,wan,pos,personal,txpool’ --nodiscover --etherbase  "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"  --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password ./pw.txt  --mine --minerthreads=1 $@
build/bin/gwan --nodiscover --internal --port 18822 --rpc --rpcapi='eth,personal,txpool,pos'   --etherbase  "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"  --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password ./pw.txt  --mine --minerthreads=1 $@

