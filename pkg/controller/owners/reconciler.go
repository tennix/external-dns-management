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
	"github.com/gardener/controller-manager-library/pkg/controllermanager/controller"
	"github.com/gardener/controller-manager-library/pkg/controllermanager/controller/reconcile"
	"github.com/gardener/controller-manager-library/pkg/logger"
	"github.com/gardener/controller-manager-library/pkg/resources"

	api "github.com/gardener/external-dns-management/pkg/apis/dns/v1alpha1"
	"github.com/gardener/external-dns-management/pkg/dns/owners"
	. "github.com/gardener/external-dns-management/pkg/dns/provider/defs"
	dnsutils "github.com/gardener/external-dns-management/pkg/dns/utils"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

const CLIENT_ID = "dns-owner-resources"

type reconciler struct {
	reconcile.DefaultReconciler
	controller controller.Interface
	ident      string
	owners     *owners.Owners
	classes    *controller.Classes
	cache      *owners.OwnerCache
}

func (this *reconciler) setupFor(obj runtime.Object, msg string, exec func(resources.Object), processors int) {
	this.controller.Infof("### setup %s", msg)
	res, _ := this.controller.GetMainCluster().Resources().GetByExample(obj)
	list, _ := res.ListCached(labels.Everything())
	dnsutils.ProcessElements(list, func(e resources.Object) {
		if this.IsResponsibleFor(this.controller, e) {
			exec(e)
		}
	}, processors)
	this.controller.Infof("standard identifier: %s", this.ident)
	this.controller.Infof("initial ownerset: %s", this.owners.GetIds())
}

func (this *reconciler) IsResponsibleFor(logger logger.LogContext, obj resources.Object) bool {
	return this.classes.IsResponsibleFor(logger, obj)
}

func (this *reconciler) Setup() {
	this.controller.Infof("*** state Setup ")

	processors, err := this.controller.GetIntOption(OPT_SETUP)
	if err != nil || processors <= 0 {
		processors = 1
	}
	this.controller.Infof("using %d parallel workers for initialization", processors)
	this.setupFor(&api.DNSOwner{}, "owners", func(e resources.Object) {
		p := dnsutils.DNSOwner(e)
		this.UpdateOwner(this.controller.NewContext("owner", p.ObjectName().String()), p)
	}, processors)
}

func (this *reconciler) Reconcile(logger logger.LogContext, obj resources.Object) reconcile.Status {
	switch {
	case obj.IsA(&api.DNSOwner{}):
		if this.IsResponsibleFor(logger, obj) {
			return this.UpdateOwner(logger, dnsutils.DNSOwner(obj))
		} else {
			return this.OwnerDeleted(logger, obj.Key())
		}
	}
	return reconcile.Succeeded(logger)
}

func (this *reconciler) Deleted(logger logger.LogContext, key resources.ClusterObjectKey) reconcile.Status {
	logger.Debugf("deleted %s", key)
	switch key.GroupKind() {
	case ownerGroupKind:
		return this.OwnerDeleted(logger, key.ObjectKey())
	}
	return reconcile.Succeeded(logger)
}

////////////////////////////////////////////////////////////////////////////////
// OwnerIds
////////////////////////////////////////////////////////////////////////////////

func (this *reconciler) UpdateOwner(logger logger.LogContext, owner *dnsutils.DNSOwnerObject) reconcile.Status {
	changed, active := this.cache.UpdateOwner(owner)
	logger.Infof("update: changed owner ids %s, active owner ids %s", changed, active)
	if len(changed) > 0 {
		this.updateOwners()
	}
	return reconcile.Succeeded(logger)
}

func (this *reconciler) OwnerDeleted(logger logger.LogContext, key resources.ObjectKey) reconcile.Status {
	changed, active := this.cache.DeleteOwner(key)
	logger.Infof("delete: changed owner ids %s, active owner ids %s", changed, active)
	if len(changed) > 0 {
		this.updateOwners()
	}
	return reconcile.Succeeded(logger)
}

func (this *reconciler) updateOwners() {
	this.owners.UpdateClient(CLIENT_ID, this.cache.GetIds())
}
