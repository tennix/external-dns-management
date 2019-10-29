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
	"fmt"
	"github.com/gardener/external-dns-management/pkg/crds"
	"github.com/gardener/external-dns-management/pkg/dns"
	"github.com/gardener/external-dns-management/pkg/dns/owners"
	. "github.com/gardener/external-dns-management/pkg/dns/provider/defs"
	"github.com/gardener/external-dns-management/pkg/dns/source"

	"github.com/gardener/controller-manager-library/pkg/controllermanager/controller"
	"github.com/gardener/controller-manager-library/pkg/controllermanager/controller/reconcile"
	"github.com/gardener/controller-manager-library/pkg/resources"
	api "github.com/gardener/external-dns-management/pkg/apis/dns/v1alpha1"
)

var ownerGroupKind = resources.NewGroupKind(api.GroupName, api.DNSOwnerKind)

const CONTROLLER_OWNER = "dns-owners"

func init() {
	controller.Configure(CONTROLLER_OWNER).
		DefaultedStringOption(OPT_CLASS, dns.DEFAULT_CLASS, "Identifier used to differentiate responsible controllers for entries").
		DefaultedStringOption(OPT_IDENTIFIER, "dnscontroller", "Identifier used to mark DNS entries").
		DefaultedIntOption(OPT_SETUP, 1, "number of processors for controller setup").
		Reconciler(create).
		Cluster(PROVIDER_CLUSTER).
		CustomResourceDefinitions(crds.DNSOwnerCRD).
		MainResource(api.GroupName, api.DNSOwnerKind).
		DefaultWorkerPool(2, 0).MustRegister(CONTROLLER_GROUP_DNS_CONTROLLERS)
}

var _ reconcile.Interface = &reconciler{}

///////////////////////////////////////////////////////////////////////////////

func create(c controller.Interface) (reconcile.Interface, error) {
	classes := controller.NewClassesByOption(c, OPT_CLASS, source.CLASS_ANNOTATION, dns.DEFAULT_CLASS)

	ident, err := c.GetStringOption(OPT_IDENTIFIER)
	if err != nil {
		return nil, fmt.Errorf("identifier not configured")
	}
	stack := c.GetEnvironment().GetOrCreateSharedValue(KEY_OWNERS,
		func() interface{} {
			return owners.NewDefaultOwnerStack()
		}).(*owners.OwnerStack)
	cache := owners.NewOwnerCache(ident)
	base := stack.BaseLayer().(*owners.Owners)
	return &reconciler{
		controller: c,
		classes:    classes,
		cache:      cache,
		ident:      ident,
		owners:     base,
	}, nil
}
