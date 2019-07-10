#!/bin/bash
# set -x
echo ''
echo ''
echo ''
echo ''
echo '=========================================='
echo '+     Welcome to Validator registion     +'
echo ''
echo 'Please Enter your validator Name:'
read YOUR_NODE_NAME
echo 'Please Enter your password of Validator account:'
read PASSWD
echo ''
echo ''
echo ''
echo ''
echo ''
DOCKERIMG=wanchain/client-go:2.0.0-beta.5
NETWORK=--testnet
NETWORKPATH=testnet

sudo wget -qO- https://get.docker.com/ | sh
sudo usermod -aG docker ${USER}
sudo service docker start
sudo docker pull ${DOCKERIMG}

getAddr=$(sudo docker run -v /home/${USER}/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan ${NETWORK} console --exec "personal.newAccount('${PASSWD}')")

ADDR=$getAddr

echo $ADDR

getPK=$(sudo docker run -v /home/${USER}/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan ${NETWORK} console --exec "personal.showPublicKey(${ADDR},'${PASSWD}')")
PK=$getPK

echo $PK

echo ${PASSWD} | sudo tee -a /home/${USER}/.wanchain/pw.txt > /dev/null

addrNew=`echo ${ADDR} | sed 's/.\(.*\)/\1/' | sed 's/\(.*\)./\1/'`

sudo docker run --restart=always -d --name gwan -p 17717:17717 -p 17717:17717/udp -v /home/${USER}/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan ${NETWORK} --etherbase ${addrNew} --unlock ${addrNew} --password /root/.wanchain/pw.txt --mine --minerthreads=1 --wanstats ${YOUR_NODE_NAME}:admin@54.193.4.239:80

echo 'Please wait a few seconds...'

sleep 5

sudo rm /home/${USER}/.wanchain/pw.txt

KEYSTOREFILE=$(sudo ls /home/${USER}/.wanchain/testnet/keystore/)

KEYSTORE=$(sudo cat /home/${USER}/.wanchain/testnet/keystore/${KEYSTOREFILE})

echo ''
echo ''
echo ''
echo ''
echo ''
echo ''
echo -e "\033[41;37m !!!!!!!!!!!!!!!Important Backup!!!!!!!!!!!!!!! \033[0m"
echo '=================================================='
echo '      Please backup Your Validator Address'
echo '     ' ${ADDR}
echo '=================================================='
echo '      Please backup Your Validator Public Key'
echo ${PK}
echo '=================================================='
echo '      Please backup Your Password:' ${PASSWD}
echo '=================================================='
echo '      Please backup Your Keystore Json string'
echo ''
echo ${KEYSTORE}
echo ''
echo '=================================================='
echo -e "\033[41;37m !!!!!!!!!!!!!!!Important Backup!!!!!!!!!!!!!!! \033[0m"
echo ''
