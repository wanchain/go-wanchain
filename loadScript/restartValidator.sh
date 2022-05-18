#!/bin/bash
# set -x
echo ''
echo ''
echo ''
echo ''
echo '=========================================='
echo '+     Welcome to Validator Restart       +'
echo ''
echo 'If you have deployed your validator with deployValidator.sh, you can restart with this script'
echo ''
echo 'Please Enter your password of Validator account:'
read -s PASSWD
echo ''
echo ''
echo ''
echo ''
echo ''
read -p "Do you want save your password to disk for auto restart? (N/y): " savepasswd

sudo docker stop gwan

echo ${PASSWD} | sudo tee ~/.wanchain/pw.txt > /dev/null

IPCFILE="$HOME/.wanchain/gwan.ipc"
if [ -d $HOME/.wanchain/testnet ]; then
    IPCFILE="$HOME/.wanchain/testnet/gwan.ipc"
fi
rm -f $IPCFILE

sudo docker start gwan

echo 'Please wait a few seconds...'

sleep 5

if [ "$savepasswd" == "Y" ] || [ "$savepasswd" == "y" ]; then
    sudo docker container update --restart=always gwan
else

    while true
    do
        if [ -e $IPCFILE ]; then
            cur=`date '+%s'`
            ft=`stat -c %Y $IPCFILE`
            if [ $cur -gt $((ft + 6)) ]; then
                break
            fi
        fi
        echo -n '.'
        sleep 1
    done
    sudo rm ~/.wanchain/pw.txt
fi

echo ''
echo ''
echo ''
echo ''

if [ $(ps -ef | grep -c "gwan") -gt 1 ]; 
then 
    echo "Validator Start Success";
else
    echo "Validator Start Failed";
    echo "Please use command 'sudo docker logs gwan' to check reason." 
fi

