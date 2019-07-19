#!/bin/bash
# set -x
echo ''
echo ''
echo ''
echo ''
echo '=========================================='
echo '+     Welcome to Validator Update        +'
echo ''
echo 'If you have deployed your validator with deployValidator.sh, you can update with this script'
echo 'We will stop all your'
echo 'Please Enter your password of Validator account:'
read PASSWD
echo ''
echo ''
echo ''
echo ''
echo ''

sudo docker stop gwan

echo ${PASSWD} | sudo tee -a /home/${USER}/.wanchain/pw.txt > /dev/null

sudo docker start gwan

echo 'Please wait a few seconds...'

sleep 5

sudo rm /home/${USER}/.wanchain/pw.txt

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

