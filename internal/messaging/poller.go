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
	matrixClient matrix.Client
	scheduler    *gocron.Scheduler
	logger       *zap.SugaredLogger
	processor    Processor
}

func NewPoller(matrixClient matrix.Client, logger *zap.SugaredLogger, processor Processor) Poller {
	return &poller{
		matrixClient: matrixClient,
		logger:       logger,
		scheduler:    gocron.NewScheduler(time.UTC),
		processor:    processor,
	}
}

func (p *poller) Start() error {
	p.logger.Info("Scheduling matrix syncing...")
	_, err := p.scheduler.Every(matrix.DefaultInterval).Do(p.matrixClient.Sync, p.processor.RoomChannel(), p.processor.MsgChannel())
	if err != nil {
		p.logger.Error(err)
		return err
	}

	p.logger.Info("Starting scheduler...")
	p.scheduler.StartAsync()
	return nil
}

func (p *poller) Stop() {
	p.logger.Info("Stopping scheduler...")
	p.scheduler.Stop()
}
