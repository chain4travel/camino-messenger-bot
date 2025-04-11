// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-bot/pkg/booking"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
)

const timeCheckBlockchainDelay = 2 * time.Second

type TokenBoughtSubscriptionStorage interface {
	RemoveTokenBoughtSubscription(ctx context.Context, session Session, tokenID *big.Int) error
	AddTokenBoughtSubscription(context.Context, Session, *TokenBoughtSubscription) error
	GetAllTokenBoughtSubscriptions(context.Context, Session) ([]TokenBoughtSubscription, error)
	GetTokenBoughtSubscription(ctx context.Context, session Session, tokenID *big.Int) (*TokenBoughtSubscription, error)
	GetTokenBoughtSubscriptionByMinTimeout(ctx context.Context, session Session) (*TokenBoughtSubscription, error)
}

type TokenBoughtSubscriber interface {
	SubscribeTokenBoughtEvent(ctx context.Context, tokenID *big.Int, mintID string, timeout time.Time) error
}

type TokenBoughtSubscription struct {
	TokenID *big.Int
	MintID  string
	Timeout time.Time
}

func (el *eventListener) SubscribeTokenBoughtEvent(ctx context.Context, tokenID *big.Int, mintID string, timeout time.Time) error {
	subscription := &TokenBoughtSubscription{
		TokenID: tokenID,
		MintID:  mintID,
		// blockchain node time might be different, so we add some delay
		// to be more sure that we are not to early, assuming timeout has already happened
		Timeout: timeout.Add(timeCheckBlockchainDelay),
	}

	session, err := el.storage.NewSession(ctx)
	if err != nil {
		el.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer el.storage.Abort(session)

	if err := el.storage.AddTokenBoughtSubscription(ctx, session, subscription); err != nil {
		el.logger.Errorf("error adding token bought subscription: %v", err)
		return err
	}

	if err := el.storage.Commit(session); err != nil {
		el.logger.Errorf("failed to commit session: %v", err)
		return err
	}

	el.resetTokenBoughtTimerIfAfter(subscription)

	return nil
}

func (el *eventListener) startTokenBoughtSubscriptions(ctx context.Context) error {
	nextToExpireSubscription, err := el.tokenBoughtSubscriptionsStartupCheck(ctx)
	if err != nil {
		el.logger.Errorf("error checking token bought subscriptions: %v", err)
		return err
	}
	el.unsubscribeTokenBought = el.subscriber.SubscribeTokenBought(el.tokenBoughtEventHandler)
	el.startTokenBoughtTimer(nextToExpireSubscription)
	return nil
}

func (el *eventListener) tokenBoughtSubscriptionsStartupCheck(ctx context.Context) (*TokenBoughtSubscription, error) {
	tokenBoughtSubscriptions, err := el.getAllTokenBoughtSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	nextTimeoutTime := maxTime
	var nextToExpireSubscription *TokenBoughtSubscription

	for _, subscription := range tokenBoughtSubscriptions {
		status, err := el.bookingService.GetBookingStatus(ctx, el.startingBlockNumber, subscription.TokenID)
		if err != nil {
			el.logger.Errorf("failed to get booking status: %v", err)
			return nil, err
		}

		if status != booking.StatusReserved {
			// if token is expired or cancelled already, we don't send any notification here
			if status == booking.StatusBought {
				if err := el.partnerPlugin.SendTokenBoughtNotificationWithoutBuyTx(ctx, subscription.TokenID, subscription.MintID); err != nil {
					el.logger.Errorf("error sending token bought notification: %v", err)
					return nil, err
				}
			}

			session, err := el.storage.NewSession(ctx)
			if err != nil {
				el.logger.Errorf("failed to create storage session: %v", err)
				return nil, err
			}
			defer el.storage.Abort(session)

			if err := el.storage.RemoveTokenBoughtSubscription(ctx, session, subscription.TokenID); err != nil {
				el.logger.Errorf("error removing token bought subscription: %v", err)
				return nil, err
			}

			if err := el.storage.Commit(session); err != nil {
				el.logger.Errorf("failed to commit session: %v", err)
				return nil, err
			}

			continue
		}

		if subscription.Timeout.Before(nextTimeoutTime) {
			nextTimeoutTime = subscription.Timeout
			nextToExpireSubscription = &subscription
		}
	}

	return nextToExpireSubscription, nil
}

func (el *eventListener) getAllTokenBoughtSubscriptions(ctx context.Context) ([]TokenBoughtSubscription, error) {
	session, err := el.storage.NewSession(ctx)
	if err != nil {
		el.logger.Errorf("failed to create storage session: %v", err)
		return nil, err
	}
	defer el.storage.Abort(session)

	tokenBoughtSubscriptions, err := el.storage.GetAllTokenBoughtSubscriptions(ctx, session)
	if err != nil {
		el.logger.Errorf("error getting token bought subscription from db: %v", err)
		return nil, err
	}

	return tokenBoughtSubscriptions, nil
}

func (el *eventListener) tokenBoughtEventHandler(event *bookingtoken.BookingtokenTokenBought) uint64 {
	el.logger.Infof("Token bought event received for token %s", event.TokenId.String())

	ctx := context.Background()

	// regardless if we succeed in updating db and sending notification, we need to reset timer
	defer el.resetTokenBoughtTimerIfMatch(ctx, event.TokenId)

	session, err := el.storage.NewSession(ctx)
	if err != nil {
		el.logger.Errorf("failed to create storage session: %v", err)
		return 0
	}
	defer el.storage.Abort(session)

	subscription, err := el.storage.GetTokenBoughtSubscription(ctx, session, event.TokenId)
	switch {
	case errors.Is(err, ErrNotFound):
		return event.Raw.BlockNumber
	case err != nil:
		el.logger.Errorf("error getting token bought subscription from db: %v", err)
		return 0
	}

	if err := el.partnerPlugin.SendTokenBoughtNotificationWithBuyTx(ctx, subscription.TokenID, subscription.MintID, event.Raw.TxHash); err != nil {
		el.logger.Errorf("error calling partner plugin TokenBoughtNotification service: %v", err)
		return 0
	}

	if err := el.storage.RemoveTokenBoughtSubscription(ctx, session, subscription.TokenID); err != nil {
		el.logger.Errorf("error removing token bought subscription from db: %v", err)
		return 0
	}

	if err := el.storage.Commit(session); err != nil {
		el.logger.Errorf("failed to commit session: %v", err)
		return 0
	}

	return event.Raw.BlockNumber
}

// If this ever runs into endless loop of failing-to-remove-from-db
// and restarting timer with same subscription again,
// we can add in-mem struct that will track number of attempts per tokenID
// and skip this tokenID and try another one
func (el *eventListener) tokenBoughtTimeoutHandler() {
	ctx := context.Background()

	// regardless if we succeed in updating db and sending notification, we need to reset timer
	defer el.resetTokenBoughtTimer(ctx)

	status, err := el.bookingService.GetBookingStatus(ctx, nil, el.tokenBoughtTimerSubscription.TokenID)
	if err != nil {
		el.logger.Errorf("failed to get booking status: %v", err)
		return
	}

	if status != booking.StatusReserved {
		// token already bought or cancelled and this event has been handled in other place
		return
	}

	el.logger.Infof("Token %s expired", el.tokenBoughtTimerSubscription.TokenID.String())

	session, err := el.storage.NewSession(ctx)
	if err != nil {
		el.logger.Errorf("failed to create storage session: %v", err)
		return
	}
	defer el.storage.Abort(session)

	if err := el.partnerPlugin.SendTokenExpiredNotification(ctx, el.tokenBoughtTimerSubscription.TokenID, el.tokenBoughtTimerSubscription.MintID); err != nil {
		el.logger.Errorf("error calling partner plugin TokenExpiredNotification service: %v", err)
		return
	}

	if el.recordTokenExpiration {
		if _, err := el.bookingService.RecordExpiration(ctx, el.tokenBoughtTimerSubscription.TokenID); err != nil {
			el.logger.Errorf("error calling booking service RecordExpiration: %v", err)
			return
		}
	}

	if err := el.storage.RemoveTokenBoughtSubscription(ctx, session, el.tokenBoughtTimerSubscription.TokenID); err != nil {
		el.logger.Errorf("error removing token bought subscription: %v", err)
		return
	}

	if err := el.storage.Commit(session); err != nil {
		el.logger.Errorf("failed to commit session: %v", err)
		return
	}
}

func (el *eventListener) startTokenBoughtTimer(nextToExpireSubscription *TokenBoughtSubscription) {
	el.tokenBoughtTimerSubscription = nextToExpireSubscription
	if nextToExpireSubscription == nil {
		el.tokenBoughtTimer = time.NewTimer(0) // dummy timer, so it will never be nil
	} else {
		el.tokenBoughtTimer = time.AfterFunc(timeUntil(nextToExpireSubscription.Timeout), el.tokenBoughtTimeoutHandler)
	}
}

func (el *eventListener) resetTokenBoughtTimerIfAfter(newSubscription *TokenBoughtSubscription) {
	el.tokenBoughtTimerMutex.Lock()
	defer el.tokenBoughtTimerMutex.Unlock()

	if el.tokenBoughtTimerSubscription != nil && !el.tokenBoughtTimerSubscription.Timeout.After(newSubscription.Timeout) {
		return
	}

	el.tokenBoughtTimer.Stop()
	el.tokenBoughtTimerSubscription = newSubscription
	el.tokenBoughtTimer = time.AfterFunc(timeUntil(newSubscription.Timeout), el.tokenBoughtTimeoutHandler)
}

func (el *eventListener) resetTokenBoughtTimerIfMatch(ctx context.Context, tokenID *big.Int) {
	el.tokenBoughtTimerMutex.Lock()
	defer el.tokenBoughtTimerMutex.Unlock()

	if el.tokenBoughtTimerSubscription != nil && el.tokenBoughtTimerSubscription.TokenID.Cmp(tokenID) != 0 {
		return
	}

	el.resetTokenBoughtTimerFromDB(ctx)
}

func (el *eventListener) resetTokenBoughtTimer(ctx context.Context) {
	el.tokenBoughtTimerMutex.Lock()
	defer el.tokenBoughtTimerMutex.Unlock()
	el.resetTokenBoughtTimerFromDB(ctx)
}

func (el *eventListener) resetTokenBoughtTimerFromDB(ctx context.Context) {
	el.tokenBoughtTimer.Stop()
	el.tokenBoughtTimerSubscription = nil

	session, err := el.storage.NewSession(ctx)
	if err != nil {
		el.logger.Errorf("failed to create storage session: %v", err)
		return
	}
	defer el.storage.Abort(session)

	nextToExpireSubscription, err := el.storage.GetTokenBoughtSubscriptionByMinTimeout(ctx, session)
	switch {
	case errors.Is(err, ErrNotFound): // do nothing
	case err != nil:
		el.logger.Errorf("error getting token bought subscription: %v", err)
		return
	case nextToExpireSubscription != nil:
		el.tokenBoughtTimerSubscription = nextToExpireSubscription
		el.tokenBoughtTimer = time.AfterFunc(timeUntil(nextToExpireSubscription.Timeout), el.tokenBoughtTimeoutHandler)
	}
}

func (el *eventListener) stopTokenBoughtTimer() {
	el.tokenBoughtTimerMutex.Lock()
	defer el.tokenBoughtTimerMutex.Unlock()
	el.tokenBoughtTimer.Stop()
}

func timeUntil(t time.Time) time.Duration {
	now := time.Now()
	if t.Before(now) {
		return 0
	}
	return t.Sub(now)
}
