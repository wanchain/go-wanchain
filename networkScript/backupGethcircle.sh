#!/bin/sh
# back up go-wanchain geth data


#   __        ___    _   _  ____ _           _       ____             

#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __

#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /

#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 

#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  

# 
GOPATH=/home/lizhan/lizhan/gopath
tarDir=$GOPATH/../backup/backupGeth
log=$tarDir/log.txt
wanChainDir=$GOPATH/src/github.com/wanchain/go-wanchain
backupNum=5

cd $wanChainDir
echo "************************************************" >> $log
echo "****** go-wanchain beth data Backup begin ******" >> $log
echo "************************************************" >> $log

DATE=`date '+%Y%m%d-%H%M%S'`
backupGethName=$DATE"-geth.tar"
echo " *** BACKUPTIME:" $DATE >> $log
	
cd data_testnet/geth
echo " " >> $log

echo "begin to tar geth file under " $wanChainDir/data_testnet/geth >> $log
echo " *** filelist *** " >> $log
tar czvf $tarDir/$backupGethName * >> $log 2>&1

if [ $? -eq 0 ]; then
    echo " *** BACKUP Geth data Name:" $backupGethName " Successsful">> $log
else
    echo " *** BACKUP Geth data Name:" $backupGethName " Fail!">> $log
fi

if [ $(ls $tarDir -l | grep "geth.tar" | wc -l) -gt $backupNum ]
then
    echo "the backup number in folder" $tarDir " is larger than " $backupNum ", follow files would be rm" >> $log
    ls $tarDir -rt | head -n1 >> $log
    cd $tarDir
    rm -r $(ls $tarDir -rt | head -n1) >> $log
fi

echo "****** go-wanchain Backup end******" >> $log
