/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pools

import (
	"sync"

	kuberecoveryv1alpha1 "freepik.com/kuberecovery/api/v1alpha1"
)

// ResourceWatcher
type ResourceWatcher struct {
	RecoveryConfig *kuberecoveryv1alpha1.RecoveryConfig
	Resource       string
	APIVersion     string
	Namespace      string
	Chan           chan struct{}
}

var (
	ResourceWatcherPoolKeyFormat = "%s/%s/%s/%s"
)

// ResourceWatcherStore
type ResourceWatcherStore struct {
	mu    sync.RWMutex
	Store map[string]*ResourceWatcher
}

func (c *ResourceWatcherStore) Set(key string, watcher *ResourceWatcher) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Store[key] = watcher
}

func (c *ResourceWatcherStore) Get(key string) (*ResourceWatcher, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	watcher, exists := c.Store[key]
	return watcher, exists
}

func (c *ResourceWatcherStore) GetAll() map[string]*ResourceWatcher {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Store
}

func (c *ResourceWatcherStore) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Store, key)
}
