// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"os"

	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/server"
)

func main() {
	if err := server.Run(); err != nil {
		os.Exit(1)
	}
}
