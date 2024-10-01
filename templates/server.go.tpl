// Code generated by '{{GENERATOR}}'. DO NOT EDIT.
// template: {{TEMPLATE}}

package generated

import (
	"{{GRPC_INC}}"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	
	"google.golang.org/grpc"
)

var _ {{GRPC_PACKAGE}}.{{SERVICE}}Server = (*{{TYPE_PACKAGE}}{{SERVICE}}Srv)(nil)

type {{TYPE_PACKAGE}}{{SERVICE}}Srv struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func register{{SERVICE}}V{{VERSION}}Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	{{GRPC_PACKAGE}}.Register{{SERVICE}}Server(grpcServer, &{{TYPE_PACKAGE}}{{SERVICE}}Srv{reqProcessor})
}
