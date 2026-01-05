// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package partnerplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/chain4travel/camino-messenger-bot/v12/tests/e2e/resources"

	"go.uber.org/zap"
)

func NewFactory(
	logger *zap.SugaredLogger,
	resourceManagerSession *resources.Session,
	e2eTmpDir string,
	binPath string,
) *Factory {
	return &Factory{
		logger:                 logger,
		resourceManagerSession: resourceManagerSession,
		binPath:                binPath,
		dir:                    path.Join(e2eTmpDir, "pp-mock"),
	}
}

type Factory struct {
	logger                 *zap.SugaredLogger
	resourceManagerSession *resources.Session
	dir                    string
	binPath                string
	mutex                  sync.Mutex
	partnerPlugins         []*PartnerPlugin
}

func (f *Factory) CreatePartnerPlugin() (*PartnerPlugin, error) {
	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create pp-mock directory: %w", err)
	}

	port, err := f.resourceManagerSession.GetNetworkPort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free port: %w", err)
	}

	pp := newPartnerPlugin(
		f.logger,
		f.binPath,
		port,
		path.Join(f.dir, fmt.Sprintf("partner-plugin-%d.log", port)),
	)

	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.partnerPlugins = append(f.partnerPlugins, pp)

	return pp, nil
}

func (f *Factory) StopPartnerPlugins(ctx context.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var errs []error
	errsMx := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, pp := range f.partnerPlugins {
		wg.Add(1)
		go func(pp *PartnerPlugin) {
			defer wg.Done()
			if err := pp.Stop(ctx); err != nil {
				errsMx.Lock()
				errs = append(errs, fmt.Errorf("failed to stop partner plugin (%d): %w", pp.pid, err))
				errsMx.Unlock()
			}
		}(pp)
	}

	wg.Wait()
	return errors.Join(errs...)
}
