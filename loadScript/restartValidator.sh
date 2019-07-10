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
read PASSWD
echo ''
echo ''
echo ''
echo ''
echo ''

sudo docker stop gwan

echo ${PASSWD} | sudo tee -a /home/${USER}/.wanchain/pw.txt > /dev/null

addrNew=`echo ${ADDR} | sed 's/.\(.*\)/\1/' | sed 's/\(.*\)./\1/'`

sudo docker start gwan

echo 'Please wait a few seconds...'

sleep 5

sudo rm /home/${USER}/.wanchain/pw.txt

echo ''
echo ''
echo ''
echo ''

