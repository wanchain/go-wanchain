#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     
# Run this file from the go-wanchain directory with ./dev_run_privatenet.sh
# First build gwan with make gwan
#
set -x
DATADIR=./data_privatenet3
# IPCPATH=[fill in your IPC path here  to use wanwalletgui with your node]
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
    ./build/bin/gwan ${NETWORKID} --nat none --verbosity 4 \
      --datadir $DATADIR --identity LocalTestNode3 init ./core/genesis_privatenet.json
  fi
else
  mkdir $DATADIR
fi

PORT=17717
RPCPORT=8545

./build/bin/gwan ${NETWORKID} --nat none --verbosity 3 --gasprice '200000' --datadir $DATADIR  \
    --port ${PORT} \
    --maxpeers 5 \
    --rpc --rpcaddr 0.0.0.0 --rpcport ${RPCPORT} --rpcapi "eth,personal,net,admin,wan" --rpccorsdomain '*' \
    --bootnodes "enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@127.0.0.1:17718" \
    --ipcpath ${IPCPATH} \
    --keystore $DATADIR/keystore \
    --identity LocalTestNode3
