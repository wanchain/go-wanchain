## WANChain Go

Branch    | Tests 
----------|-------
master    | [![CircleCI](https://circleci.com/gh/wanchain/go-wanchain/tree/master.svg?style=shield)](https://circleci.com/gh/wanchain/go-wanchain/tree/master) 
jia_add_precomiled_contract    | [![CircleCI](https://circleci.com/gh/wanchain/go-wanchain/tree/jia_add_precomiled_contract.svg?style=shield)](https://circleci.com/gh/wanchain/go-wanchain/tree/jia_add_precomiled_contract) 


Requirement: Docker

1. For wanchain: git clone this repo, then run dev_mkimg.sh to build and run docker.
alternatively, run dev_setup.sh to use the prebuild docker image from aliyun.
   In docker container, run /wanchain/src/chainbatch.sh to start wanchain

2. For other tools in wanchain such as wanwallet or wanchain explorer, just run release_mkimg.sh
   and then run a docker container: docker run -d wanchainrelease
