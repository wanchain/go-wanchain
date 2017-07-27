#!/bin/sh


#   __        ___    _   _  ____ _           _       ____             
#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __
#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /
#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V / 
#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/  
#                                                                     


SRCDIR="$(pwd)"
docker inspect wanchaindev > /dev/null 2>&1
if [ $? -eq 1 ]; then
	docker run --restart always --name wanchaindev -itd -v $SRCDIR:/wanchain/src -p 8545:8545 registry.cn-hangzhou.aliyuncs.com/wanglutech/wanchaindev /bin/sh
fi
docker exec -it wanchaindev /bin/sh

