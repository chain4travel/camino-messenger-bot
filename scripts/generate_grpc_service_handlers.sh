#!/bin/bash

BUF_SDK_BASE="buf.build/gen/go/chain4travel/camino-messenger-protocol"
TEMPLATES_DIR="templates"
CLIENT_TEMPLATE="${TEMPLATES_DIR}/client.go.tpl"
CLIENT_METHOD_TEMPLATE="${TEMPLATES_DIR}/client_method.go.tpl"
SERVER_TEMPLATE="${TEMPLATES_DIR}/server.go.tpl"
SERVER_METHOD_TEMPLATE="${TEMPLATES_DIR}/server_method.go.tpl"
GEN_OUTPATH="internal/rpc/generated"
REGISTER_SERVICES_SERVER_FILE="${GEN_OUTPATH}/register_server_services.go"
REGISTER_SERVICES_CLIENT_FILE="${GEN_OUTPATH}/register_client_services.go"
UNMARSHALLING_FILE="${GEN_OUTPATH}/unmarshal.go"

DEFAULT_BLACKLIST="notification" # we don't want to generate handlers for notifications - if we ever need more filters here the impl. need to change!

SCRIPT=$0
FILTER=$1 #optional filter for files -- used for testing

function generate_with_templates() {
	FQPN="$1"
	SERVICE="$2"
	GRPC_PACKAGE="$3"
	TYPE_PACKAGE="$4"
	local -n _METHODS=$5
	local -n _INPUTS=$6
	local -n _OUTPUTS=$7
	GRPC_INCLUDE="$8"
	PROTO_INCLUDE="$9"
	_VERSION="${10}"
	GENERATOR="${11}"

	# From here on it's very easy in case it's just search replace
	# Simply copy the template to the target directory
	# And do some sed -i -e "s#{{.FQDN}}#$FQDN#g" -e "s#{{.SERVICE}}#$SERVICE#g" [...] $file
	GENERAL_PARAM_REPLACE=""
	GENERAL_PARAM_REPLACE+=" -e s#{{FQPN}}#$FQPN#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{SERVICE}}#$SERVICE#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{GRPC_PACKAGE}}#$GRPC_PACKAGE#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{TYPE_PACKAGE}}#$TYPE_PACKAGE#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{GRPC_INC}}#$GRPC_INCLUDE#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{PROTO_INC}}#$PROTO_INCLUDE#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{VERSION}}#$_VERSION#g"
	GENERAL_PARAM_REPLACE+=" -e s#{{GENERATOR}}#$GENERATOR#g"

	# Generate client
	CLIENT_GEN_FILE="${GEN_OUTPATH}/${TYPE_PACKAGE}_${SERVICE}_client.go"
	echo "🔨 Generating client: $CLIENT_GEN_FILE"
	cp $CLIENT_TEMPLATE $CLIENT_GEN_FILE
	sed -i $GENERAL_PARAM_REPLACE $CLIENT_GEN_FILE
	sed -i -e "s#{{TEMPLATE}}#$CLIENT_TEMPLATE#g" $CLIENT_GEN_FILE

	# Generate client methods
	for num in `seq 0 $(( ${#_METHODS[@]} - 1 ))` ; do
		METHOD=${_METHODS[$num]}
		INPUT=${_INPUTS[$num]}
		OUTPUT=${_OUTPUTS[$num]}
		METHOD_PARAM_REPLACE=""
		METHOD_PARAM_REPLACE+=" -e s#{{METHOD}}#$METHOD#g"
		METHOD_PARAM_REPLACE+=" -e s#{{REQUEST}}#$INPUT#g"
		METHOD_PARAM_REPLACE+=" -e s#{{RESPONSE}}#$OUTPUT#g"
		METHOD_GEN_FILE="${GEN_OUTPATH}/${TYPE_PACKAGE}_${SERVICE}_${METHOD}_client_method.go"
		echo "🔨 Generating client method: $METHOD_GEN_FILE"
		cp $CLIENT_METHOD_TEMPLATE $METHOD_GEN_FILE
		sed -i $GENERAL_PARAM_REPLACE $METHOD_GEN_FILE
		sed -i $METHOD_PARAM_REPLACE $METHOD_GEN_FILE
		sed -i -e "s#{{TEMPLATE}}#$CLIENT_METHOD_TEMPLATE#g" $METHOD_GEN_FILE
	done

	# Generate server
	SERVER_GEN_FILE="${GEN_OUTPATH}/${TYPE_PACKAGE}_${SERVICE}_server.go"
	echo "🔨 Generating server: $SERVER_GEN_FILE"
	cp $SERVER_TEMPLATE $SERVER_GEN_FILE
	sed -i $GENERAL_PARAM_REPLACE $SERVER_GEN_FILE
	sed -i -e "s#{{TEMPLATE}}#$SERVER_TEMPLATE#g" $SERVER_GEN_FILE

	# Generate server methods
	for num in `seq 0 $(( ${#_METHODS[@]} - 1 ))` ; do
		METHOD=${_METHODS[$num]}
		INPUT=${_INPUTS[$num]}
		OUTPUT=${_OUTPUTS[$num]}
		METHOD_PARAM_REPLACE=""
		METHOD_PARAM_REPLACE+=" -e s#{{METHOD}}#$METHOD#g"
		METHOD_PARAM_REPLACE+=" -e s#{{REQUEST}}#$INPUT#g"
		METHOD_PARAM_REPLACE+=" -e s#{{RESPONSE}}#$OUTPUT#g"
		METHOD_GEN_FILE="${GEN_OUTPATH}/${TYPE_PACKAGE}_${SERVICE}_${METHOD}_server_method.go"
		echo "🔨 Generating server method: $METHOD_GEN_FILE"
		cp $SERVER_METHOD_TEMPLATE $METHOD_GEN_FILE
		sed -i $GENERAL_PARAM_REPLACE $METHOD_GEN_FILE
		sed -i $METHOD_PARAM_REPLACE $METHOD_GEN_FILE
		sed -i -e "s#{{TEMPLATE}}#$SERVER_METHOD_TEMPLATE#g" $METHOD_GEN_FILE
	done
}

