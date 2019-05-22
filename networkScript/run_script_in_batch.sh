#!/bin/sh
# set up the logrotate environment to backup wan-chain log data

#   __        ___    _   _  ____ _           _       ____

#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __

#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /

#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V /

#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/

#

#set logrotate at the miner server in batch

serverUser="ubuntu"
serverPwd=""
serverIps=(
    #"127.0.0.1" #the ip server should be included here
    )
serverKey="/Users/aaron/wanglu/wanchain_key/wanchain_b.pem.pub"
script="set_logrotate_env.sh"

echo "The log rotate script will be run in batch in servers!\n"

for serverIp in "${serverIps[@]}"
do
    echo "The server " $serverIp " will run the script"

    command=`echo ssh -i $serverKey $serverUser@$serverIp -C \"/bin/bash\"`
    echo $command
    $command < $script

    if [ $? -ne 0 ];then
        echo "The script run with fail"
    else
        echo "The script run successfully"
    fi

    echo ""
done
