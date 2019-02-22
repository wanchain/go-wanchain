#!/bin/bash

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#
# Run this file from the go-wanchain directory with ./dev_run_privatenet.sh
# First build gwan with make gwan
#
DATADIR=./data_privatenet
ACCOUNT1=0x68489694189aa9081567dfc6d74a08c0c21d92c6
ACCOUNT2=0x184bfe537380d650533846c8c7e2a80d75acee63
KEYSTORE1=./accounts/keystore/${ACCOUNT1}
KEYSTORE2=./accounts/keystore/${ACCOUNT2}
# IPCPATH=[fill in your IPC path here  to use wanwalletgui with your node]

# Cleanup the datadir
CLEANUP=true

NETWORKID='--networkid 99'

# Perform cleanup
if [ -d $DATADIR ]
then
  if [ "$CLEANUP" == "true" ]
  then
    rm -rf $DATADIR
    # Initialize chain
    ./build/bin/gwan ${NETWORKID} --etherbase "${ACCOUNT1}" --nat none --verbosity 4 \
      --datadir $DATADIR --identity LocalTestNode init ./core/genesis_privatenet.json
  fi
else
  mkdir $DATADIR
fi

if [ ! -d $DATADIR/keystore ]
then
  mkdir -p $DATADIR/keystore
fi

echo "password1" > ./passwd.txt
echo "password1" >> ./passwd.txt
cp $KEYSTORE1 $DATADIR/keystore
cp $KEYSTORE2 $DATADIR/keystore

PORT=17718
RPCPORT=8546

./build/bin/gwan ${NETWORKID} --etherbase "${ACCOUNT1}" --nat none --gasprice '200000' --verbosity 4 --datadir $DATADIR  \
    --unlock "${ACCOUNT1},${ACCOUNT2}" --password ./passwd.txt \
    --port ${PORT} --mine --minerthreads 1 \
    --maxpeers 5 --nodiscover --nodekey ./bootnode/privatenet1 \
    --rpc --rpcaddr 0.0.0.0 --rpcport ${RPCPORT} --rpcapi "eth,personal,net,admin,wan" --rpccorsdomain '*' \
    --bootnodes "enode://f9b91ac38231ecfb12564a006ad6d97d5d1bdaf1a74134fc4cf08e1d7151f7e18e00181cb94347ce3272d6af79d1ebc8b8bf50ce50b40b013a8ff9cf16ff034a@127.0.0.1:17719" \
    --ipcpath ${IPCPATH} \
    --keystore $DATADIR/keystore \
    --identity LocalTestNode
