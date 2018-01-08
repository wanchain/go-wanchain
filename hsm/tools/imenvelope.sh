#!/bin/bash
#Usage: imenvelope -u [UserName] -p [Password] -w [WrapHandler] -d [DataFolder] -o [MappingFile]
###############################
prevContext=""
UserName=""
Password=""
WrapHandler=0
DataFolder="data"
MappingFile="mapping"
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
            echo "Usage: imenvelope -u [UserName] -p [Password] -w [WrapHandler] -d [DataFolder] -o [MappingFile]"
            exit 0
        fi
    elif [[ $prevContext = "-u" ]]
    then
        UserName=$1
        prevContext=""
    elif [[ $prevContext = "-p" ]]
    then
        Password=$1
        prevContext=""
    elif [[ $prevContext = "-w" ]]
    then
        let temp=$1
        WrapHandler=$temp
        prevContext=""
    elif [[ $prevContext = "-d" ]]
    then
        DataFolder=$1
        prevContext=""
    elif [[ $prevContext = "-o" ]]
    then
        MappingFile=$1
        prevContext=""
    else
        IsError=1
     fi
     shift
done

if [[ $IsError -eq 1 || $UserName = "" || $Password = "" || $DataFolder = "" || $MappingFile = "" || $WrapHandler -lt 1 ]]
then
    echo "Invalid Argument Error!"
    echo "Usage: imenvelope -u [UserName] -p [Password] -w [WrapHandler] -d [DataFolder] -o [MappingFile]"
    exit 0
fi

###############################
rm -rf $MappingFile
for file in ${DataFolder}/*.zip
do
    envFile=${file}
    label=${file}
    #Attention that should use "[0-9]+".
    imHandle=`/opt/cloudhsm/bin/key_mgmt_util Cfm3Util singlecmd loginHSM -u CU -s ${UserName} -p ${Password} imSymKey -f ${envFile} -w ${WrapHandler} -t 16 -l ${label} | grep "Handle:" | grep -o  -E "[0-9]+"`
    echo ${label} ${imHandle} >> $MappingFile
done

echo "imenvelope completes!"
history -c
