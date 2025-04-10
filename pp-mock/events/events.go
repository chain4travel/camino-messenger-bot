// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/chain4travel/camino-messenger-bot/pp-mock/proto/pb/events"
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
	events.MyEventsServiceServer

	Start(ctx context.Context)
}

type server struct {
	events.UnimplementedMyEventsServiceServer

	eventChan              chan []byte
	subscriptionChans      map[string]chan []byte
	subscriptionChansMutex sync.RWMutex
	stopChan               chan struct{}
}

func NewServer() (Server, Sender) {
	eventChan := make(chan []byte)
	stopChan := make(chan struct{})
	server := &server{
		eventChan:         eventChan,
		subscriptionChans: make(map[string]chan []byte),
		stopChan:          stopChan,
	}
	sender := &eventSender{
		eventChan: eventChan,
		stopCh:    stopChan,
	}
	return server, sender

}

func (s *server) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case event := <-s.eventChan:
				s.propagate(event)
			case <-ctx.Done():
				s.stop()
				return
			}
		}
	}()
}

func (s *server) stop() {
	close(s.eventChan)
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
func (s *server) Subscribe(_ *emptypb.Empty, stream events.MyEventsService_SubscribeServer) error {
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

type Sender interface {
	SendProtoEventAsync(event proto.Message) error
}

type eventSender struct {
	stopCh    chan struct{}
	eventChan chan []byte
}

func (e *eventSender) SendProtoEventAsync(event proto.Message) error {
	log.Printf("Sending event: %T: %s", event, protoMessageToJSON(event))
	eventBytes, err := proto.Marshal(event)
	if err != nil { // should never happen
		return err
	}

	select {
	case <-e.stopCh:
		log.Printf("Sender is stopped, event sending aborted")
		return nil
	default:
	}
	e.eventChan <- eventBytes

	return nil
}

type dummySender struct{}

func NewDummySender() Sender {
	return &dummySender{}
}

func (d *dummySender) SendProtoEventAsync(proto.Message) error {
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
