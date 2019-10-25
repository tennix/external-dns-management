/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package owners

import (
	"github.com/gardener/controller-manager-library/pkg/utils"
	"sync"
)

type Accepted func(id string) bool

type FilteredLayer struct {
	lock sync.RWMutex
	up   *OwnerLink
	down *OwnerLayerAccess

	cache    utils.StringSet
	accepted Accepted
}

func NewFilteredLayer(accept Accepted, up *OwnerLink, down *OwnerLayerAccess) OwnerLayer {
	return &FilteredLayer{
		up:       up,
		down:     down,
		accepted: accept,
		cache:    utils.StringSet{},
	}
}

func FilteredLayerCreator(accept Accepted) OwnerLayerCreator {
	return func(up *OwnerLink, down *OwnerLayerAccess) OwnerLayer {
		return NewFilteredLayer(accept, up, down)
	}
}

func (this *FilteredLayer) Start(down utils.StringSet) OwnerHandler {
	debug("start lock\n")

	this.lock.Lock()
	defer this.lock.Unlock()
	debug("start locked\n")

	this.cache = utils.StringSet{}
	deleted := utils.StringSet{}
	for id := range down {
		if this.accepted(id) {
			this.cache.Add(id)
		} else {
			deleted.Add(id)
		}
	}
	if len(deleted) > 0 {
		this.up.OwnerSetChanged(utils.StringSet{}, deleted)
	}
	return this
}

func (this *FilteredLayer) OwnerSetChanged(added, deleted utils.StringSet) {
	del := utils.StringSet{}
	add := utils.StringSet{}
	for id := range added {
		if this.accepted(id) {
			this.cache.Add(id)
			add.Add(id)
		}
	}
	for id := range deleted {
		if this.cache.Contains(id) {
			this.cache.Remove(id)
			del.Add(id)
		}
	}
	if len(del) > 0 || len(add) > 0 {
		this.up.OwnerSetChanged(add, del)
	}
}

func (this *FilteredLayer) GetIds() utils.StringSet {
	debug("ids lock\n")

	this.lock.RLock()
	defer this.lock.RUnlock()
	debug("ids locked\n")

	return this.cache.Copy()
}

func (this *FilteredLayer) IsResponsibleFor(id string) bool {
	debug("filter respo lock\n")

	this.lock.RLock()
	defer this.lock.RUnlock()
	debug("filter respo locked\n")
	return this.cache.Contains(id)
}

func (this *FilteredLayer) FilterChanged() {
	debug("changed lock\n")

	this.up.Lock(&this.lock)
	defer this.up.Unlock(&this.lock)
	debug("changed locked\n")

	cur := this.cache
	this.cache = utils.StringSet{}
	deleted := utils.StringSet{}
	added := utils.StringSet{}
	for id := range this.down.GetIds() {
		if this.accepted(id) {
			this.cache.Add(id)
			if !cur.Contains(id) {
				added.Add(id)
			}
		} else {
			if cur.Contains(id) {
				deleted.Add(id)
			}
		}
	}
	if len(deleted) > 0 || len(added) > 0 {
		this.up.OwnerSetChanged(added, deleted)
	}
}
