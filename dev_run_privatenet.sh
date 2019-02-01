#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     
#
# Run this file from the go-wanchain directory with ./dev_run_privatenet.sh
# First build gwan with make gwan
#
set -x
DATADIR=./data_privatenet
ACCOUNT=0x68489694189aa9081567dfc6d74a08c0c21d92c6
KEYSTORE=./accounts/keystore/${ACCOUNT}
IPCPATH=setipcpathhereforwanwalletgui
# Cleanup the datadir
CLEANUP=false

NETWORKID='--networkid 99'

# Perform cleanup
if [ -d $DATADIR ]
then
  if [ "$CLEANUP" == "true" ]
  then
    rm -rf $DATADIR
    # Initialize chain
    ./build/bin/gwan ${NETWORKID} --etherbase "${ACCOUNT}" --nat none --verbosity 4 \
      --datadir $DATADIR --identity “LocalTestNode” init ./core/genesis_privatenet.json
  fi
else
  mkdir $DATADIR
fi

if [ ! -d $DATADIR/keystore ]
then
  mkdir -p $DATADIR/keystore
fi

PORT=17718
RPCPORT=8546

echo "password1" > ./passwd.txt
cp $KEYSTORE $DATADIR/keystore

./build/bin/gwan ${NETWORKID} --etherbase "${ACCOUNT}" --nat none --verbosity 3 --gasprice '200000' --datadir $DATADIR  \
    --unlock "${ACCOUNT}" --password ./passwd.txt \
    --port ${PORT} --mine --minerthreads 1 \
    --maxpeers 5 --nodiscover --nodekey ./bootnode/privatenet1 \
    --rpc --rpcaddr 0.0.0.0 --rpcport ${RPCPORT} --rpcapi "eth,personal,net,admin" --rpccorsdomain '*' \
    --bootnodes "enode://f9b91ac38231ecfb12564a006ad6d97d5d1bdaf1a74134fc4cf08e1d7151f7e18e00181cb94347ce3272d6af79d1ebc8b8bf50ce50b40b013a8ff9cf16ff034a@127.0.0.1:17719" \
    --ipcpath ${IPCPATH} \
    --identity “LocalTestNode”
