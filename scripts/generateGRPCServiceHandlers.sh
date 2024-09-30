#!/bin/bash

BUF_SDK_URL=buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go
echo "Downloading SDK from $BUF_SDK_URL"
go get $BUF_SDK_URL

VERSION=$(grep -oP "buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go.*" go.mod | cut -d" " -f2)
echo "Extracting version from go.mod: $VERSION"

SDK_PATH="${GOPATH:-~/.go}/pkg/mod/buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go@${VERSION}/"
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

