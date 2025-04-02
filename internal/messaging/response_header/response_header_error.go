package responseheader

import (
	"fmt"

	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func AddErrorToResponseHeader(response protoreflect.ProtoMessage, errMessage string) error {
	headerFieldDescriptor := response.ProtoReflect().Descriptor().Fields().ByName("header")
	headerReflectValue := response.ProtoReflect().Get(headerFieldDescriptor)

	switch header := headerReflectValue.Message().Interface().(type) {
	case *typesv1.ResponseHeader:
		AddErrorToResponseHeaderV1(header, errMessage)
	default:
		return fmt.Errorf("failed add error (%s) to response header: unknown header type", errMessage)
	}
	return nil
}

func AddErrorToResponseHeaderV1(header *typesv1.ResponseHeader, errMessage string) {
	header.Status = typesv1.StatusType_STATUS_TYPE_FAILURE
	header.Alerts = append(header.Alerts, &typesv1.Alert{
		Message: errMessage,
		Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
	})
}
