// Code generated by '{{GENERATOR}}'. DO NOT EDIT.
// template: {{TEMPLATE}}

package generated

import (
	"context"
	"fmt"

	{{TYPE_PACKAGE}} "{{PROTO_INC}}"

	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients/generated"
)

func (s *{{TYPE_PACKAGE}}{{SERVICE}}Srv) {{METHOD}}(ctx context.Context, request *{{TYPE_PACKAGE}}.{{REQUEST}}) (*{{TYPE_PACKAGE}}.{{RESPONSE}}, error) {
	response, err := s.reqProcessor.ProcessExternalRequest(ctx, generated.{{SERVICE}}V{{VERSION}}Request, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process %s request: %w", generated.{{SERVICE}}V{{VERSION}}Request, err)
	}
	resp, ok := response.(*{{TYPE_PACKAGE}}.{{RESPONSE}})
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected %s, got %T", generated.{{SERVICE}}V{{VERSION}}Response, response)
	}
	return resp, nil
}
