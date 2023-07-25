// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"encoding/json"
	"sync"

	cfgcpi "github.com/open-component-model/ocm/v2/pkg/contexts/config/cpi"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/plugin"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/plugin/cache"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/plugin/config"
	"github.com/open-component-model/ocm/v2/pkg/contexts/ocm/plugin/descriptor"
	"github.com/open-component-model/ocm/v2/pkg/utils"
)

type Set = *pluginsImpl

type pluginSettings struct {
	config                  json.RawMessage
	disableAutoRegistration bool
}
type pluginsImpl struct {
	lock sync.RWMutex

	updater cfgcpi.Updater
	ctx     cpi.Context
	base    cache.PluginDir
	configs map[string]*pluginSettings
	plugins map[string]plugin.Plugin
}

var _ config.Target = (*pluginsImpl)(nil)

func New(ctx cpi.Context, path string) Set {
	pi := &pluginsImpl{
		ctx:     ctx,
		configs: map[string]*pluginSettings{},
		plugins: map[string]plugin.Plugin{},
	}
	pi.updater = cfgcpi.NewUpdater(ctx.ConfigContext(), pi)
	pi.Update()
	pi.base = cache.Get(path)
	for _, n := range pi.base.PluginNames() {
		cfg := pi.configs[n]
		if cfg == nil {
			pi.plugins[n] = plugin.NewPlugin(ctx, pi.base.Get(n), nil)
		} else {
			p := plugin.NewPlugin(ctx, pi.base.Get(n), cfg.config)
			p.DisableAutoConfiguration(cfg.disableAutoRegistration)
			pi.plugins[n] = p
		}
	}
	return pi
}

func (pi *pluginsImpl) GetContext() cpi.Context {
	return pi.ctx
}

func (pi *pluginsImpl) Update() {
	err := pi.updater.Update()
	if err != nil {
		pi.ctx.Logger(descriptor.REALM).Error("config update failed", "error", err.Error())
	}
}

func (pi *pluginsImpl) getSettings(name string) *pluginSettings {
	cfg := pi.configs[name]
	if cfg == nil {
		cfg = &pluginSettings{}
		pi.configs[name] = cfg
	}
	return cfg
}

func (pi *pluginsImpl) DisableAutoConfiguration(name string, flag bool) {
	pi.lock.Lock()
	defer pi.lock.Unlock()

	pi.getSettings(name).disableAutoRegistration = flag
	if pi.plugins[name] != nil {
		pi.plugins[name].DisableAutoConfiguration(flag)
	}
}

func (pi *pluginsImpl) ConfigurePlugin(name string, config json.RawMessage) {
	pi.lock.Lock()
	defer pi.lock.Unlock()

	pi.getSettings(name).config = config
	if pi.plugins[name] != nil {
		pi.plugins[name].SetConfig(config)
	}
}

func (pi *pluginsImpl) PluginNames() []string {
	pi.lock.RLock()
	defer pi.lock.RUnlock()

	return utils.StringMapKeys(pi.plugins)
}

func (pi *pluginsImpl) Get(name string) plugin.Plugin {
	pi.Update()

	pi.lock.RLock()
	defer pi.lock.RUnlock()

	p, ok := pi.plugins[name]
	if ok {
		return p
	}
	return nil
}
