package evmtrader

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type ConfigLoader struct {
	configPath     string
	currentConfig  *AutoTradingConfig
	mu             sync.RWMutex
	lastModTime    time.Time
	reloadInterval time.Duration
}

func NewConfigLoader(configPath string, reloadInterval time.Duration) (*ConfigLoader, error) {
	if reloadInterval == 0 {
		reloadInterval = 5 * time.Second
	}

	loader := &ConfigLoader{
		configPath:     configPath,
		reloadInterval: reloadInterval,
	}

	if err := loader.Reload(); err != nil {
		return nil, fmt.Errorf("initial config load: %w", err)
	}

	return loader, nil
}

func (cl *ConfigLoader) Reload() error {
	jsonCfg, err := LoadAutoTradingConfig(cl.configPath)
	if err != nil {
		return fmt.Errorf("load config file: %w", err)
	}

	newConfig := jsonCfg.ToAutoTradingConfig()

	cl.mu.Lock()
	defer cl.mu.Unlock()

	fileInfo, err := os.Stat(cl.configPath)
	if err == nil {
		cl.lastModTime = fileInfo.ModTime()
	}

	cl.currentConfig = &newConfig
	return nil
}

func (cl *ConfigLoader) GetConfig() AutoTradingConfig {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if cl.currentConfig == nil {
		return AutoTradingConfig{}
	}

	return *cl.currentConfig
}

func (cl *ConfigLoader) StartWatcher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cl.reloadInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:

				fileInfo, err := os.Stat(cl.configPath)
				if err != nil {
					continue
				}

				cl.mu.RLock()
				needsReload := fileInfo.ModTime().After(cl.lastModTime)
				cl.mu.RUnlock()

				if needsReload {
					if err := cl.Reload(); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to reload config: %v\n", err)
					} else {
						fmt.Printf("Config reloaded from: %s\n", cl.configPath)
					}
				}
			}
		}
	}()
}
