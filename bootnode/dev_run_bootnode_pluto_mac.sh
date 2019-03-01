#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     

echo "run gwan in pluto bootnode testnet"
rm -rf ~/Library/Wanchain/pluto
mkdir -p ~/Library/Wanchain/pluto/keystore
dir=$(dirname $0)
#cp ${dir}/nodekey ~/Library/Wanchain/pluto/gwan/
cp ${dir}/UTC* ~/Library/Wanchain/pluto/keystore
echo 'wanglu' > /tmp/pw.txt

#0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e
rm -rf ~/Library/Wanchain/pluto/gwan
make && build/bin/gwan --pluto --rpc --ipcpath ~/Library/Wanchain/gwan.ipc --nodiscover --etherbase "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password /tmp/pw.txt --rpc  --mine --minerthreads=1 --verbosity 3 --syncmode "full"
#make && build/bin/gwan --pluto --rpc --ipcpath ~/Library/Wanchain/gwan.ipc --etherbase "0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8" --unlock "0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8" --password /tmp/pw.txt --rpc  --mine --minerthreads=1 --verbosity 3 --syncmode "full" --verbosity 3
#make && build/bin/gwan --nodiscover --pluto --rpc --ipcpath ~/Library/Wanchain/gwan.ipc --nodiscover --etherbase "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --unlock "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password /tmp/pw.txt --rpc  --mine --minerthreads=1 --verbosity 3 --syncmode "full"

