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
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/gardener/controller-manager-library/pkg/logger"
	"github.com/gardener/controller-manager-library/pkg/utils"
	"golang.org/x/oauth2/google"
	googledns "google.golang.org/api/dns/v1"

	"github.com/gardener/external-dns-management/pkg/controller/provider/infoblox/sdk"
	"github.com/gardener/external-dns-management/pkg/dns"
	"github.com/gardener/external-dns-management/pkg/dns/provider"
	"github.com/gardener/external-dns-management/pkg/dns/provider/raw"

	ibclient "github.com/infobloxopen/infoblox-go-client"
)

type Handler struct {
	provider.ZoneCache
	provider.DefaultDNSHandler
	config         provider.DNSHandlerConfig
	infobloxConfig InfobloxConfig
	credentials    *google.Credentials
	access         *access
	ctx            context.Context
	service        *googledns.Service
}

type InfobloxConfig struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	SSLVerify       *bool   `json:"sslVerify,omitempty"`
	Version         *string `json:"version,omitempty"`
	View            *string `json:"view,omitempty"`
	PoolConnections *int    `json:"httpPoolConnections,omitempty"`
	RequestTimeout  *int    `json:"httpRequestTimeout,omitempty"`
	CaCert          *string `json:"caCert,omitempty"`
	MaxResults      int     `json:"maxResults,omitempty"`
}

var _ provider.DNSHandler = &Handler{}

