// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package eventlistener

import (
	"context"
	"fmt"
	"math/big"

	cancellationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/cancellation/v1"
	notificationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pkg/booking"
	"github.com/chain4travel/camino-messenger-contracts/go/contracts/bookingtoken"
)

const zeroHashStr = "0x0000000000000000000000000000000000000000000000000000000000000000"

type CancellationSubscriptionStorage interface {
	RemoveCancellationSubscription(ctx context.Context, session Session, tokenID *big.Int) error
	AddCancellationSubscription(ctx context.Context, session Session, tokenID *big.Int) error
	GetAllCancellationSubscriptions(context.Context, Session) ([]*big.Int, error)
	IsCancellationSubscriptionExist(ctx context.Context, session Session, tokenID *big.Int) (bool, error)
}

type CancellationSubscriber interface {
	SubscribeCancellationEvents(ctx context.Context, tokenID *big.Int) error
}

func (l *eventListener) SubscribeCancellationEvents(ctx context.Context, tokenID *big.Int) error {
	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return err
	}
	defer l.storage.Abort(session)

	if err := l.storage.AddCancellationSubscription(ctx, session, tokenID); err != nil {
		l.logger.Errorf("error adding cancellation events subscription: %v", err)
		return err
	}

	if err := l.storage.Commit(session); err != nil {
		l.logger.Errorf("failed to commit session: %v", err)
		return err
	}

	return nil
}

func (l *eventListener) startCancellationSubscriptions(ctx context.Context) error {
	if err := l.cancellationSubscriptionsStartupCheck(ctx); err != nil {
		l.logger.Errorf("error checking cancellation subscriptions: %v", err)
		return err
	}

	unsubscribeCancellationPending := l.subscriber.SubscribeCancellationPending(l.cancellationPendingEventHandler)
	unsubscribeCancellationRejected := l.subscriber.SubscribeCancellationRejected(l.cancellationRejectedEventHandler)
	unsubscribeCancellationWithdrawn := l.subscriber.SubscribeCancellationWithdrawn(l.cancellationWithdrawnEventHandler)
	unsubscribeCancellationFinalized := l.subscriber.SubscribeCancellationFinalized(l.cancellationFinalizedEventHandler)

	l.unsubscribeCancellation = func() {
		unsubscribeCancellationPending()
		unsubscribeCancellationRejected()
		unsubscribeCancellationWithdrawn()
		unsubscribeCancellationFinalized()
	}
	return nil
}

