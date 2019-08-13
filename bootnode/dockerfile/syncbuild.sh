#!/bin/bash

# cd  ../go-wanchain-bak
# files=`git diff --name-only`


# for f in ${files[@]}; do
#        cp ${f} ../go-wanchain/${f}
# done

#cd ../go-wanchain
#make
#cp ./build/bin/gwan ../pos6/bin/
#cd ../pos6

docker build . -t wanchain/client-go:2.1.2
docker push wanchain/client-go:2.1.2