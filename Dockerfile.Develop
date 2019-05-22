# This Dockfile is for WANChain Developer
# After download WANChain main code from  Github, user can build docker image
# sudo docker build -t="wanchain/alpine:1.0" -f ./DOCKER/Dockfile.Develop
# sudo docker run -it -v absolute_path4src:/wanchain/src wachain/alpine_go:1.0 sh
#  

FROM alpine:3.6

RUN mkdir /wanchain

#ENV WANCHAIN /
#ADD ./go-ethereum-ota /wanchain/src
ADD ./DOCKER/data /wanchain/data

VOLUME /wanchain/src

#bash
RUN \
  apk add --update git go make gcc musl-dev linux-headers
  #(cd wanchain && make geth)                              && \
  #cp /wanchain/build/bin/geth /usr/local/bin/

EXPOSE 8545
EXPOSE 17717
EXPOSE 17717/udp


#
# geth --verbosity 5 --datadir data --etherbase '0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e' --networkid 5201314 --mine --minerthreads 1 --nodiscover --rpc
#