#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     

echo "run gwan in pluto bootnode testnet"
mkdir -p ~/Library/Wanchain/pluto/keystore
mkdir -p ~/Library/Wanchain/pluto/gwan
dir=$(dirname $0)
cp ${dir}/nodekey ~/Library/Wanchain/pluto/gwan/
cp ${dir}/UTC* ~/Library/Wanchain/pluto/keystore
echo 'wanglu' > /tmp/pw.txt
build/bin/gwan --pluto --ipcpath ~/Library/Wanchain/gwan.ipc --nodiscover --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password /tmp/pw.txt  --mine --minerthreads=1 --verbosity 4

