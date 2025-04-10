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
	subscriptionChansMutex sync.Mutex
}

func NewServer() (Server, Sender) {
	eventChan := make(chan []byte)
	return &server{
		eventChan:         eventChan,
		subscriptionChans: make(map[string]chan []byte),
	}, &eventSender{eventChan: eventChan}
}

func (s *server) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case event := <-s.eventChan:
				s.propagate(event)
			case <-ctx.Done():
				s.stop()
			}
		}
	}()
}

func (s *server) stop() {
	close(s.eventChan)

	s.subscriptionChansMutex.Lock()
	defer s.subscriptionChansMutex.Unlock()

	for subscriptionID, ch := range s.subscriptionChans {
		close(ch)
		delete(s.subscriptionChans, subscriptionID)
	}
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
	s.subscriptionChansMutex.Lock()
	defer s.subscriptionChansMutex.Unlock()
	for _, ch := range s.subscriptionChans {
		go func() { ch <- event }()
	}
}

// Subscribe implements the server-side streaming RPC.
func (s *server) Subscribe(_ *emptypb.Empty, stream1 events.MyEventsService_SubscribeServer) error {
	subscriptionID, subscriptionChan := s.subscribe()
	defer s.unsubscribe(subscriptionID)

	for event := range subscriptionChan {
		log.Printf("Sending event to stream: %s", string(event))
		if err := stream1.Send(&events.SubscribeResponse{Data: event}); err != nil {
			return err
		}
	}

	return nil
}

type Sender interface {
	SendProtoEventAsync(event proto.Message) error
}

type eventSender struct {
	eventChan chan []byte
}

func (e *eventSender) SendProtoEventAsync(event proto.Message) error {
	log.Printf("Sending event: %T: %s", event, protoMessageToJSON(event))
	eventBytes, err := proto.Marshal(event)
	if err != nil { // should never happen
		return err
	}

	go func() {
		e.eventChan <- eventBytes
	}()

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
		panic(fmt.Sprintf("Error marshalling: %v", err))
	}
	return string(jsonData)
}
