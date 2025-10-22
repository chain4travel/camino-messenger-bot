// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ ResponseHeaderHandler = (*responseHeaderHandler)(nil)

type ResponseHeaderHandler interface {
	AddError(response protoreflect.ProtoMessage, errMessage string)
}

type responseHeaderHandler struct {
	logger *zap.SugaredLogger
}

func NewResponseHeaderHandler(logger *zap.SugaredLogger) ResponseHeaderHandler {
	return &responseHeaderHandler{
		logger: logger,
	}
}

func (h *responseHeaderHandler) AddError(response protoreflect.ProtoMessage, errMessage string) {
	headerFieldDescriptor := response.ProtoReflect().Descriptor().Fields().ByName("header")
	headerReflectValue := response.ProtoReflect().Get(headerFieldDescriptor)

	switch header := headerReflectValue.Message().Interface().(type) {
	case *typesv1.ResponseHeader:
		addErrorToResponseHeaderV1(header, errMessage)
	case *typesv4.ResponseHeader:
		addErrorToResponseHeaderV4(header, errMessage)
	default:
		h.logger.Errorf("failed add error to response header: %v", errMessage)
	}
}

func addErrorToResponseHeaderV1(header *typesv1.ResponseHeader, errMessage string) {
	header.Status = typesv1.StatusType_STATUS_TYPE_FAILURE
	header.Alerts = append(header.Alerts, &typesv1.Alert{
		Message: errMessage,
		Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
	})
}

func addErrorToResponseHeaderV4(header *typesv4.ResponseHeader, errMessage string) {
	header.Status = typesv4.StatusType_STATUS_TYPE_FAILURE
	header.Alerts = append(header.Alerts, &typesv4.Alert{
		Message: errMessage,
		Type:    typesv4.AlertType_ALERT_TYPE_ERROR,
	})
}
