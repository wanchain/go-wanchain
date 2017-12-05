#!/bin/sh

#   __        ___    _   _  ____ _           _       ____
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V /
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/
#

if [ $# != 2 ] ; then
    echo "input 1: $1 log dir"
    echo "input 2: $2 data dir"
    exit 1;
fi

logDir=$1
if [ ! $logDir ] ; then
    echo "log dir is not input"
fi

dataDir=$2
if [ ! $dataDir ] ; then
    echo "data dir is not input"
fi

SRCDIR="$(pwd)"

bakDate=`date +%Y%m%d`
echo $bakDate

ipAddress=$(ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:")
echo $ipAddress

ipStr=`echo $ipAddress | cut -c1-16`

echo $ipStr

bakLogDir="$SRCDIR/backup/log"
bakDataDir="$SRCDIR/backup/data/$ipStr-$bakDate"

echo $bakDataDir
echo $bakDataDir

mkdir -p $bakLogDir
mkdir -p $bakDataDir

cp $logDir"/running.log" $bakLogDir"/"$ipStr"-"$bakDate".log"

cp -r ~/.wanchain/* $bakDataDir
