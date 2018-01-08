#!/bin/bash
#Assume that the nodes and ip folders have been created with pem file.
#mapping and allocation file should exists.
#Usage: gennodes
#Example: gennodes.sh
###############################
if [ ! -f "mapping" ]; then
    echo "mapping file does not exist!"
    exit 0
fi

if [ ! -f "allocation" ]; then
    echo "mapping file does not exist!"
    exit 0
fi

if [ ! -f "nodes.zip" ]; then
    echo "nodes.zip file does not exist!"
    exit 0
fi

if [ ! -f "hsmsync.sh" ]; then
    echo "hsmsync.sh file does not exist!"
    exit 0
fi

if [ ! -f "ssl.cnf" ]; then
    echo "ssl.cnf file does not exist!"
    exit 0
fi

if [ ! -f "syncagent" ]; then
    echo "syncagent file does not exist!"
    exit 0
fi

###############################
rm -rf ../nodes
unzip -d ../ nodes.zip

while read LINE
do
    curKeystore=`echo "$LINE" | grep -o -E "^[a-z,A-Z,0-9,.]+"`
    curIP=`echo "$LINE" | grep -o "[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}"`
    if [ ! -d "../nodes/$curIP" ]; then
        echo "../nodes/$curIP does not exist and is created!"
        mkdir ../nodes/$curIP
    fi
    cat mapping | grep "$curKeystore\." | grep -o  -E '[0-9]+$' > ../nodes/$curIP/handler
    cp ./hsmsync.sh ../nodes/$curIP/
    cp ./ssl.cnf ../nodes/$curIP/
    cp ./syncagent ../nodes/$curIP/
done < ./allocation

echo "gennodes completes!"
history -c
