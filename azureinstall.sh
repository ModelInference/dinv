#!/bin/bash

#go to home directory
cd

#download go binary
wget https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz

#unzip and remove
sudo tar -C /usr/local -xzf go1.7.4.linux-amd64.tar.gz
rm go1.7.4.linux-amd64.tar.gz

#export path
export PATH=$PATH:/usr/local/go/bin
echo "" >> .profile
echo "#export go path" >> .profile
echo "export PATH=$PATH:/usr/local/go/bin" >> .profile

#make root directory and set GOPATH
mkdir go
export GOPATH=$HOME/go
echo "" >> .profile
echo "#set GOPATH" >> .profile
echo "export GOPATH=$HOME/go" >> .profile

#export workspace bin
export PATH=$PATH:$GOPATH/bin
echo "" >> .profile
echo "#set local bin" >> .profile
echo "export PATH=$PATH:$GOPATH/bin" >> .profile

#install mercurial
sudo apt-get install mercurial

#install GoVector to boot strap directories
go get github.com/arcaneiceman/GoVector

JAVAV=jdk-8u112-linux-x64
JAVADIR=jdk1.8.0_112
DAIKONV=daikon-5.5.0
#install java

wget --no-check-certificate --no-cookies --header "Cookie: oraclelicense=accept-securebackup-cookie" http://download.oracle.com/otn-pub/java/jdk/8u112-b15/$JAVAV.tar.gz
cd

tar -xvf $JAVAV.tar.gz
sudo mkdir -p /usr/lib/jvm

sudo mv ./$JAVADIR /usr/lib/jvm/
sudo update-alternatives --install "/usr/bin/java" "java" "/usr/lib/jvm/$JAVADIR/bin/java" 1
sudo update-alternatives --install "/usr/bin/javac" "javac" "/usr/lib/jvm/$JAVADIR/bin/javac" 1
sudo update-alternatives --install "/usr/bin/javaws" "javaws" "/usr/lib/jvm/$JAVADIR/bin/javaws" 1

sudo chmod a+x /usr/bin/java
sudo chmod a+x /usr/bin/javac
sudo chmod a+x /usr/bin/javaws
sudo chown -R root:root /usr/lib/jvm/$JAVADIR

sudo update-alternatives --config java
sudo update-alternatives --config javac
sudo update-alternatives --config javaws



#Install daikon
wget http://plse.cs.washington.edu/daikon/download/$DAIKONV.tar.gz
tar zxf $DAIKONV.tar.gz

echo "export DAIKONDIR=/home/stewart/$DAIKONV" >> ~/.bashrc
export DAIKONDIR=/home/stewart/$DAIKONV
echo "export JAVA_HOME=/usr/lib/jvm/$JAVADIR" >> ~/.bashrc
export JAVA_HOME=/usr/lib/jvm/$JAVADIR
echo "source $DAIKONDIR/scripts/daikon.bashrc" >> ~/.bashrc
source $DAIKONDIR/scripts/daikon.bashrc

#install make
sudo apt-get install make

make -C $DAIKONDIR rebuild-everything
