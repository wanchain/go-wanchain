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

sudo docker stop gwan

echo ${PASSWD} | sudo tee -a ~/.wanchain/pw.txt > /dev/null

sudo docker start gwan

echo 'Please wait a few seconds...'

sleep 5

sudo rm ~/.wanchain/pw.txt

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