func NewHandler(config *provider.DNSHandlerConfig) (provider.DNSHandler, error) {

	infobloxConfig := InfobloxConfig{}
	if config.Config != nil {
		err := json.Unmarshal(config.Config.Raw, &infobloxConfig)
		if err != nil {
			return nil, fmt.Errorf("unmarshal infoblox providerConfig failed with: %s", err)
		}
	}

	h := &Handler{
		DefaultDNSHandler: provider.NewDefaultDNSHandler(TYPE_CODE),
		config:            *config,
		infobloxConfig:    infobloxConfig,
		ctx:               config.Context,
	}

	username, err := config.GetRequiredProperty("USERNAME", "username")
	if err != nil {
		return nil, err
	}
	password, err := config.GetRequiredProperty("PASSWORD", "password")
	if err != nil {
		return nil, err
	}

	if err := config.FillRequiredProperty(&infobloxConfig.Version, "VERSION", "password"); err != nil {
		return nil, err
	}
	if err := config.FillRequiredProperty(&infobloxConfig.View, "VIEW", "view"); err != nil {
		return nil, err
	}
	if err := config.FillRequiredProperty(&infobloxConfig.Host, "HOST", "host"); err != nil {
		return nil, err
	}
	if err := config.FillRequiredIntProperty(&infobloxConfig.Port, "PORT", "port"); err != nil {
		return nil, err
	}
	if err := config.FillDefaultedIntProperty(&infobloxConfig.PoolConnections, 10, "HTTP_POOL_CONNECTIONS", "http_pool_connections"); err != nil {
		return nil, err
	}
	if err := config.FillDefaultedIntProperty(&infobloxConfig.RequestTimeout, 60, "HTTP_REQUEST_TIMEOUT", "http_request_timeout"); err != nil {
		return nil, err
	}

	config.Logger.Infof("creating infoblox handler for %s", *infobloxConfig.Host)

	hostConfig := ibclient.HostConfig{
		Host:     *infobloxConfig.Host,
		Port:     strconv.Itoa(*infobloxConfig.Port),
		Version:  *infobloxConfig.Version,
		Username: username,
		Password: password,
	}

	verify := "true"
	if infobloxConfig.SSLVerify != nil {
		verify = strconv.FormatBool(*infobloxConfig.SSLVerify)
	}
	transportConfig := ibclient.NewTransportConfig(
		verify,
		*infobloxConfig.RequestTimeout,
		*infobloxConfig.PoolConnections,
	)

	if infobloxConfig.CaCert != nil && verify == "true" {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM([]byte(*infobloxConfig.CaCert)) {
			return nil, fmt.Errorf("Cannot append certificate")
		}
		utils.SetValue(reflect.ValueOf(transportConfig).FieldByName("certPool"), caPool)
	}

	var requestBuilder ibclient.HttpRequestBuilder = &ibclient.WapiRequestBuilder{}
	if infobloxConfig.MaxResults != 0 {
		// wrap request builder which sets _max_results parameter on GET requests
		requestBuilder = NewMaxResultsRequestBuilder(infobloxConfig.MaxResults, requestBuilder)
	}
	client, err := ibclient.NewConnector(hostConfig, transportConfig, requestBuilder, &ibclient.WapiHttpRequestor{})
	if err != nil {
		return nil, err
	}

	h.access = NewAccess(client, *h.infobloxConfig.View, config.Metrics)

	h.ZoneCache, err = provider.NewZoneCache(config.CacheConfig, config.Metrics, nil, h.getZones, h.getZoneState)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// Infoblox does not support zone forwarding???
// Just removed the forwarding stuff from code

func (h *Handler) getZones(cache provider.ZoneCache) (provider.DNSHostedZones, error) {
	var raw []ibclient.ZoneAuth
	h.config.Metrics.AddRequests(provider.M_LISTZONES, 1)
	obj := ibclient.NewZoneAuth(ibclient.ZoneAuth{})
	err := h.access.GetObject(obj, "", &raw)
	if err != nil {
		return nil, err
	}

	zones := provider.DNSHostedZones{}
	for _, z := range raw {
		h.config.Metrics.AddRequests(provider.M_LISTRECORDS, 1)
		var resN []sdk.RecordNS
		objN := sdk.NewRecordNS(
			sdk.RecordNS{
				Zone: z.Fqdn,
				View: *h.infobloxConfig.View,
			},
		)
		err = h.access.GetObject(objN, "", &resN)
		if err != nil {
			return nil, fmt.Errorf("could not fetch NS records from zone '%s': %s", z.Fqdn, err)
		}
		forwarded := []string{}
		for _, res := range resN {
			if res.Name != z.Fqdn {
				forwarded = append(forwarded, res.Name)
			}
		}
		hostedZone := provider.NewDNSHostedZone(h.ProviderType(), z.Ref, dns.NormalizeHostname(z.Fqdn), z.Fqdn, forwarded, false)
		zones = append(zones, hostedZone)
	}
	return zones, nil
}

func (h *Handler) getZoneState(zone provider.DNSHostedZone, cache provider.ZoneCache) (provider.DNSZoneState, error) {
	state := raw.NewState()
	rt := provider.M_LISTRECORDS

	h.config.Metrics.AddRequests(rt, 1)
	var resA []sdk.RecordA
	objA := sdk.NewRecordA(
		sdk.RecordA{
			Zone: zone.Key(),
			View: *h.infobloxConfig.View,
		},
	)
	err := h.access.GetObject(objA, "", &resA)
	if err != nil {
		return nil, fmt.Errorf("could not fetch A records from zone '%s': %s", zone.Key(), err)
	}
	for _, res := range resA {
		state.AddRecord((*RecordA)(&res).Copy())
	}

	h.config.Metrics.AddRequests(rt, 1)
	var resC []sdk.RecordCNAME
	objC := sdk.NewRecordCNAME(
		sdk.RecordCNAME{
			Zone: zone.Key(),
			View: *h.infobloxConfig.View,
		},
	)
	err = h.access.GetObject(objC, "", &resC)
	if err != nil {
		return nil, fmt.Errorf("could not fetch CNAME records from zone '%s': %s", zone.Key(), err)
	}
	for _, res := range resC {
		state.AddRecord((*RecordCNAME)(&res).Copy())
	}

	h.config.Metrics.AddRequests(rt, 1)
	var resT []sdk.RecordTXT
	objT := sdk.NewRecordTXT(
		sdk.RecordTXT{
			Zone: zone.Key(),
			View: *h.infobloxConfig.View,
		},
	)
	err = h.access.GetObject(objT, "", &resT)
	if err != nil {
		return nil, fmt.Errorf("could not fetch TXT records from zone '%s': %s", zone.Key(), err)
	}
	for _, res := range resT {
		state.AddRecord((*RecordTXT)(&res).Copy())
	}

	state.CalculateDNSSets()
	return state, nil
}

func (h *Handler) ExecuteRequests(logger logger.LogContext, zone provider.DNSHostedZone, state provider.DNSZoneState, reqs []*provider.ChangeRequest) error {
	err := raw.ExecuteRequests(logger, &h.config, h.access, zone, state, reqs)
	h.ApplyRequests(err, zone, reqs)
	return err
}
