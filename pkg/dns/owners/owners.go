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

func debug(msg string) {
	//fmt.Println(msg)
}

type OwnerHandler interface {
	OwnerSetChanged(added utils.StringSet, deleted utils.StringSet)
}

type OwnerLayerCreator func(*OwnerLink, *OwnerLayerAccess) OwnerLayer

type OwnerStack struct {
	lock sync.RWMutex
	*OwnerLayerAccess
}

func NewOwners() (*OwnerStack, *Owners) {
	stack := NewOwnerStack(CreateOwnerBaseLayer)
	return stack, stack.link.layer.(*Owners)
}

func NewDefaultOwnerStack(ids ...string) *OwnerStack {
	stack := NewOwnerStack(CreateOwnerBaseLayer)
	ApplyLayers(stack)
	return stack
}

func NewOwnerStack(c OwnerLayerCreator) *OwnerStack {
	stack := &OwnerStack{}
	access := &OwnerLayerAccess{link: &OwnerLink{stack: stack}, stack: stack}
	stack.OwnerLayerAccess = access
	access.link.up = access
	access.link.layer = c(stack.OwnerLayerAccess.link, nil)
	return stack
}

func (this *OwnerStack) Access() *OwnerLayerAccess {
	return this.OwnerLayerAccess
}

func (this *OwnerStack) BaseLayer() OwnerLayer {
	debug("base query lock\n")

	this.lock.Lock()
	defer this.lock.Unlock()
	debug("base query locked\n")

	cur := this.OwnerLayerAccess
	for cur.link.down != nil {
		cur = cur.link.down
	}
	return cur.link.layer
}

func (this *OwnerStack) InserLayer(acc *OwnerLayerAccess, c OwnerLayerCreator) *OwnerLayerAccess {
	debug("insert lock\n")

	this.lock.Lock()
	defer this.lock.Unlock()
	debug("insert locked\n")

	cur := this.OwnerLayerAccess
	for cur != nil {
		if cur == acc {
			break
		}
		cur = cur.link.down
	}
	if cur == nil {
		return nil
	}
	pos := acc.link
	set := utils.StringSet{}
	if pos.down != nil {
		set = pos.down.GetIds()
	}
	access := &OwnerLayerAccess{link: pos, stack: this}
	link := &OwnerLink{up: pos.up, down: access, stack: this}
	pos.up.link = link
	pos.up = access
	layer := c(link, access)
	link.layer = layer
	h := layer.Start(set)
	if h != nil {
		access.registerHandler(h)
	}
	return acc
}

////////////////////////////////////////////////////////////////////////////////
//  An element in a layer stack for owner determination
// It implements this layer and use an OwnerLink to propagate changes
// to a layer access used by upper layers or users of the owner layer stack

type OwnerLayer interface {
	GetIds() utils.StringSet
	IsResponsibleFor(id string) bool
	Start(down utils.StringSet) OwnerHandler
}

////////////////////////////////////////////////////////////////////////////////
//  An api endpoint for accessing an owner layer
// It implements this layer and use an OwnerLink to propagate changes
// to a layer access used by upper layers or users of the owner layer stack

type OwnerLayerAccess struct {
	handlers []OwnerHandler
	link     *OwnerLink
	stack    *OwnerStack
}

func (this *OwnerLayerAccess) NextLayer() *OwnerLayerAccess {
	debug("next lock\n")

	this.stack.lock.Lock()
	defer this.stack.lock.Unlock()
	debug("next locked\n")
	return this.link.down
}

func (this *OwnerLayerAccess) Layer() OwnerLayer {
	debug("layer lock\n")

	this.stack.lock.Lock()
	defer this.stack.lock.Unlock()
	debug("layer locked\n")
	return this.link.layer
}

func (this *OwnerLayerAccess) GetIds() utils.StringSet {
	return this.link.layer.GetIds()
}

func (this *OwnerLayerAccess) IsResponsibleFor(id string) bool {
	return this.link.layer.IsResponsibleFor(id)
}

