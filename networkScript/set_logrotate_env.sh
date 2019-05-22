#!/bin/sh
# set up the logrotate environment to backup wan-chain log data

#   __        ___    _   _  ____ _           _       ____

#   \ \      / / \  | \ | |/ ___| |__   __ _(_)_ __ |  _ \  _____   __

#    \ \ /\ / / _ \ |  \| | |   | '_ \ / _` | | '_ \| | | |/ _ \ \ / /

#     \ V  V / ___ \| |\  | |___| | | | (_| | | | | | |_| |  __/\ V /

#      \_/\_/_/   \_\_| \_|\____|_| |_|\__,_|_|_| |_|____/ \___| \_/

#

#add wanchainlog logrotateconf 
version="v1.0.1"
if [ ! -n "$1" ];then
    echo "There is no version parameter input"
    tmp=`ls -lt $HOME/wanchain | grep '^d' | awk '{print $9}' | head -1`
    if [ -f "$HOME/wanchain/$tmp/bin/gwan" ];then
        version=$tmp
        echo "The newest version is $version"
    fi
else
    echo "The input version parameter is $1"
    version=$1
fi
wanchainLogPath=$HOME/wanchain/$version/log/running.log
wanchainLogRotateConf=/etc/logrotate.d/wanchainlog

sudo touch $wanchainLogRotateConf
sudo chmod 777 $wanchainLogRotateConf
echo "
$wanchainLogPath
{
   su root list
   daily
   dateext
   rotate 31
   compress
   notifempty
   missingok
   copytruncate
}
" > $wanchainLogRotateConf
sudo chmod 644 $wanchainLogRotateConf

#add daily schedule to crontab
sudo chmod 777 /etc/crontab
sed -n '/cron.daily/p' /etc/crontab | sudo sed -i 's/25 6/59 23/g' /etc/crontab
sudo chmod 644 /etc/crontab

sudo /etc/init.d/cron restart
