#!/bin/sh

#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     

echo "run gwan in pluto bootnode testnet"
build/bin/gwan --pluto --nodiscover --etherbase  "0xcf696d8eea08a311780fb89b20d4f0895198a489"  --unlock "0xcf696d8eea08a311780fb89b20d4f0895198a489" --password ./pw.txt  --mine --minerthreads=1 $@

