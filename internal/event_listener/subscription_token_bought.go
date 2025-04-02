// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"context"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
)

type tokenBoughtSubscription struct {
	TokenID *big.Int
	MintID  string
	Timeout time.Time
}

func (el *eventListener) SubscribeForTokenBoughtEvent(tokenID *big.Int, mintID string, timeout time.Time) error {
	subscription := &tokenBoughtSubscription{
		TokenID: tokenID,
		MintID:  mintID,
		Timeout: timeout,
	}
	unsubscriber := &unsubscriber{}
	if err := el.registerEVMTokenBoughtSubscription(unsubscriber, subscription); err != nil {
		return err
	}

	el.startTokenBoughtTimeoutTimer(unsubscriber, subscription)
	el.addUnsubscriber(*unsubscriber)
	return nil
}

func (el *eventListener) registerEVMTokenBoughtSubscription(unsubscriber *unsubscriber, subscription *tokenBoughtSubscription) error {
	unsubscribeFunc, err := el.eventListener.RegisterTokenBoughtHandler(
		el.bookingTokenAddress,
		[]*big.Int{subscription.TokenID},
		nil,
		func(event any) {
			unsubscriber.timeoutTimer.Stop()
			el.logger.Infof("Token bought event received for token %s", subscription.TokenID.String())
			tokenBoughtEvent := event.(*bookingtoken.BookingtokenTokenBought)
			if err := el.partnerPlugin.SendTokenBoughtNotification(context.Background(), subscription.TokenID, subscription.MintID, tokenBoughtEvent.Raw.TxHash); err != nil {
				el.logger.Errorf("error calling partner plugin TokenBoughtNotification service: %v", err)
			}
		},
	)
	if err != nil {
		el.logger.Errorf("error registering token bought handler: %v", err)
		return err
	}
	unsubscriber.unsubscribe = unsubscribeFunc
	return nil
}

func (el *eventListener) startTokenBoughtTimeoutTimer(subscriptionCanceller *unsubscriber, subscription *tokenBoughtSubscription) {
	subscriptionCanceller.timeoutTimer = time.AfterFunc(time.Until(subscription.Timeout), func() {
		subscriptionCanceller.unsubscribe()
		el.logger.Infof("Token %s expired", subscription.TokenID.String())
		if err := el.partnerPlugin.SendTokenExpiredNotification(context.Background(), subscription.TokenID, subscription.MintID); err != nil {
			el.logger.Errorf("error calling partner plugin TokenExpiredNotification service: %v", err)
		}
	})
}
