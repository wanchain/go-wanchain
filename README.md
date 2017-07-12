## WANChain Go

Requirement: Docker

1. For wanchain: git clone this repo, then run dev_setup.sh.
   In docker container, run /wanchain/chainbatch.sh to start wanchain

2. For other tools in wanchain such as wanwallet or wanchain explorer, just run release_mkimg.sh
   and then run a docker container: docker run -d wanchainrelease
