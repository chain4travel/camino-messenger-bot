// Code generated by '{{GENERATOR}}'. DO NOT EDIT.
// template: {{TEMPLATE}}

package generated

import (
	"{{GRPC_INC}}"
	
	"google.golang.org/grpc"
)

var _ {{GRPC_PACKAGE}}.{{SERVICE}}Server = (*{{TYPE_PACKAGE}}{{SERVICE}}Srv)(nil)

type {{TYPE_PACKAGE}}{{SERVICE}}Srv struct {
	reqProcessor externalRequestProcessor
}

func register{{SERVICE}}V{{VERSION}}Server(grpcServer *grpc.Server, reqProcessor externalRequestProcessor) {
	{{GRPC_PACKAGE}}.Register{{SERVICE}}Server(grpcServer, &{{TYPE_PACKAGE}}{{SERVICE}}Srv{reqProcessor})
}
