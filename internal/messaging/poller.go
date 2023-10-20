package messaging

import (
	"camino-messenger-provider/internal/matrix"
	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
	"time"
)

type Poller interface {
	Start() error
	Stop()
}

type poller struct {
	matrixClient            matrix.Client
	scheduler               *gocron.Scheduler
	logger                  *zap.SugaredLogger
	roomChannel, msgChannel chan matrix.Rooms
	processor               Processor
}

func NewPoller(matrixClient matrix.Client, logger *zap.SugaredLogger, userID string) Poller {
	return &poller{
		matrixClient: matrixClient,
		logger:       logger,
		scheduler:    gocron.NewScheduler(time.UTC),
		roomChannel:  make(chan matrix.Rooms),
		msgChannel:   make(chan matrix.Rooms),
		processor:    NewProcessor(matrixClient, userID),
	}
}

func (p *poller) Start() error {
	p.logger.Info("Scheduling matrix syncing...")
	_, err := p.scheduler.Every(matrix.DefaultInterval).Do(p.matrixClient.Sync, p.roomChannel, p.msgChannel)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	p.logger.Info("Starting scheduler...")
	p.scheduler.StartAsync()

	select {
	case roomEvent := <-p.roomChannel:
		p.logger.Info("Received room event: ", roomEvent)
		err = p.processor.ProcessRoomInvitation(roomEvent)
		if err != nil {
			p.logger.Error(err)
		}
	case msgEvent := <-p.msgChannel:
		p.logger.Info("Received roomEvent event: ", msgEvent)
		err = p.processor.ProcessMessage(msgEvent)
		if err != nil {
			p.logger.Error(err)
		}
	}
	return nil
}

func (p *poller) Stop() {
	p.logger.Debug("Stopping scheduler...")
	p.scheduler.Stop()
}
