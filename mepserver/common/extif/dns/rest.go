/*
 * Copyright 2020 Huawei Technologies Co., Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package path implements dns client
package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/apache/servicecomb-service-center/pkg/log"

	meputil "mepserver/common/util"
)

var remoteServerHost = meputil.DefaultDnsHost
var remoteServerPort = meputil.DefaultDnsManagementPort

func init() {
	host := os.Getenv("DNS_SERVER_HOST")
	if len(host) > meputil.MaxFQDNLength {
		log.Warn("invalid dns remote server host configured, reset back to default")
	} else {
		remoteServerHost = host
	}

	port := os.Getenv("DNS_SERVER_PORT")
	if len(port) > meputil.MaxPortLength {
		log.Warn("invalid dns remote server port configured, reset back to default")
	} else if num, err := strconv.Atoi(port); err == nil {
		if num <= 0 || num > meputil.MaxPortNumber {
			log.Warn("invalid dns remote server port range, reset back to default")
		} else {
			remoteServerPort = num
		}
	}
}

type ResourceRecord struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Class string   `json:"class"`
	TTL   uint32   `json:"ttl"`
	RData []string `json:"rData"`
}

type ZoneEntry struct {
	Zone string            `json:"zone"`
	RR   *[]ResourceRecord `json:"rr"`
}

type RestClient struct {
	DNSAgent
	ServerEndPoint *url.URL `json:"serverEndPoint"`
}

func NewRestClient() *RestClient {
	u, err := url.Parse(fmt.Sprintf("http://%s:%d/mep/dns_server_mgmt/v1/", remoteServerHost, remoteServerPort))
	if err != nil {
		log.Errorf(nil, "could not parse the DNS server endpoint.")
		return &RestClient{}
	}
	return &RestClient{ServerEndPoint: u}
}

func (d *RestClient) SetResourceRecordTypeA(host, rrtype, class string, pointTo []string, ttl uint32) (resp *http.Response, err error) {
	if d.ServerEndPoint == nil {
		log.Errorf(nil, "invalid dns remote end point")
		return nil, fmt.Errorf("invalid dns server endpoint")
	}

	hostName := host
	if !strings.HasSuffix(host, ".") {
		hostName = host + "."
	}

	zones := []ZoneEntry{{Zone: ".", RR: &[]ResourceRecord{
		{Name: hostName, Type: rrtype, Class: class, TTL: ttl, RData: pointTo}}}}
	zoneJSON, err := json.Marshal(zones)
	if err != nil {
		log.Errorf(nil, "marshal dns info failed")
		return nil, err
	}

	httpClient := &http.Client{}
	httpReq, err := http.NewRequest(http.MethodPut,
		d.joinURL(d.ServerEndPoint.String(), "rrecord"),
		bytes.NewBuffer(zoneJSON))
	if err != nil {
		log.Errorf(nil, "http request creation for dns update failed.")
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Errorf(nil, "request to dns server failed in update")
		return nil, err
	}
	if !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		log.Errorf(nil, "dns rule update failed on server(%d: %s).", httpResp.StatusCode, httpResp.Status)
		return nil, fmt.Errorf("update request to dns server failed")
	}
	return httpResp, nil

}

func (d *RestClient) DeleteResourceRecordTypeA(host, rrtype string) (resp *http.Response, err error) {
	if d.ServerEndPoint == nil {
		log.Errorf(nil, "invalid dns remote end point")
		return nil, fmt.Errorf("invalid dns server endpoint")
	}
	hostName := host
	if !strings.HasSuffix(host, ".") {
		hostName = host + "."
	}

	httpClient := &http.Client{}

	httpReq, err := http.NewRequest(
		http.MethodDelete,
		d.joinURL(d.ServerEndPoint.String(), "rrecord", hostName, rrtype),
		bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Errorf(nil, "http request creation for dns delete failed")
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Errorf(nil, "request to dns server failed in delete")
		return nil, err
	}
	if !meputil.IsHttpStatusOK(httpResp.StatusCode) {
		log.Errorf(nil, "dns rule delete failed on server(%d: %s).", httpResp.StatusCode, httpResp.Status)
		return nil, fmt.Errorf("delete request to dns server failed")
	}
	return httpResp, nil
}

func (d *RestClient) joinURL(base string, paths ...string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"),
		strings.TrimLeft(path.Join(paths...), "/"))
}
