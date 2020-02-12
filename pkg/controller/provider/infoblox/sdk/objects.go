/*
 * Copyright 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

package sdk

import (
	ibclient "github.com/infobloxopen/infoblox-go-client"
)

type IBBase struct {
	objectType   string
	returnFields []string
	eaSearch     ibclient.EASearch
}

func NewIBBase(objectType string, returnFields []string, eaSearch ibclient.EASearch) IBBase {
	return IBBase{
		objectType:   objectType,
		returnFields: returnFields,
		eaSearch:     eaSearch,
	}
}

func (obj *IBBase) ObjectType() string {
	return obj.objectType
}

func (obj *IBBase) ReturnFields() []string {
	return obj.returnFields
}

func (obj *IBBase) EaSearch() ibclient.EASearch {
	return obj.eaSearch
}

///////////////////////////////////////////////////////////////////////////////

type RecordA struct {
	IBBase   `json:"-"`
	Ref      string      `json:"_ref,omitempty"`
	Ipv4Addr string      `json:"ipv4addr,omitempty"`
	Name     string      `json:"name,omitempty"`
	View     string      `json:"view,omitempty"`
	Zone     string      `json:"zone,omitempty"`
	Ttl      uint        `json:"ttl,omitempty"`
	Ea       ibclient.EA `json:"extattrs,omitempty"`
}

func NewRecordA(ra RecordA) *RecordA {
	res := ra
	res.objectType = "record:a"
	res.returnFields = []string{"extattrs", "ipv4addr", "name", "view", "zone", "ttl"}
	return &res
}

///////////////////////////////////////////////////////////////////////////////

type RecordCNAME struct {
	IBBase    `json:"-"`
	Ref       string      `json:"_ref,omitempty"`
	Canonical string      `json:"canonical,omitempty"`
	Name      string      `json:"name,omitempty"`
	View      string      `json:"view,omitempty"`
	Zone      string      `json:"zone,omitempty"`
	Ttl       uint        `json:"ttl,omitempty"`
	Ea        ibclient.EA `json:"extattrs,omitempty"`
}

func NewRecordCNAME(rc RecordCNAME) *RecordCNAME {
	res := rc
	res.objectType = "record:cname"
	res.returnFields = []string{"extattrs", "canonical", "name", "view", "zone", "ttl"}

	return &res
}

///////////////////////////////////////////////////////////////////////////////

type RecordTXT struct {
	IBBase `json:"-"`
	Ref    string      `json:"_ref,omitempty"`
	Name   string      `json:"name,omitempty"`
	Text   string      `json:"text,omitempty"`
	View   string      `json:"view,omitempty"`
	Zone   string      `json:"zone,omitempty"`
	Ttl    uint        `json:"ttl,omitempty"`
	Ea     ibclient.EA `json:"extattrs,omitempty"`
}

func NewRecordTXT(rt RecordTXT) *RecordTXT {
	res := rt
	res.objectType = "record:txt"
	res.returnFields = []string{"extattrs", "name", "text", "view", "zone", "ttl"}

	return &res
}

///////////////////////////////////////////////////////////////////////////////

type RecordNS struct {
	IBBase     `json:"-"`
	Ref        string           `json:"_ref,omitempty"`
	Addresses  []ZoneNameServer `json:"addresses,omitempty"`
	Name       string           `json:"name,omitempty"`
	Nameserver string           `json:"nameserver,omitempty"`
	View       string           `json:"view,omitempty"`
	Zone       string           `json:"zone,omitempty"`
	Ea         ibclient.EA      `json:"extattrs,omitempty"`
}

func NewRecordNS(rc RecordNS) *RecordNS {
	res := rc
	res.objectType = "record:ns"
	res.returnFields = []string{"extattrs", "addresses", "name", "nameserver", "view", "zone"}

	return &res
}

type ZoneNameServer struct {
	Address string `json:"address,omitempty"`
}
