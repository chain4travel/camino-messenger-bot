// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/pkg/booking"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
)

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

func (l *eventListener) SubscribeTokenBoughtEvent(ctx context.Context, tokenID *big.Int, mintID string, timeout time.Time) error {
	subscription := &TokenBoughtSubscription{
		TokenID: tokenID,
		MintID:  mintID,
		// blockchain node time might be different, so we add some delay
		// to be more sure that we are not to early, assuming timeout has already happened
		Timeout: timeout.Add(timeCheckBlockchainDelay),
	}

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer l.storage.Abort(session)

	if err := l.storage.AddTokenBoughtSubscription(ctx, session, subscription); err != nil {
		l.logger.Errorf("error adding token bought subscription: %v", err)
		return err
	}

	if err := l.storage.Commit(session); err != nil {
		l.logger.Errorf("failed to commit session: %v", err)
		return err
	}

	l.resetTokenBoughtTimerIfAfter(subscription)

	return nil
}

func (l *eventListener) startTokenBoughtSubscriptions(ctx context.Context) error {
	nextToExpireSubscription, err := l.tokenBoughtSubscriptionsStartupCheck(ctx)
	if err != nil {
		l.logger.Errorf("error checking token bought subscriptions: %v", err)
		return err
	}
	l.unsubscribeTokenBought = l.subscriber.SubscribeTokenBought(l.tokenBoughtEventHandler)
	l.startTokenBoughtTimer(nextToExpireSubscription)
	return nil
}

