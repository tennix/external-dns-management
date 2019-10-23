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

package extension

import (
	"github.com/gardener/controller-manager-library/pkg/utils"
	"sync"
)

type OwnerHandler interface {
	OwnerSetChanged(added utils.StringSet, deleted utils.StringSet)
}

type Owners struct {
	lock     sync.RWMutex
	owners   map[string]int64
	clients  map[string]utils.StringSet
	handlers []OwnerHandler
}

func NewOwners() *Owners {
	return &Owners{
		owners:  map[string]int64{},
		clients: map[string]utils.StringSet{},
	}
}

func (this *Owners) GetIds() utils.StringSet {
	this.lock.RLock()
	defer this.lock.RUnlock()

	set:= utils.StringSet{}
	for i := range this.owners {
		set.Add(i)
	}
	return set
}

func (this *Owners) GetIdsFor(client string) utils.StringSet {
	this.lock.RLock()
	defer this.lock.RUnlock()

	set:= utils.StringSet{}
	set.AddSet(this.clients[client])
	return set
}

func (this *Owners) RegisterHandler(h OwnerHandler) {
	this.lock.Lock()
	defer this.lock.Unlock()

	for _, o := range this.handlers {
		if o == h {
			return
		}
	}
	this.handlers = append(this.handlers, h)
}

func (this *Owners) UnRegisterHandler(h OwnerHandler) {
	this.lock.Lock()
	defer this.lock.Unlock()

	for i, o := range this.handlers {
		if o == h {
			this.handlers = append(this.handlers[:i], this.handlers[i+1:]...)
			return
		}
	}
}

func (this *Owners) notify(added utils.StringSet, deleted utils.StringSet) {
	for _, h := range this.handlers {
		h.OwnerSetChanged(added, deleted)
	}
}

func (this *Owners) UpdateClient(client string, owners utils.StringSet) {
	this.lock.Lock()
	defer this.lock.Unlock()

	old := this.clients[client]

	added := utils.StringSet{}
	deleted := utils.StringSet{}
	for o := range owners {
		if !old.Contains(o) {
			cnt := this.owners[o] + 1
			this.owners[o] = cnt
			if cnt == 1 {
				added.Add(o)
			}
		}
	}
	for o := range old {
		if !owners.Contains(o) {
			cnt := this.owners[o] - 1
			if cnt == 0 {
				delete(this.owners, o)
				deleted.Add(o)
			} else {
				this.owners[o] = cnt
			}
		}
	}
	this.clients[client]=owners
	this.notify(added, deleted)
}

func (this *Owners) DeleteClient(client string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	old := this.clients[client]

	added := utils.StringSet{}
	deleted := utils.StringSet{}

	for o := range old {
		cnt := this.owners[o] - 1
		this.owners[o] = cnt
		if cnt == 0 {
			delete(this.owners, o)
			deleted.Add(o)
		}
	}

	delete(this.clients,client)
	this.notify(added, deleted)
}
