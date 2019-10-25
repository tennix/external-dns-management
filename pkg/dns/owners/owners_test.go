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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"reflect"
)

type TestHandler struct {
	added   utils.StringSet
	deleted utils.StringSet
}

func (this *TestHandler) OwnerSetChanged(added, deleted utils.StringSet) {
	this.added = added
	this.deleted = deleted
}

func (this *TestHandler) Check(added, deleted utils.StringSet) {
	if len(this.added) > 0 || len(added) > 0 {
		Expect(this.added).To(Equal(added))
	}
	if len(this.added) > 0 || len(added) > 0 {
		Expect(this.deleted).To(Equal(deleted))
	}
}

var (
	client1 = "client1"
	client2 = "client2"
)

func basetest(setup func() (*OwnerStack, *Owners), filtered utils.StringSet) {
	Context("base tests", func() {
		It("initializes correctly", func() {
			stack, _ := setup()
			Expect(stack.GetIds()).To(Equal(utils.NewStringSet()))
		})

		It("adds client", func() {
			stack, cache := setup()

			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			Expect(stack.GetIds()).To(Equal(utils.NewStringSet("id1", "id2").Intersect(filtered)))
		})

		It("updates client", func() {
			stack, cache := setup()

			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			cache.UpdateClient(client1, utils.NewStringSet("id3", "id2"))

			Expect(stack.GetIds()).To(Equal(utils.NewStringSet("id2", "id3").Intersect(filtered)))
		})

		It("handles client delete", func() {
			stack, cache := setup()

			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			cache.DeleteClient(client1)

			Expect(stack.GetIds()).To(Equal(utils.NewStringSet()))
		})

		It("handles two clients", func() {
			stack, cache := setup()

			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))

			Expect(stack.GetIds()).To(Equal(utils.NewStringSet("id1", "id2", "id3").Intersect(filtered)))
		})

		It("handles two clients with one deleted", func() {
			stack, cache := setup()

			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))
			cache.DeleteClient(client1)

			Expect(stack.GetIds()).To(Equal(utils.NewStringSet("id2", "id3").Intersect(filtered)))
		})
		It("handles two clients with both deleted", func() {
			stack, cache := setup()

			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))
			cache.DeleteClient(client1)
			cache.DeleteClient(client2)

			Expect(stack.GetIds()).To(Equal(utils.NewStringSet()))
		})

		It("emits event for added client ", func() {
			stack, cache := setup()

			handler := &TestHandler{}
			stack.RegisterHandler(handler)
			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))

			handler.Check(utils.NewStringSet("id1", "id2").Intersect(filtered), utils.StringSet{})
		})

		It("emits event for updated client ", func() {
			stack, cache := setup()

			handler := &TestHandler{}
			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))

			stack.RegisterHandler(handler)
			cache.UpdateClient(client1, utils.NewStringSet("id3", "id2"))

			handler.Check(utils.NewStringSet("id3").Intersect(filtered), utils.NewStringSet("id1").Intersect(filtered))
		})

		It("emits event for deleted client ", func() {
			stack, cache := setup()

			handler := &TestHandler{}
			cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
			cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))

			stack.RegisterHandler(handler)
			cache.DeleteClient(client1)

			handler.Check(utils.NewStringSet(), utils.NewStringSet("id1").Intersect(filtered))
		})
	})
}

var _ = Describe("Owners", func() {

	all := utils.NewStringSet("id1", "id2", "id3", "id4")

	Context("owner base layer", func() {
		basetest(NewOwners, all)
	})

	Context("filter layer", func() {

		It("inserts a layer", func() {
			stack, owners := NewOwners()
			stack.InserLayer(stack.Access(), DummyLayerCreator)

			Expect(stack.Access().NextLayer()).To(Not(BeNil()))
			Expect(stack.Access().NextLayer().Layer()).To(Equal(owners))
			Expect(reflect.TypeOf(stack.Access().Layer())).To(Equal(reflect.TypeOf(&DummyLayer{})))
		})

		basetest(func() (*OwnerStack, *Owners) {
			stack, owners := NewOwners()
			stack.InserLayer(stack.Access(), DummyLayerCreator)
			return stack, owners
		}, all)
	})

	Context("two filter layers", func() {

		It("inserts a layer", func() {
			stack, owners := NewOwners()
			stack.InserLayer(stack.Access(), DummyLayerCreator)
			stack.InserLayer(stack.Access(), DummyLayerCreator)

			Expect(stack.Access().NextLayer()).To(Not(BeNil()))
			Expect(reflect.TypeOf(stack.Access().Layer())).To(Equal(reflect.TypeOf(&DummyLayer{})))
			Expect(stack.Access().NextLayer().NextLayer()).To(Not(BeNil()))
			Expect(reflect.TypeOf(stack.Access().NextLayer().Layer())).To(Equal(reflect.TypeOf(&DummyLayer{})))
			Expect(stack.Access().NextLayer().NextLayer().Layer()).To(Equal(owners))
		})

		basetest(func() (*OwnerStack, *Owners) {
			stack, owners := NewOwners()
			stack.InserLayer(stack.Access(), DummyLayerCreator)
			stack.InserLayer(stack.Access(), DummyLayerCreator)
			return stack, owners
		}, all)
	})

	Context("filtering layer 2,3", func() {
		filter := utils.NewStringSet("id2", "id3")
		basetest(func() (*OwnerStack, *Owners) {
			stack, owners := NewOwners()
			stack.InserLayer(stack.Access(), FilteredLayerCreator(filter.Contains))
			return stack, owners
		}, filter)
	})

	Context("filtering layer 2,1", func() {
		filter := utils.NewStringSet("id2", "id1")
		basetest(func() (*OwnerStack, *Owners) {
			stack, owners := NewOwners()
			stack.InserLayer(stack.Access(), FilteredLayerCreator(filter.Contains))
			return stack, owners
		}, filter)
	})
})

type DummyLayer struct {
	up   *OwnerLink
	down *OwnerLayerAccess
}

func (this *DummyLayer) GetIds() utils.StringSet {
	return this.down.GetIds()
}
func (this *DummyLayer) IsResponsibleFor(id string) bool {
	return this.down.IsResponsibleFor(id)
}
func (this *DummyLayer) Start(utils.StringSet) OwnerHandler {
	return this.up
}

func DummyLayerCreator(up *OwnerLink, down *OwnerLayerAccess) OwnerLayer {
	return &DummyLayer{up, down}
}
