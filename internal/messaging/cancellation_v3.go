package messaging

import (
	"context"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func (h *evmResponseHandler) handelCancellationInfoResponseV3(ctx context.Context, response protoreflect.ProtoMessage, request protoreflect.ProtoMessage) bool {
	// <on supplier side>
	// TODO:

	return false
}

func (h *evmResponseHandler) handleCancellationInfoRequestV3(ctx context.Context, response protoreflect.ProtoMessage, request protoreflect.ProtoMessage) bool {
	// <on distributor side>

	// TODO:
	// Inventory system replied if cancellation is possible or not
	return false
}