func (this *OwnerLayerAccess) RegisterHandler(h OwnerHandler) {
	debug("register lock\n")

	this.stack.lock.Lock()
	defer this.stack.lock.Unlock()
	debug("register locked\n")
	this.registerHandler(h)
}

func (this *OwnerLayerAccess) registerHandler(h OwnerHandler) {
	for _, o := range this.handlers {
		if o == h {
			return
		}
	}
	this.handlers = append(this.handlers, h)
}

func (this *OwnerLayerAccess) UnRegisterHandler(h OwnerHandler) {
	this.stack.lock.Lock()
	defer this.stack.lock.Unlock()

	for i, o := range this.handlers {
		if o == h {
			this.handlers = append(this.handlers[:i], this.handlers[i+1:]...)
			return
		}
	}
}

func (this *OwnerLayerAccess) notify(added utils.StringSet, deleted utils.StringSet) {
	for _, h := range this.handlers {
		h.OwnerSetChanged(added, deleted)
	}
}

////////////////////////////////////////////////////////////////////////////////
//  The layer link elements that binds two layers together
// It manages the connection between the layer access endpoint and actual
// laver implementation

type OwnerLink struct {
	up    *OwnerLayerAccess
	layer OwnerLayer
	down  *OwnerLayerAccess
	stack *OwnerStack
}

func (this *OwnerLink) OwnerSetChanged(added, deleted utils.StringSet) {
	this.up.notify(added, deleted)
}

func (this *OwnerLink) Lock(sub sync.Locker) {
	debug("com lock\n")
	this.stack.lock.Lock()
	if sub != nil {
		sub.Lock()
	}
	debug("com locked\n")
}

func (this *OwnerLink) Unlock(sub sync.Locker) {
	if sub != nil {
		sub.Unlock()
	}
	this.stack.lock.Unlock()
}

////////////////////////////////////////////////////////////////////////////////
// Owner Base layer

func CreateOwnerBaseLayer(up *OwnerLink, down *OwnerLayerAccess) OwnerLayer {
	return &Owners{
		owners:  map[string]int64{},
		clients: map[string]utils.StringSet{},
		up:      up,
	}
}

type Owners struct {
	lock    sync.RWMutex
	owners  map[string]int64
	clients map[string]utils.StringSet
	up      *OwnerLink
}

func (this *Owners) Start(down utils.StringSet) OwnerHandler {
	return nil
}

func (this *Owners) GetIds() utils.StringSet {
	debug("id lock\n")
	this.lock.RLock()
	defer this.lock.RUnlock()
	debug("id locked\n")

	set := utils.StringSet{}
	for i := range this.owners {
		set.Add(i)
	}
	return set
}

func (this *Owners) IsResponsibleFor(id string) bool {
	debug("respo lock\n")

	this.lock.RLock()
	defer this.lock.RUnlock()
	debug("respo locked\n")

	_, ok := this.owners[id]
	return ok
}

func (this *Owners) GetIdsFor(client string) utils.StringSet {
	debug("ids for lock\n")

	this.lock.RLock()
	defer this.lock.RUnlock()
	debug("ids for locked\n")

	set := utils.StringSet{}
	set.AddSet(this.clients[client])
	return set
}

func (this *Owners) notify(added utils.StringSet, deleted utils.StringSet) {
	this.up.OwnerSetChanged(added, deleted)
}

func (this *Owners) UpdateClient(client string, owners utils.StringSet) {
	debug("update lock\n")

	this.up.Lock(&this.lock)
	defer this.up.Unlock(&this.lock)
	debug("update locked\n")

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
	this.clients[client] = owners
	this.notify(added, deleted)
}

func (this *Owners) DeleteClient(client string) {
	debug("delete lock\n")

	this.lock.Lock()
	defer this.lock.Unlock()
	debug("delete locked\n")

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

	delete(this.clients, client)
	this.notify(added, deleted)
}
