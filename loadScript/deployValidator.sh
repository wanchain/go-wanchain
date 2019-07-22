#!/bin/bash
# set -x
echo ''
echo ''
echo '=========================================='
echo '+     Welcome to Validator registion     +'
echo ''
echo 'Please Enter your validator Name:'
read YOUR_NODE_NAME

echo -e "\033[41;30m !!!!!! WARNING Please Remember Your Password !!!!!!!! \033[0m"
echo -e "\033[41;30m !!!!!!Otherwise You will lose all your assets!!!!!!!! \033[0m"
echo 'Enter your password of validator account:'
read -s PASSWD
echo 'Confirm your password of validator account:'
read -s PASSWD2
echo ''
DOCKERIMG=wanchain/client-go:2.1.0-beta
NETWORK=--testnet
NETWORKPATH=testnet

if [ ${PASSWD} != ${PASSWD2} ]
then
    echo 'Passwords mismatched'
    exit
fi

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

sudo docker run -d --name gwan -p 17717:17717 -p 17717:17717/udp -v /home/${USER}/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan ${NETWORK} --etherbase ${addrNew} --unlock ${addrNew} --password /root/.wanchain/pw.txt --mine --minerthreads=1 --wanstats ${YOUR_NODE_NAME}:admin@54.193.4.239:80

echo 'Please wait a few seconds...'

sleep 5

sudo rm /home/${USER}/.wanchain/pw.txt

KEYSTOREFILE=$(sudo ls /home/${USER}/.wanchain/testnet/keystore/)

KEYSTORE=$(sudo cat /home/${USER}/.wanchain/testnet/keystore/${KEYSTOREFILE})

echo ''
echo ''
echo -e "\033[41;30m !!!!!!!!!!!!!!! Important !!!!!!!!!!!!!!! \033[0m"
echo '=================================================='
echo '      Please Backup Your Validator Address'
echo '     ' ${ADDR}
echo '=================================================='
echo '      Please Backup Your Validator Public Key'
echo ${PK}
echo '=================================================='
echo '      Please Backup Your Keystore JSON String'
echo ''
echo ${KEYSTORE}
echo ''
echo '=================================================='
echo ''

if [ $(ps -ef | grep -c "gwan") -gt 1 ]; 
then 
    echo "Validator Start Successfully";
else
    echo "Validator Start Failed";
    echo "Please use command 'sudo docker logs gwan' to check reason." 
fi
