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

package infoblox

import (
	ibclient "github.com/infobloxopen/infoblox-go-client"

	"github.com/gardener/external-dns-management/pkg/controller/provider/infoblox/sdk"
	"github.com/gardener/external-dns-management/pkg/dns"
	"github.com/gardener/external-dns-management/pkg/dns/provider"
)

type Record interface {
	ibclient.IBObject

	Id() string
	Type() string
	Value() string
	DNSName() string
	TTL() int
	SetTTL(int)
	Copy() Record
}

type RecordA sdk.RecordA

func (r *RecordA) Type() string    { return dns.RS_A }
func (r *RecordA) Id() string      { return r.Ref }
func (r *RecordA) DNSName() string { return r.Name }
func (r *RecordA) Value() string   { return r.Ipv4Addr }
func (r *RecordA) TTL() int        { return int(r.Ttl) }
func (r *RecordA) SetTTL(ttl int)  {}
func (r *RecordA) Copy() Record    { n := *r; return &n }

type RecordCNAME sdk.RecordCNAME

func (r *RecordCNAME) Type() string    { return dns.RS_CNAME }
func (r *RecordCNAME) Id() string      { return r.Ref }
func (r *RecordCNAME) DNSName() string { return r.Name }
func (r *RecordCNAME) Value() string   { return r.Canonical }
func (r *RecordCNAME) TTL() int        { return int(r.Ttl) }
func (r *RecordCNAME) SetTTL(ttl int)  {}
func (r *RecordCNAME) Copy() Record    { n := *r; return &n }

type RecordTXT sdk.RecordTXT

func (r *RecordTXT) Type() string    { return dns.RS_TXT }
func (r *RecordTXT) Id() string      { return r.Ref }
func (r *RecordTXT) DNSName() string { return r.Name }
func (r *RecordTXT) Value() string   { return r.Text }
func (r *RecordTXT) TTL() int        { return int(r.Ttl) }
func (r *RecordTXT) SetTTL(ttl int)  {}
func (r *RecordTXT) Copy() Record    { n := *r; return &n }

type RecordSet []Record
type DNSSet map[string]RecordSet

type zonestate struct {
	dnssets dns.DNSSets
	records map[string]DNSSet
}

var _ provider.DNSZoneState = &zonestate{}

func newState() *zonestate {
	return &zonestate{records: map[string]DNSSet{}}
}

func (this *zonestate) GetDNSSets() dns.DNSSets {
	return this.dnssets
}

func (this *zonestate) addRecord(r Record) {
	name := r.DNSName()
	t := r.Type()
	e := this.records[name]
	if e == nil {
		e = DNSSet{}
		this.records[name] = e
	}
	e[t] = append(e[t], r)
}

func (this *zonestate) getRecord(dnsname, rtype, value string) Record {
	e := this.records[dnsname]
	if e != nil {
		for _, r := range e[rtype] {
			if r.Value() == value {
				return r
			}
		}
	}
	return nil
}

func (this *zonestate) calculateDNSSets() {
	this.dnssets = dns.DNSSets{}
	for dnsname, dset := range this.records {
		for rtype, rset := range dset {
			rs := dns.NewRecordSet(rtype, 0, nil)
			for _, r := range rset {
				rs.Add(&dns.Record{Value: r.Value()})
			}
			this.dnssets.AddRecordSetFromProvider(dnsname, rs)
		}
	}
}
