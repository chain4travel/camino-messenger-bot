#!/bin/bash

BUF_SDK_URL=buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go
echo "Downloading SDK from $BUF_SDK_URL"
go get $BUF_SDK_URL

VERSION=$(grep -oP "buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go.*" go.mod | cut -d" " -f2)
echo "Extracting version from go.mod: $VERSION"

echo "Searching for go path"
if [ ! -z "$GOPATH" ] ; then
	GO_PATH=$GOPATH
else
	# try out the defaults:
	if [ -f ~/.go ] ; then
		GO_PATH=~/.go
	elif [ -f ~/go ] ; then
		GO_PATH=~/go
	fi
	
fi

if [ -z "$GO_PATH" ] ; then
	# still no luck? extract it from go env!
	GO_PATH=$(go env | grep GOPATH | cut -d"'" -f2)
fi

if [ -z "$GO_PATH" ] ; then
	echo "Can't find go path :("
	exit 1
fi

SDK_PATH="${GO_PATH}/pkg/mod/buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go@${VERSION}/"
echo "SDK_PATH: $SDK_PATH"

while read file ; do
	echo "Scanning file $file"

	PACKAGE=$(grep -oP '^package \S+$' $file | cut -d" " -f2)
	FQPN=$(grep -P 'FullMethodName' $file | grep -oP 'cmp\.services\.[^/]+' | head -n1) # only 1 - it may contain more

	echo " Package: $PACKAGE"
	echo " FQPN   : $FQPN"

	echo " Method names: "
	for method in $(grep -P 'FullMethodName' $file | grep -oP 'cmp\.services\.[^"]+' | cut -d"/" -f2) ; do
		echo " -> $method"
		# Now we try to extract the inputs and outputs for each method
		# for this grep $method(xxx, INPUT) (OUTPUT, yyy)
		# strip out the version prefix (*vX.)Name
	done

	echo 
done < <(find "$SDK_PATH/cmp/services/" -name "*_grpc.pb.go")

