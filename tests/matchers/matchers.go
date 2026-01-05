// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package matchers

import (
	"context"

	"go.uber.org/mock/gomock"
)

var Context = gomock.Cond(func(x any) bool {
	_, ok := x.(context.Context)
	return ok
})
