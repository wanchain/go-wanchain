#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     


echo "run geth in pluto bootnode testnet"
echo -n ${COINBASEPW} > /tmp/pw.txt
geth --datadir=./data --pluto --nodiscover --unlock "0xe8ffc3d0c02c0bfc39b139fa49e2c5475f000000" --password /tmp/pw.txt \
--nodekey nodekey.key --mine --minerthreads=1