function generate_register_services_server() {
	OUTFILE="$1"
	local -n _SERVICES=$2

	echo "📝 Registering server services in $OUTFILE"
	
	echo "// Code generated by '{{GENERATOR}}'. DO NOT EDIT." > $OUTFILE
	echo >> $OUTFILE
	echo "package generated" >> $OUTFILE
	echo >> $OUTFILE
	echo "import (" >> $OUTFILE
	echo "    \"github.com/chain4travel/camino-messenger-bot/internal/rpc\"" >> $OUTFILE
	echo "    \"google.golang.org/grpc\"" >> $OUTFILE
	echo ")" >> $OUTFILE
	echo >> $OUTFILE
	echo "func RegisterServerServices(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {" >> $OUTFILE
	for service in "${_SERVICES[@]}" ; do
		echo "    register${service}Server(grpcServer, reqProcessor)" >> $OUTFILE
	done
	echo "}" >> $OUTFILE
}

function generate_register_services_client() {
	OUTFILE="$1"
	local -n _SERVICES=$2

	echo "📝 Registering client services in $OUTFILE"
	
	echo "// Code generated by '{{GENERATOR}}'. DO NOT EDIT." > $OUTFILE
	echo >> $OUTFILE
	echo "package generated" >> $OUTFILE
	echo >> $OUTFILE
	echo "import (" >> $OUTFILE
	echo "    \"github.com/chain4travel/camino-messenger-bot/internal/messaging/types\"" >> $OUTFILE
	echo "    \"github.com/chain4travel/camino-messenger-bot/internal/rpc\"" >> $OUTFILE
	echo "    \"google.golang.org/grpc\"" >> $OUTFILE
	echo ")" >> $OUTFILE
	echo >> $OUTFILE
	echo "func RegisterClientServices(rpcConn *grpc.ClientConn, serviceNames map[string]struct{}) map[types.MessageType]rpc.Service {" >> $OUTFILE
	echo "    services := make(map[types.MessageType]rpc.Service, len(serviceNames))" >> $OUTFILE
	echo >> $OUTFILE
	for service in "${_SERVICES[@]}" ; do
		echo "    if _, ok := serviceNames[${service}]; ok {" >> $OUTFILE
		echo "        services[${service}Request] = rpc.NewService(New${service}(rpcConn), ${service})" >> $OUTFILE
		echo "        delete(serviceNames, ${service})" >> $OUTFILE
		echo "    }" >> $OUTFILE
	done
	echo "    return services" >> $OUTFILE
	echo "}" >> $OUTFILE
}

