#!/bin/sh


#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
# 

nodesNum=$1

if [ $nodesNum -lt 0 ]; then
	return
fi        
                                                            
cd ..

SRCDIR="$(pwd)"

gethFile="$SRCDIR/build/bin/geth"
if [ ! -x $gethFile ]; then
	make all
fi

containerPrefix="wanchainContainer"
nodeDirPrefix="$SRCDIR/networkScript/node"
ethbase="0x8b179c2b542f47bb2fb2dc40a3cf648aaae1df16"

allEndNodes=""
port1=8746
port2=3803

for ((i=0;i<$nodesNum;i++));
do
	containerName=$containerPrefix$i
	nodeName="node$i"
	
	let port1++  
	let port2++  

	echo $port1  $port2
	
	cd $SRCDIR/build/bin/
	
	mkdir $SRCDIR/networkScript/$nodeName/
	
	bootnode -genkey $SRCDIR/networkScript/$nodeName/nodekey	
	pubHash=`bootnode -nodekey $SRCDIR/networkScript/$nodeName/nodekey -writeaddress`
	
	if [ ! -x $nodeDirPrefix$i ]; then
		mkdir $nodeDirPrefix$i
	fi
	
	echo "wl" | sudo -S rm $nodeDirPrefix$i/data-loadScript/geth
	echo "wl" | sudo -S chmod 777 $nodeDirPrefix$i -R
	
	docker stop $containerName
	docker rm   $containerName
	docker inspect $containerName > /dev/null 2>&1
	if [ $? -eq 1 ]; then
		docker run --restart always --name $containerName -itd -v $SRCDIR:/wanchain/src -p $port1:8545 -p $port2:17717 -p $port2:17717/udp registry.cn-hangzhou.aliyuncs.com/wanglutech/wanchaindev /bin/sh
	fi
	
	ip=$(docker exec $containerName ifconfig | grep "inet addr" | grep -v 127.0.0.1 | awk '{print $2}' | awk -F ':' '{print $2}')
	endnodeurl="enode://$pubHash@$ip:17717"
	
	if [ $i -eq 0 ]; then
		allEndNodes="$endnodeurl"
	else
		allEndNodes="$allEndNodes,$endnodeurl"
	fi
	
	echo $allEndNodes
 
	cd $SRCDIR

	docker exec -it $containerName /wanchain/src/build/bin/geth --datadir "/wanchain/src/networkScript/$nodeName/data-loadScript" init /wanchain/src/genesis_example/genesis.json

	if [ $i -eq 0 ]; then
		echo " start $i"
		docker exec -d $containerName /wanchain/src/build/bin/geth --datadir "/wanchain/src/networkScript/$nodeName/data-loadScript" --networkid 314590 --ipcdisable --gasprice 20000 --mine --minerthreads 1 --rpc --rpcaddr 0.0.0.0 --rpcapi "eth,personal,net,admin,wan" --etherbase $ethbase --nodekey "/wanchain/src/networkScript/$nodeName/nodekey"
	else
		echo " start $i"
		
		docker exec -d $containerName /wanchain/src/build/bin/geth --datadir "/wanchain/src/networkScript/$nodeName/data-loadScript" --networkid 314590 --ipcdisable --gasprice 20000 --mine --minerthreads 1 --rpc --rpcaddr 0.0.0.0 --rpcapi "eth,personal,net,admin,wan" --etherbase $ethbase --nodekey "/wanchain/src/networkScript/$nodeName/nodekey" --bootnodes $allEndNodes
		
	fi

done

