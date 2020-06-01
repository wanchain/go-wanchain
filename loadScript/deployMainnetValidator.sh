#!/bin/bash
# set -x
echo ''
echo ''
echo '=========================================='
echo '|  Welcome to Mainnet Validator Deploy   |'
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
DOCKERIMG=wanchain/client-go:2.1.5

if [ ${PASSWD} != ${PASSWD2} ]
then
    echo 'Passwords mismatched'
    exit
fi

sudo wget -qO- https://get.docker.com/ | sh
sudo usermod -aG docker ${USER}
if [ $? -ne 0 ]; then
    echo "sudo usermod -aG docker ${USER} failed"
    exit 1
fi

sudo service docker start
if [ $? -ne 0 ]; then
    echo "service docker start failed"
    exit 1
fi

sudo docker pull ${DOCKERIMG}
if [ $? -ne 0 ]; then
    echo "docker pull failed"
    exit 1
fi

getAddr=$(sudo docker run -v ~/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan console --exec "personal.newAccount('${PASSWD}')")

ADDR=$getAddr

echo $ADDR

getPK=$(sudo docker run -v ~/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan console --exec "personal.showPublicKey(${ADDR},'${PASSWD}')")
PK=$getPK

echo $PK

echo ${PASSWD} | sudo tee -a ~/.wanchain/pw.txt > /dev/null
if [ $? -ne 0 ]; then
    echo "write pw.txt failed"
    exit 1
fi

addrNew=`echo ${ADDR} | sed 's/.\(.*\)/\1/' | sed 's/\(.*\)./\1/'`

sudo docker run -d --name gwan -p 17717:17717 -p 17717:17717/udp -v ~/.wanchain:/root/.wanchain ${DOCKERIMG} /bin/gwan --etherbase ${addrNew} --unlock ${addrNew} --password /root/.wanchain/pw.txt --mine --minerthreads=1 --wanstats ${YOUR_NODE_NAME}:wanchainmainnetvalidator@wanstats.io

if [ $? -ne 0 ]; then
    echo "docker run failed"
    exit 1
fi

echo 'Please wait a few seconds...'

sleep 5

sudo rm ~/.wanchain/pw.txt

KEYSTOREFILE=$(sudo ls ~/.wanchain/keystore/)

KEYSTORE=$(sudo cat ~/.wanchain/keystore/${KEYSTOREFILE})

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
