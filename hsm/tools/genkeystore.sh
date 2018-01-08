#!/bin/bash
#Assume two input file, one is Password file, the other is Envelope file.
#Password file contains all the keystore password.
#Envelope file contains all the envelope password.
#Items in the files above should meet the count restriction.
#In the output folder, the keystore files and envelope files should be generated.
#Usage: genkeystore -p [PasswordFile] -e [EnvelopeFile] -o [OutFile] -s [StartIndex] [Total]
#Example: genkeystore.sh -p passwords -e envelopes -o data -s 1 5
###############################
prevContext=""
PasswordFile="passwords"
EnvelopeFile="envelopes"
OutFile="data"
StartIndex=0
Total=0
IsError=0

#Check geth for the path.
geth=`which geth`
if [[ $geth = "" ]]
then
    echo "Cannot find geth!"
    exit 0
fi

count=1
while [  "$#" -ge  "1" ]
do
    if [[ $prevContext = "" ]]
    then
        temp=$1
        if [[ ${temp:0:1} = "-"  ]]
        then
            prevContext=$1
        else
            let total=$1
            Total=$total
        fi
    elif [[ $prevContext = "-p" ]]
    then
        PasswordFile=$1
        prevContext=""
    elif [[ $prevContext = "-e" ]]
    then
        EnvelopeFile=$1
        prevContext=""
    elif [[ $prevContext = "-o" ]]
    then
        OutFile=$1
        prevContext=""
    elif [[ $prevContext = "-s" ]]
    then
        let temp=$1
        StartIndex=$temp
        prevContext=""
    else
        IsError=1
     fi
     let count=$count+1
     shift
done

if [[ $IsError -eq 1 || $PasswordFile = "" || $EnvelopeFile = "" || $OutFile = "" || $StartIndex -lt 1 || $Total -lt 1 ]]
then
    echo "Invalid Argument Error!"
    echo "Usage: genkeystore -p [PasswordFile] -e [EnvelopeFile] -o [OutFile] -s [StartIndex] 200"
    exit 0
fi

###############################
#while_read_file filename linecount
function while_read_file() {
    let i=0
    temp=()
    myfilename=$1
    let linecount=$2
    while read LINE
    do
        temp[$i]=$LINE
        let i=$i+1
        if [[ $i -ge $linecount ]]
        then
            break
        fi
    done < $myfilename

    echo ${temp[*]}
}

passwords=(`while_read_file $PasswordFile  $Total`)
envelopes=(`while_read_file $EnvelopeFile $Total`)

passlen=${#passwords[@]}
envlen=${#envelopes[@]}

if [[ $passlen -lt $Total || $envlen -lt $Total ]]
then
    echo "Please ensure the passwords or evelopes being enouth!"
    exit 0
fi

###############################
rm -rf ./log
rm -rf ./mypassword
rm -rf ~/.wanchain/keystore/*
rm -rf ./$OutFile
rm -rf ./addresses
mkdir ./$OutFile

let i=1
while [[ $i -le $Total ]]
do
    echo ${passwords[${i}-1]} > ./mypassword
    geth --password ./mypassword account new >> log
    keystore=`ls ~/.wanchain/keystore/`
    let curIndex=$StartIndex+${i}-1
    echo $curIndex $keystore
    curAddress=`ls ~/.wanchain/keystore/ | grep -o -E "[0-9,a-z,A-Z]+$"`
    echo "keystore${curIndex}" $curAddress >> ./addresses
    mv ~/.wanchain/keystore/${keystore} $OutFile/keystore${curIndex}
    7za a -tzip -p${envelopes[${i}-1]} -mem=AES256 $OutFile/keystore${curIndex}.zip $OutFile/keystore${curIndex} >> log
    rm -rf $OutFile/keystore${curIndex}
    let i=$i+1
done

rm -rf ./mypassword
rm -rf ~/.wanchain/keystore/*

echo "genkeystore completes!"
history -c