function generate_unmarshalling() {
	OUTFILE="$1"
	local -n _PROTO_INCLUDES=$2
	local -n _UNMARSHAL_METHODS=$3

	echo "📝 Generating unmarshalling in $OUTFILE"
	
	echo "// Code generated by '{{GENERATOR}}'. DO NOT EDIT." > $OUTFILE
	echo >> $OUTFILE
	echo "package generated" >> $OUTFILE
	echo >> $OUTFILE
	echo "import (" >> $OUTFILE
	for include in "${_PROTO_INCLUDES[@]}" ; do
		echo "    \"${include}\"" >> $OUTFILE
	done
	echo "    \"github.com/chain4travel/camino-messenger-bot/internal/messaging/types\"" >> $OUTFILE
	echo "    \"google.golang.org/protobuf/proto\"" >> $OUTFILE
	echo "    \"google.golang.org/protobuf/reflect/protoreflect\"" >> $OUTFILE
	echo ")" >> $OUTFILE
	echo >> $OUTFILE

	echo "func UnmarshalContent(src []byte, msgType types.MessageType, destination *protoreflect.ProtoMessage) error {" >> $OUTFILE
	echo "    switch msgType {" >> $OUTFILE
	
	for method in "${_UNMARSHAL_METHODS[@]}" ; do
		IFS=' ' read -r -a parts <<< "$method"
		echo "    case ${parts[0]}:" >> $OUTFILE
		echo "        *destination = &${parts[1]}.${parts[2]}{}" >> $OUTFILE
	done

	echo "    default:" >> $OUTFILE
	echo "        return types.ErrUnknownMessageType" >> $OUTFILE
	echo "    }" >> $OUTFILE
	echo "    return proto.Unmarshal(src, *destination)" >> $OUTFILE
	echo "}" >> $OUTFILE
}

# Prepare the output directories
echo "🧹 Cleaning and generating output directories"
rm -rf $GEN_OUTPATH
mkdir -p $GEN_OUTPATH

BUF_SDK_URL_GO="${BUF_SDK_BASE}/grpc/go"
echo "⌛ Downloading SDK from $BUF_SDK_URL_GO"
go get $BUF_SDK_URL_GO

BUF_VERSION=$(grep -oP "buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go.*" go.mod | cut -d" " -f2)
echo "🔗 Extracting version from go.mod: $BUF_VERSION"

echo "🔍 Searching for go path"
if [ ! -z "$GOPATH" ] ; then
	GO_PATH=$GOPATH
fi

if [ -z "$GO_PATH" ] ; then
	# no luck? extract it from go env!
	GO_PATH=$(go env | grep GOPATH | cut -d"'" -f2)
fi

if [ -z "$GO_PATH" ] ; then
	# still no luck?!
	# try out the defaults:
	if [ -f ~/.go ] ; then
		GO_PATH=~/.go
	elif [ -f ~/go ] ; then
		GO_PATH=~/go
	fi
fi

if [ -z "$GO_PATH" ] ; then
	echo "❌ Can't find go path :("
	exit 1
fi

SDK_PATH="${GO_PATH}/pkg/mod/buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go@${BUF_VERSION}"
echo "SDK_PATH: $SDK_PATH"

echo "⌛ Making sure gofumpt is installed"
go install mvdan.cc/gofumpt@latest

FUMPT="${GO_PATH}/bin/gofumpt"
if [ ! -f $FUMPT ] ; then
	echo "❌ gofumpt not found at $FUMPT"
	exit 1
fi

declare -a SERVICES_TO_REGISTER=()
declare -a PROTO_INCLUDES_FOR_UNMARSHALLING=()
declare -a UNMARSHAL_METHODS=()