func (l *eventListener) cancellationSubscriptionsStartupCheck(ctx context.Context) error {
	cancellationSubscriptionTokenIDs, err := l.getAllCancellationSubscriptionTokenIDs(ctx)
	if err != nil {
		return err
	}

	for _, tokenID := range cancellationSubscriptionTokenIDs {
		cancellation, err := l.bookingService.GetCancellationProposal(ctx, l.startingBlockNumber, tokenID)
		if err != nil {
			l.logger.Errorf("error getting cancellation proposal for token %s: %v", tokenID.String(), err)
			return err
		}

		switch cancellation.Status {
		case booking.CancellationProposalStatusPending:
			reasons, err := l.bookingService.GetCancellationReasons(ctx, l.startingBlockNumber, tokenID)
			if err != nil {
				l.logger.Errorf("error getting cancellation reasons for token %s: %v", tokenID.String(), err)
				return err
			}

			if err := l.partnerPlugin.CancellationPendingNotification(ctx, &notificationv3.CancellationPending{
				TokenId:            tokenID.Uint64(),
				InitialProposer:    &typesv4.EVMAddress{Address: cancellation.InitialProposer.Hex()},
				CurrentProposer:    &typesv4.EVMAddress{Address: cancellation.CurrentProposer.Hex()},
				RefundAmount:       cancellation.RefundAmount.Uint64(),
				OwnerAccepted:      cancellation.OwnerAccepted,
				SupplierAccepted:   cancellation.SupplierAccepted,
				TimesCountered:     uint64(cancellation.TimesCountered),
				TimesRejected:      uint64(cancellation.TimesRejected),
				TxId:               &typesv4.EVMTransactionID{Hash: zeroHashStr},
				CancellationReason: cancellationv1.CancellationReason(reasons.CancellationReason),
				RejectionReason:    cancellationv1.RejectionReason(reasons.RejectionReason),
				CounterReason:      cancellationv1.CounterReason(reasons.CounterReason),
				WithdrawalReason:   cancellationv1.WithdrawalReason(reasons.WithdrawalReason),
			}); err != nil {
				l.logger.Errorf("error sending CancellationPending notification: %v", err)
				return err
			}
		case booking.CancellationProposalStatusRejected:
			reasons, err := l.bookingService.GetCancellationReasons(ctx, l.startingBlockNumber, tokenID)
			if err != nil {
				l.logger.Errorf("error getting cancellation reasons for token %s: %v", tokenID.String(), err)
				return err
			}

			if err := l.partnerPlugin.CancellationRejectedNotification(ctx, &notificationv3.CancellationRejected{
				TokenId: tokenID.Uint64(),
				Reason:  cancellationv1.RejectionReason(reasons.RejectionReason),
				TxId:    &typesv4.EVMTransactionID{Hash: zeroHashStr},
			}); err != nil {
				l.logger.Errorf("error sending CancellationRejected notification: %v", err)
				return err
			}
		case booking.CancellationProposalStatusWithdrawn:
			reasons, err := l.bookingService.GetCancellationReasons(ctx, l.startingBlockNumber, tokenID)
			if err != nil {
				l.logger.Errorf("error getting cancellation reasons for token %s: %v", tokenID.String(), err)
				return err
			}

			if err := l.partnerPlugin.CancellationWithdrawnNotification(ctx, &notificationv3.CancellationWithdrawn{
				TokenId: tokenID.Uint64(),
				Reason:  cancellationv1.WithdrawalReason(reasons.WithdrawalReason),
				TxId:    &typesv4.EVMTransactionID{Hash: zeroHashStr},
			}); err != nil {
				l.logger.Errorf("error sending CancellationWithdrawn notification: %v", err)
				return err
			}
		case booking.CancellationProposalStatusFinalized:
			session, err := l.storage.NewSession(ctx)
			if err != nil {
				l.logger.Errorf("failed to create storage session: %v", err)
				return err
			}
			defer l.storage.Abort(session)

			if err := l.partnerPlugin.CancellationFinalizedNotification(ctx, &notificationv3.CancellationFinalized{
				TokenId: tokenID.Uint64(),
				TxId:    &typesv4.EVMTransactionID{Hash: zeroHashStr},
			}); err != nil {
				l.logger.Errorf("error sending CancellationFinalized notification: %v", err)
				return err
			}

			if err := l.storage.RemoveCancellationSubscription(ctx, session, tokenID); err != nil {
				l.logger.Errorf("error removing CancellationFinalized subscription from db: %v", err)
				return err
			}

			if err := l.storage.Commit(session); err != nil {
				l.logger.Errorf("failed to commit session: %v", err)
				return err
			}
		default:
			err := fmt.Errorf("unknown cancellation proposal status for token %s: %d", tokenID.String(), cancellation.Status)
			l.logger.Error(err)
			return err
		}
	}
	return nil
}

func (l *eventListener) getAllCancellationSubscriptionTokenIDs(ctx context.Context) ([]*big.Int, error) {
	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return nil, err
	}
	defer l.storage.Abort(session)

	cancellationSubscriptionTokenIDs, err := l.storage.GetAllCancellationSubscriptions(ctx, session)
	if err != nil {
		l.logger.Errorf("error getting all cancellation subscriptions from db: %v", err)
		return nil, err
	}

	return cancellationSubscriptionTokenIDs, nil
}

func (l *eventListener) cancellationPendingEventHandler(event *bookingtoken.BookingtokenCancellationPending) uint64 {
	l.logger.Infof("CancellationPending event received for token %s", event.TokenId.String())

	ctx := context.Background()

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return 0
	}
	defer l.storage.Abort(session)

	ok, err := l.storage.IsCancellationSubscriptionExist(ctx, session, event.TokenId)
	switch {
	case err != nil:
		l.logger.Errorf("error checking if cancellation subscription exists in db: %v", err)
		return 0
	case !ok:
		l.logger.Infof("Ignoring CancellationPending event for token %s, subscription does not exist", event.TokenId.String())
		return event.Raw.BlockNumber
	}

	reasons, err := l.bookingService.GetCancellationReasonsEvent(ctx, event.Raw.TxHash)
	if err != nil {
		l.logger.Errorf("error getting cancellation reasons event: %v", err)
		return 0
	}

	if err := l.partnerPlugin.CancellationPendingNotification(ctx, &notificationv3.CancellationPending{
		TokenId:            event.TokenId.Uint64(),
		InitialProposer:    &typesv4.EVMAddress{Address: event.InitialProposer.Hex()},
		CurrentProposer:    &typesv4.EVMAddress{Address: event.CurrentProposer.Hex()},
		RefundAmount:       event.RefundAmount.Uint64(),
		OwnerAccepted:      event.OwnerAccepted,
		SupplierAccepted:   event.SupplierAccepted,
		TimesCountered:     uint64(event.TimesCountered),
		TimesRejected:      uint64(event.TimesRejected),
		TxId:               &typesv4.EVMTransactionID{Hash: event.Raw.TxHash.Hex()},
		CancellationReason: cancellationv1.CancellationReason(reasons.CancellationReason),
		RejectionReason:    cancellationv1.RejectionReason(reasons.RejectionReason),
		CounterReason:      cancellationv1.CounterReason(reasons.CounterReason),
		WithdrawalReason:   cancellationv1.WithdrawalReason(reasons.WithdrawalReason),
	}); err != nil {
		l.logger.Errorf("error sending CancellationPending notification: %v", err)
		return 0
	}

	return event.Raw.BlockNumber
}

