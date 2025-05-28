// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/proto/pb/events"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	_ Server = (*server)(nil)
	_ Sender = (*eventSender)(nil)
	_ Sender = (*dummySender)(nil)
)

type Server interface {
	events.EventsServiceServer

	Start(ctx context.Context)
}

type Sender interface {
	SendProtoEvent(event proto.Message) error
}

type server struct {
	events.UnimplementedEventsServiceServer

	subscriptionChans      map[string]chan []byte
	subscriptionChansMutex sync.RWMutex
	stopChan               chan struct{}
	sender                 *eventSender
}

func NewServer() (Server, Sender) {
	server := &server{
		subscriptionChans: make(map[string]chan []byte),
		stopChan:          make(chan struct{}),
		sender:            &eventSender{eventChan: make(chan []byte)},
	}
	return server, server.sender
}

func (s *server) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case event := <-s.sender.eventChan:
				s.propagate(event)
			case <-ctx.Done():
				s.stop()
				return
			}
		}
	}()
}

func (s *server) stop() {
	s.sender.stop()
	close(s.stopChan)
}

func (s *server) subscribe() (string, chan []byte) {
	s.subscriptionChansMutex.Lock()
	defer s.subscriptionChansMutex.Unlock()

	subscriptionID := fmt.Sprintf("%d", time.Now().UnixNano())
	ch := make(chan []byte)
	s.subscriptionChans[subscriptionID] = ch

	return subscriptionID, ch
}

func (s *server) unsubscribe(subscriptionID string) {
	s.subscriptionChansMutex.Lock()
	defer s.subscriptionChansMutex.Unlock()

	if ch, ok := s.subscriptionChans[subscriptionID]; ok {
		close(ch)
		delete(s.subscriptionChans, subscriptionID)
	}
}

func (s *server) propagate(event []byte) {
	s.subscriptionChansMutex.RLock()
	defer s.subscriptionChansMutex.RUnlock()
	for _, ch := range s.subscriptionChans {
		ch <- event
	}
}

// Subscribe implements the server-side streaming RPC.
func (s *server) Subscribe(_ *emptypb.Empty, stream events.EventsService_SubscribeServer) error {
	subscriptionID, subscriptionChan := s.subscribe()
	defer s.unsubscribe(subscriptionID)

	for {
		select {
		case event := <-subscriptionChan:
			log.Printf("Sending event to stream: %s", string(event))
			if err := stream.Send(&events.SubscribeResponse{Data: event}); err != nil {
				return err
			}
		case <-s.stopChan:
			return nil
		case <-stream.Context().Done():
			return nil
		}
	}
}

type eventSender struct {
	sendMutex sync.Mutex
	isStopped bool
	eventChan chan []byte
}

func (e *eventSender) stop() {
	e.sendMutex.Lock()
	defer e.sendMutex.Unlock()

	e.isStopped = true
	for range e.eventChan { //nolint:revive
	} // drain channel
	close(e.eventChan)
}

func (e *eventSender) SendProtoEvent(event proto.Message) error {
	e.sendMutex.Lock()
	defer e.sendMutex.Unlock()

	if e.isStopped {
		return nil
	}

	log.Printf("Sending event: %T: %s", event, protoMessageToJSON(event))
	eventBytes, err := proto.Marshal(event)
	if err != nil { // should never happen
		return err
	}

	e.eventChan <- eventBytes

	return nil
}

type dummySender struct{}

func NewDummySender() Sender {
	return &dummySender{}
}

func (d *dummySender) SendProtoEvent(proto.Message) error {
	return nil
}

func protoMessageToJSON(message proto.Message) string {
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonData, err := marshaler.Marshal(message)
	if err != nil {
		return fmt.Sprintf("Error marshalling %T: %v", message, err)
	}
	return string(jsonData)
}