while read file ; do
	if [ ! -z "$FILTER" ] ; then
		if [[ ! $file =~ $FILTER ]] ; then
			continue
		fi
	fi

	if [[ $file =~ $DEFAULT_BLACKLIST ]] ; then
		echo "⚠️ Skipping (blacklisted): $file"
		echo 
		continue 
	fi

	echo "🔍 Scanning file $file"

	FQPN=$(grep -P 'FullMethodName' $file | grep -oP 'cmp\.services\.[^/]+' | head -n1) # only 1 - it may contain more
	SERVICE=${FQPN##*.}
	PACKAGE=$(grep -oP '^package \S+$' $file | cut -d" " -f2)
	TYPE=${PACKAGE%*grpc}	
	VERSION=$(echo "$FQPN" | grep -oP "\.v[0-9]+\." | cut -d"." -f2 )

	echo "🔑 FQPN      : $FQPN"
	echo "⚙️ Service   : $SERVICE"
	echo "📦 Go Package: $PACKAGE"
	echo "🔧 Type      : $TYPE"

	echo "📑 Methods: "
	declare -a METHODS=()
	declare -a INPUTS=()
	declare -a OUTPUTS=()
	for method in $(grep -P 'FullMethodName' $file | grep -oP 'cmp\.services\.[^"]+' | cut -d"/" -f2) ; do
		# Now we try to extract the inputs and outputs for each method
		# for this grep $method(xxx, INPUT) (OUTPUT, yyy)
		# strip out the version prefix (*vX.)Name

		PARAMS_CUT=$(grep -P "^\t${method}\(context.Context" $file | cut -d"," -f2 | tr -d '()')
		# This gives us " *stuff.INPUT *stuff.OUTPUT" -- let's get INPUT and OUTPUT separately
		INPUT=$(echo $PARAMS_CUT | cut -d" " -f1 | cut -d"." -f2)
		OUTPUT=$(echo $PARAMS_CUT | cut -d" " -f2 | cut -d"." -f2)

		METHODS+=("$method")
		INPUTS+=("$INPUT")
		OUTPUTS+=("$OUTPUT")

		# We need to pass something for the unmarshalling here
		# First we need the service name including the version plus if it's request or response
		# Then we need the package name
		# And finally Input and Output
		# So something like:
		# PingServiceV1Request pingv1 PingRequest
		# PingServiceV1Response pingv1 PingResponse
		UNMARSHAL_METHODS+=("${SERVICE}V${VERSION:1}Request $TYPE $INPUT")
		UNMARSHAL_METHODS+=("${SERVICE}V${VERSION:1}Response $TYPE $OUTPUT")

		echo " ◉ $method (↓ in: '$INPUT' - ↑ out: '$OUTPUT')"
	done

	# We also need something like:
	# "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	# "buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	# We already have the base URL in BUF_SDK_URL_GO - now we only need the suffixes for the protocolbuffers and grpc
	# And store them into the INCLUDES array
	declare -a INCLUDES=()
	# First the protocolbuffers
	SUFFIX=$(echo ${FQPN%.*} | tr "." "/")
	PROTO_INCLUDE="${BUF_SDK_BASE}/protocolbuffers/go/${SUFFIX}"
	# Now the grpc which is quite similar by just adding the package name at the end
	GRPC_INCLUDE="${BUF_SDK_BASE}/grpc/go/${SUFFIX}/${PACKAGE}"
	generate_with_templates "$FQPN" "$SERVICE" "$PACKAGE" "$TYPE" METHODS INPUTS OUTPUTS "$GRPC_INCLUDE" "$PROTO_INCLUDE" "${VERSION:1}" "$SCRIPT" 
	SERVICES_TO_REGISTER+=("${SERVICE}V${VERSION:1}")
	PROTO_INCLUDES_FOR_UNMARSHALLING+=("$PROTO_INCLUDE")

	echo 
done < <(find "$SDK_PATH/cmp/services/" -name "*_grpc.pb.go")

generate_register_services_server "$REGISTER_SERVICES_SERVER_FILE" SERVICES_TO_REGISTER 
generate_register_services_client "$REGISTER_SERVICES_CLIENT_FILE" SERVICES_TO_REGISTER 
generate_unmarshalling "$UNMARSHALLING_FILE" PROTO_INCLUDES_FOR_UNMARSHALLING UNMARSHAL_METHODS

echo "🧹 Running gofumpt on all generated files"
$FUMPT -w $GEN_OUTPATH

