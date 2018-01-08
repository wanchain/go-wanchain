#handleaddr.sh
#addresses file should be put to the folder first.
rm -rf tempadd
rm -rf finaladdr
cat addresses | grep -o -E "[0-9,a-z,A-Z]+$" > tempadd
while read LINE
do
    curAddr=$LINE
    echo "\"$curAddr\"," >> finaladdr
done < tempadd
rm -rf tempadd