func (l *eventListener) tokenBoughtSubscriptionsStartupCheck(ctx context.Context) (*TokenBoughtSubscription, error) {
	tokenBoughtSubscriptions, err := l.getAllTokenBoughtSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	nextTimeoutTime := maxTime
	var nextToExpireSubscription *TokenBoughtSubscription

	for _, subscription := range tokenBoughtSubscriptions {
		status, err := l.bookingService.GetBookingStatus(ctx, l.startingBlockNumber, subscription.TokenID)
		if err != nil {
			l.logger.Errorf("failed to get booking status: %v", err)
			return nil, err
		}

		if status != booking.StatusReserved {
			// if token is expired or cancelled already, we don't send any notification here
			if status == booking.StatusBought {
				if err := l.partnerPlugin.TokenBoughtNotificationWithoutBuyTx(ctx, subscription.TokenID, subscription.MintID); err != nil {
					l.logger.Errorf("error sending token bought notification: %v", err)
					return nil, err
				}
			}

			session, err := l.storage.NewSession(ctx)
			if err != nil {
				l.logger.Errorf("failed to create storage session: %v", err)
				return nil, err
			}
			defer l.storage.Abort(session)

			if err := l.storage.RemoveTokenBoughtSubscription(ctx, session, subscription.TokenID); err != nil {
				l.logger.Errorf("error removing token bought subscription: %v", err)
				return nil, err
			}

			if err := l.storage.Commit(session); err != nil {
				l.logger.Errorf("failed to commit session: %v", err)
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

func (l *eventListener) getAllTokenBoughtSubscriptions(ctx context.Context) ([]TokenBoughtSubscription, error) {
	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return nil, err
	}
	defer l.storage.Abort(session)

	tokenBoughtSubscriptions, err := l.storage.GetAllTokenBoughtSubscriptions(ctx, session)
	if err != nil {
		l.logger.Errorf("error getting all token bought subscriptions from db: %v", err)
		return nil, err
	}

	return tokenBoughtSubscriptions, nil
}

func (l *eventListener) tokenBoughtEventHandler(event *bookingtoken.BookingtokenTokenBought) uint64 {
	l.logger.Infof("Token bought event received for token %s", event.TokenId.String())

	ctx := context.Background()

	// regardless if we succeed in updating db and sending notification, we need to reset timer
	defer l.resetTokenBoughtTimerIfMatch(ctx, event.TokenId)

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return 0
	}
	defer l.storage.Abort(session)

	subscription, err := l.storage.GetTokenBoughtSubscription(ctx, session, event.TokenId)
	switch {
	case errors.Is(err, ErrNotFound):
		l.logger.Infof("Ignoring token bought event for token %s, subscription does not exist", event.TokenId.String())
		return event.Raw.BlockNumber
	case err != nil:
		l.logger.Errorf("error getting token bought subscription from db: %v", err)
		return 0
	}

	if err := l.partnerPlugin.TokenBoughtNotificationWithBuyTx(ctx, subscription.TokenID, subscription.MintID, event.Raw.TxHash); err != nil {
		l.logger.Errorf("error calling partner plugin TokenBoughtNotification service: %v", err)
		return 0
	}

	if isCancellable, err := l.bookingService.IsBookingCancellable(ctx, nil, subscription.TokenID); err != nil {
		l.logger.Errorf("failed to get booking status: %v", err)
		return 0
	} else if isCancellable {
		if err := l.storage.AddCancellationSubscription(ctx, session, subscription.TokenID); err != nil {
			l.logger.Errorf("Error subscribing for cancellation events as supplier (tokenID: %d, mintID: %s): %v", subscription.TokenID.Int64(), subscription.MintID, err)
			return 0
		}
		l.logger.Infof("Subscribed for cancellation events as supplier for token %s", subscription.TokenID.String())
	}

	if err := l.storage.RemoveTokenBoughtSubscription(ctx, session, subscription.TokenID); err != nil {
		l.logger.Errorf("error removing token bought subscription from db: %v", err)
		return 0
	}

	if err := l.storage.Commit(session); err != nil {
		l.logger.Errorf("failed to commit session: %v", err)
		return 0
	}

	return event.Raw.BlockNumber
}

// If this ever runs into endless loop of failing-to-remove-from-db
// and restarting timer with same subscription again,
// we can add in-mem struct that will track number of attempts per tokenID
// and skip this tokenID and try another one
func (l *eventListener) tokenBoughtTimeoutHandler() {
	ctx := context.Background()

	// regardless if we succeed in updating db and sending notification, we need to reset timer
	defer l.resetTokenBoughtTimer(ctx)

	status, err := l.bookingService.GetBookingStatus(ctx, nil, l.tokenBoughtTimerSubscription.TokenID)
	if err != nil {
		l.logger.Errorf("failed to get booking status: %v", err)
		return
	}

	if status != booking.StatusReserved {
		// token already bought or cancelled and this event has been handled in other place
		return
	}

	l.logger.Infof("Token %s expired", l.tokenBoughtTimerSubscription.TokenID.String())

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return
	}
	defer l.storage.Abort(session)

	if err := l.partnerPlugin.TokenExpiredNotification(ctx, l.tokenBoughtTimerSubscription.TokenID, l.tokenBoughtTimerSubscription.MintID); err != nil {
		l.logger.Errorf("error calling partner plugin TokenExpiredNotification service: %v", err)
		return
	}

	if l.recordTokenExpiration {
		if _, err := l.bookingService.RecordExpiration(ctx, l.tokenBoughtTimerSubscription.TokenID); err != nil {
			l.logger.Errorf("error calling booking service RecordExpiration: %v", err)
			return
		}
	}

	if err := l.storage.RemoveTokenBoughtSubscription(ctx, session, l.tokenBoughtTimerSubscription.TokenID); err != nil {
		l.logger.Errorf("error removing token bought subscription: %v", err)
		return
	}

	if err := l.storage.Commit(session); err != nil {
		l.logger.Errorf("failed to commit session: %v", err)
		return
	}
}

func (l *eventListener) startTokenBoughtTimer(nextToExpireSubscription *TokenBoughtSubscription) {
	l.tokenBoughtTimerSubscription = nextToExpireSubscription
	if nextToExpireSubscription == nil {
		l.tokenBoughtTimer = time.NewTimer(0) // dummy timer, so it will never be nil
	} else {
		l.tokenBoughtTimer = time.AfterFunc(timeUntil(nextToExpireSubscription.Timeout), l.tokenBoughtTimeoutHandler)
	}
}

func (l *eventListener) resetTokenBoughtTimerIfAfter(newSubscription *TokenBoughtSubscription) {
	l.tokenBoughtTimerMutex.Lock()
	defer l.tokenBoughtTimerMutex.Unlock()

	if l.tokenBoughtTimerSubscription != nil && !l.tokenBoughtTimerSubscription.Timeout.After(newSubscription.Timeout) {
		return
	}

	l.tokenBoughtTimer.Stop()
	l.tokenBoughtTimerSubscription = newSubscription
	l.tokenBoughtTimer = time.AfterFunc(timeUntil(newSubscription.Timeout), l.tokenBoughtTimeoutHandler)
}

func (l *eventListener) resetTokenBoughtTimerIfMatch(ctx context.Context, tokenID *big.Int) {
	l.tokenBoughtTimerMutex.Lock()
	defer l.tokenBoughtTimerMutex.Unlock()

	if l.tokenBoughtTimerSubscription != nil && l.tokenBoughtTimerSubscription.TokenID.Cmp(tokenID) != 0 {
		return
	}

	l.resetTokenBoughtTimerFromDB(ctx)
}

func (l *eventListener) resetTokenBoughtTimer(ctx context.Context) {
	l.tokenBoughtTimerMutex.Lock()
	defer l.tokenBoughtTimerMutex.Unlock()
	l.resetTokenBoughtTimerFromDB(ctx)
}

func (l *eventListener) resetTokenBoughtTimerFromDB(ctx context.Context) {
	l.tokenBoughtTimer.Stop()
	l.tokenBoughtTimerSubscription = nil

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return
	}
	defer l.storage.Abort(session)

	nextToExpireSubscription, err := l.storage.GetTokenBoughtSubscriptionByMinTimeout(ctx, session)
	switch {
	case errors.Is(err, ErrNotFound): // do nothing
	case err != nil:
		l.logger.Errorf("error getting token bought subscription: %v", err)
		return
	case nextToExpireSubscription != nil:
		l.tokenBoughtTimerSubscription = nextToExpireSubscription
		l.tokenBoughtTimer = time.AfterFunc(timeUntil(nextToExpireSubscription.Timeout), l.tokenBoughtTimeoutHandler)
	}
}

func (l *eventListener) stopTokenBoughtTimer() {
	l.tokenBoughtTimerMutex.Lock()
	defer l.tokenBoughtTimerMutex.Unlock()
	l.tokenBoughtTimer.Stop()
}

func timeUntil(t time.Time) time.Duration {
	now := time.Now()
	if t.Before(now) {
		return 0
	}
	return t.Sub(now)
}
