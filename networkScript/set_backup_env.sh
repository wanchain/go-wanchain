#!/bin/sh
# set up the crontab environment to backup wan-chain data


#   __        ___    _   _  ____ _           _       ____             

#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __

#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /

#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 

#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  

# 

cronfile=/tmp/crontab.${USER}
crontab -l > $cronfile
CURDIR="$(pwd)"
echo "1 */3 * * *" $CURDIR/"wanchainBackup.sh" >> $cronfile
crontab $cronfile
rm -rf $cronfile
