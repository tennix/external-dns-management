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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type TestHandler struct {
	added utils.StringSet
	deleted utils.StringSet
}

func (this *TestHandler) OwnerSetChanged(added, deleted utils.StringSet) {
	this.added = added
	this.deleted =deleted
}

func (this *TestHandler) Check(added, deleted utils.StringSet) {
	Expect(this.added).To(Equal(added))
	Expect(this.deleted).To(Equal(deleted))
}

var _ = Describe("Owner cache", func() {
	client1 := "client1"
	client2 := "client2"

	It("initializes  correctly", func() {
		cache := NewOwners()
		Expect(cache.GetIds()).To(Equal(utils.NewStringSet()))
	})

	It("adds client", func() {
		cache := NewOwners()

		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		Expect(cache.GetIds()).To(Equal(utils.NewStringSet("id1", "id2")))
	})

	It("updates client", func() {
		cache := NewOwners()

		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		cache.UpdateClient(client1, utils.NewStringSet("id3", "id2"))

		Expect(cache.GetIds()).To(Equal(utils.NewStringSet("id2", "id3")))
	})

	It("handles client delete", func() {
		cache := NewOwners()

		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		cache.DeleteClient(client1)

		Expect(cache.GetIds()).To(Equal(utils.NewStringSet()))
	})

	It("handles two clients", func() {
		cache := NewOwners()

		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))

		Expect(cache.GetIds()).To(Equal(utils.NewStringSet("id1", "id2", "id3")))
	})

	It("handles two clients with one deleted", func() {
		cache := NewOwners()

		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))
		cache.DeleteClient(client1)

		Expect(cache.GetIds()).To(Equal(utils.NewStringSet("id2", "id3")))
	})
	It("handles two clients with both deleted", func() {
		cache := NewOwners()

		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))
		cache.DeleteClient(client1)
		cache.DeleteClient(client2)

		Expect(cache.GetIds()).To(Equal(utils.NewStringSet()))
	})

	It("emits event for added client ", func() {
		cache := NewOwners()

		handler := &TestHandler{}
		cache.RegisterHandler(handler)
		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))

		handler.Check(utils.NewStringSet("id1", "id2"), utils.StringSet{})
	})

	It("emits event for updated client ", func() {
		cache := NewOwners()

		handler := &TestHandler{}
		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))

		cache.RegisterHandler(handler)
		cache.UpdateClient(client1, utils.NewStringSet("id3", "id2"))

		handler.Check(utils.NewStringSet("id3"), utils.NewStringSet("id1"))
	})

	It("emits event for deleted client ", func() {
		cache := NewOwners()

		handler := &TestHandler{}
		cache.UpdateClient(client1, utils.NewStringSet("id1", "id2"))
		cache.UpdateClient(client2, utils.NewStringSet("id3", "id2"))

		cache.RegisterHandler(handler)
		cache.DeleteClient(client1)

		handler.Check(utils.NewStringSet(), utils.NewStringSet("id1"))
	})

})
