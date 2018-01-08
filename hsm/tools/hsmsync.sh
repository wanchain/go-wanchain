#!/bin/bash
#Assume it is executed under IP folder.
#syncagent --address [IP] --nodePIN [nodePIN] --envPWD [envPWD] --file [keyfile]
#Usage: hsmsync -a [IP] -u [hsmName] -p [hsmPWD] -n [nodePIN] -e [envPWD]
###############################
prevContext=""
hsmName=""
hsmPWD=""
nodePIN=""
envPWD=""
ipAddr=""
sslcnf="./ssl.cnf"
wrapKey=8

while [  "$#" -ge  "1" ]
do
    if [[ $prevContext = "" ]]
    then
        temp=$1
        if [[ ${temp:0:1} = "-"  ]]
        then
            prevContext=$1
        else
            echo "Invalid Argument Error!"
            echo "Usage: hsmsync -u [hsmName] -p [hsmPWD] -n [nodePIN] -e [envPWD]"
            exit 0
        fi
    elif [[ $prevContext = "-a" ]]
    then
        ipAddr=$1
        prevContext=""
    elif [[ $prevContext = "-u" ]]
    then
        hsmName=$1
        prevContext=""
    elif [[ $prevContext = "-p" ]]
    then
        hsmPWD=$1
        prevContext=""
    elif [[ $prevContext = "-n" ]]
    then
        nodePIN=$1
        prevContext=""
    elif [[ $prevContext = "-e" ]]
    then
        envPWD=$1
        prevContext=""
    else
        IsError=1
     fi
     shift
done

if [[ $IsError -eq 1 || $hsmName = "" || $hsmPWD = "" || $nodePIN = "" || $envPWD = "" ]]
then
    echo "Invalid Argument Error!"
    echo "Usage: hsmsync -u [hsmName] -p [hsmPWD] -n [nodePIN] -e [envPWD]"
    exit 0
fi

###############################
curIP=${PWD##*/}
curHandler=`cat handler`

if [[ $curHandler = "" ]]
then
    echo "handler is not correct!"
    exit 0
fi

if [[ ! $ipAddr = "" && ! $ipAddr = $curIP ]]
then
    if [ ! -f "ssl.cnf" ]; then
        echo "ssl.cnf file does not exist!"
        exit 0
    fi   
    export SAN="IP:$ipAddr"
    reqret=`openssl req -new -x509 -days 3650 -key ./server.key -out ./server.pem -subj "/CN=${ipAddr}" -config $sslcnf`
    if [[ ! $reqret = "" ]]
    then
        echo "Error happens when generate ip cert."
        echo $reqret 
        exit 0
    fi
fi

`/opt/cloudhsm/bin/key_mgmt_util Cfm3Util singlecmd loginHSM -u CU -s ${hsmName} -p ${hsmPWD} exSymKey -k ${curHandler} -w ${wrapKey} -out tempkey.zip`

if [ ! -f "tempkey.zip" ]; then
    echo "key is not retrieved correctly!"
    exit 0
fi

if [[ ! $ipAddr = "" && ! $ipAddr = $curIP ]]
then
    ./syncagent --address ${ipAddr} --nodePIN ${nodePIN} --envPWD ${envPWD} --file tempkey.zip
    if [ ! -d "../$ipAddr" ]; then
        mv ../$curIP ../$ipAddr
    fi
else
    ./syncagent --address ${curIP} --nodePIN ${nodePIN} --envPWD ${envPWD} --file tempkey.zip
fi

rm -rf tempkey.zip
history -c
