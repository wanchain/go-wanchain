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
DATADIR=./data_privatenet2
ACCOUNT=0x184bfe537380d650533846c8c7e2a80d75acee63
KEYSTORE=./accounts/keystore/${ACCOUNT}
NETWORKID='--networkid 99'
# Cleanup the datadir
CLEANUP=true

# Perform cleanup
if [ -d $DATADIR ]
then
  if [ "$CLEANUP" == "true" ]
  then
    rm -rf $DATADIR
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

echo "password1" > ./passwd.txt
cp $KEYSTORE $DATADIR/keystore

PORT=17718
RPCPORT=8546

./build/bin/gwan ${NETWORKID} --etherbase "${ACCOUNT}" --nat none --verbosity 3 --gasprice '200000' --datadir $DATADIR  \
         --unlock "${ACCOUNT}" --password ./passwd.txt \
         --targetgaslimit 900000000  --port ${PORT} --mine --minerthreads 1 \
         --maxpeers 5 --nodekey ./bootnode/privatenet2 \
        --rpc --rpcaddr 0.0.0.0 --rpcport ${RPCPORT} --rpcapi "eth,personal,net,admin" --rpccorsdomain '*' \
        --identity "LocalTestNode2" --bootnodes "enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@127.0.0.1:17718"
