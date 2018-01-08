#!/bin/bash
#Assume it is executed under IP cert folder.
#Usage: changeip [ipaddress] 
###############################
#serverPath="./"
serverPath="~/.wanchain/server/"
serverkey="${serverPath}server.key"
serverpem="${serverPath}server.pem"
sslcnf="${serverPath}ssl.cnf"
ipaddress=""

if [ "$#" -eq  "1" ]
then
    ipaddress=$1
else
    echo "Invalid Argument Error!"
    echo "Usage: changeip [ipaddress]"
    exit 0
fi

###############################
export SAN="IP:$ipaddress"
reqret=`openssl req -new -x509 -days 3650 -key $serverkey -out $serverpem -subj "/CN=$ipaddress" -config $sslcnf`
if [[ $reqret = "" ]]
then
    echo "Change ip with cert successfully!"
else
    echo "Error happens when generate ip cert."
    echo $reqret 
fi
