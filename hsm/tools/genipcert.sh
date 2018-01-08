#!/bin/bash
#Usage: genipcert -i [IPFile] -o [DataFolder]
#Example: genipcert.sh -i ipfile -o ipcert
###############################
prevContext=""
IPFile="ipfile"
DataFolder="ipcert"
IsError=0

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
            echo "Usage: genipcert -i [IPFile] -o [DataFolder]"
            exit 0
        fi
    elif [[ $prevContext = "-i" ]]
    then
        IPFile=$1
        prevContext=""
    elif [[ $prevContext = "-o" ]]
    then
        DataFolder=$1
        prevContext=""
    else
        IsError=1
     fi
     shift
done

if [[ $IsError -eq 1 || $IPFile = "" || $DataFolder = "" ]]
then
    echo "Invalid Argument Error!"
    echo "Usage: genipcert -i [IPFile] -o [DataFolder]"
    exit 0
fi

###############################
rm -rf ./$DataFolder
mkdir ./$DataFolder

while read LINE
do
    curIP=$LINE
    export SAN="IP:$curIP"
    mkdir ./$DataFolder/$curIP
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 -subj '/CN=:$curIP' -keyout ./$DataFolder/$curIP/server.key -out ./$DataFolder/$curIP/server.pem -config ssl.cnf
done < $IPFile

cp -r ./$DataFolder ./nodes
#need to keep all the files in nodes.zip!
#find ./nodes -name "*.key" | xargs rm -f
zip -r nodes.zip ./nodes/*
rm -rf ./nodes

echo "genipcert completes!"
history -c
