#!/bin/sh
# back up go-wanchain use admin.exportChain


#   __        ___    _   _  ____ _           _       ____             

#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __

#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /

#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 

#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  

# 
GOPATH=/home/lizhan/lizhan/gopath
tarDir=$GOPATH/../backup/backupChain
log=$tarDir/log.txt
ipcDir=/home/lizhan/.wanchain/testnet/gwan.ipc

wanChainDir=$GOPATH/src/github.com/wanchain/go-wanchain
backupNum=5

cd $wanChainDir
echo "**************************************" >> $log
echo "****** go-wanchain Backup begin ******" >> $log
echo "**************************************" >> $log

DATE=`date '+%Y%m%d-%H%M%S'`
backupChainName=$DATE"-wanchain"

echo " *** BACKUPTIME: " $DATE >> $log

echo " " >> $log

echo "admin.exportChain('$tarDir/$backupChainName')" | ./build/bin/geth attach ipc:$ipcDir exit >> $log
echo " *** BACKUP Chain Name: " $backupChainName >> $log

if [ $(ls $tarDir -l | grep "wanchain" | wc -l) -gt $backupNum ]
then

    echo "The backup number alredy reache the largest backupNum " $backupNum " in folder " $tarDir ", follow files would be rm" >> $log
    ls $tarDir -rt | head -n1 >> $log
    cd $tarDir
    rm -r $(ls $tarDir -rt | head -n1) >> $log
fi

echo "****** go-wanchain Backup end******" >> $log