func (l *eventListener) cancellationRejectedEventHandler(event *bookingtoken.BookingtokenCancellationRejected) uint64 {
	l.logger.Infof("CancellationRejected event received for token %s", event.TokenId.String())

	ctx := context.Background()

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return 0
	}
	defer l.storage.Abort(session)

	ok, err := l.storage.IsCancellationSubscriptionExist(ctx, session, event.TokenId)
	switch {
	case err != nil:
		l.logger.Errorf("error checking if cancellation subscription exists in db: %v", err)
		return 0
	case !ok:
		return event.Raw.BlockNumber
	}

	if err := l.partnerPlugin.CancellationRejectedNotification(ctx, &notificationv3.CancellationRejected{
		TokenId: event.TokenId.Uint64(),
		Reason:  cancellationv1.RejectionReason(event.RejectionReason),
		TxId:    &typesv4.EVMTransactionID{Hash: event.Raw.TxHash.Hex()},
	}); err != nil {
		l.logger.Errorf("error sending CancellationRejected notification: %v", err)
		return 0
	}

	return event.Raw.BlockNumber
}

func (l *eventListener) cancellationWithdrawnEventHandler(event *bookingtoken.BookingtokenCancellationWithdrawn) uint64 {
	l.logger.Infof("CancellationWithdrawn event received for token %s", event.TokenId.String())

	ctx := context.Background()

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return 0
	}
	defer l.storage.Abort(session)

	ok, err := l.storage.IsCancellationSubscriptionExist(ctx, session, event.TokenId)
	switch {
	case err != nil:
		l.logger.Errorf("error checking if cancellation subscription exists in db: %v", err)
		return 0
	case !ok:
		return event.Raw.BlockNumber
	}

	if err := l.partnerPlugin.CancellationWithdrawnNotification(ctx, &notificationv3.CancellationWithdrawn{
		TokenId: event.TokenId.Uint64(),
		Reason:  cancellationv1.WithdrawalReason(event.WithdrawalReason),
		TxId:    &typesv4.EVMTransactionID{Hash: event.Raw.TxHash.Hex()},
	}); err != nil {
		l.logger.Errorf("error sending CancellationWithdrawn notification: %v", err)
		return 0
	}

	return event.Raw.BlockNumber
}

func (l *eventListener) cancellationFinalizedEventHandler(event *bookingtoken.BookingtokenCancellationFinalized) uint64 {
	l.logger.Infof("CancellationFinalized event received for token %s", event.TokenId.String())

	ctx := context.Background()

	session, err := l.storage.NewSession(ctx)
	if err != nil {
		l.logger.Errorf("failed to create storage session: %v", err)
		return 0
	}
	defer l.storage.Abort(session)

	ok, err := l.storage.IsCancellationSubscriptionExist(ctx, session, event.TokenId)
	switch {
	case err != nil:
		l.logger.Errorf("error checking if cancellation subscription exists in db: %v", err)
		return 0
	case !ok:
		return event.Raw.BlockNumber
	}

	if err := l.partnerPlugin.CancellationFinalizedNotification(ctx, &notificationv3.CancellationFinalized{
		TokenId: event.TokenId.Uint64(),
		TxId:    &typesv4.EVMTransactionID{Hash: event.Raw.TxHash.Hex()},
	}); err != nil {
		l.logger.Errorf("error sending CancellationFinalized notification: %v", err)
		return 0
	}

	if err := l.storage.RemoveCancellationSubscription(ctx, session, event.TokenId); err != nil {
		l.logger.Errorf("error removing CancellationFinalized subscription from db: %v", err)
		return 0
	}

	if err := l.storage.Commit(session); err != nil {
		l.logger.Errorf("failed to commit session: %v", err)
		return 0
	}

	return event.Raw.BlockNumber
}
