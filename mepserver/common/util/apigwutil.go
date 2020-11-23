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

package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/astaxie/beego/httplib"
)

var cipherSuiteMap = map[string]uint16{
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

type RouteInfo struct {
	Id      int64   `json:"routeId"`
	AppId   string  `json:"appId"`
	SerInfo SerInfo `orm:"type(json)" json:"serInfo"`
}

type SerInfo struct {
	SerName string   `json:"serName"`
	Uris    []string `json:"uris"`
}

func AddApigwService(routeInfo RouteInfo) {
	appConfig, err := GetAppConfig()

	serName := routeInfo.SerInfo.SerName
	kongServiceUrl := fmt.Sprintf("https://%s:%s/services",
		appConfig["apigw_host"],
		appConfig["apigw_port"])
	serUrl := routeInfo.SerInfo.Uris[0]
	jsonStr := []byte(fmt.Sprintf(`{ "url": "%s", "name": "%s" }`, serUrl, serName))
	err = SendPostRequest(kongServiceUrl, jsonStr)
	if err != nil {
		log.Error(err)
	}
}

func AddApigwRoute(routeInfo RouteInfo) {
	appConfig, err := GetAppConfig()

	serName := routeInfo.SerInfo.SerName
	kongRouteUrl := fmt.Sprintf("https://%s:%s/services/%s/routes",
		appConfig["apigw_host"],
		appConfig["apigw_port"],
		serName)
	jsonStr := []byte(fmt.Sprintf(`{ "paths": ["/%s"], "name": "%s" }`, serName, serName))
	err = SendPostRequest(kongRouteUrl, jsonStr)
	if err != nil {
		log.Error(err)
	}
}

func ApigwDelRoute(serName string) {
	appConfig, err := GetAppConfig()

	kongRouteUrl := fmt.Sprintf("https://%s:%s/services/%s/routes/%s",
		appConfig["apigw_host"],
		appConfig["apigw_port"],
		serName, serName)
	req := httplib.Delete(kongRouteUrl)
	str, err := req.String()
	if err != nil {
		log.Error(err)
	}
	log.Infof("request=%s", str)
}

func GetAppConfig() (AppConfigProperties, error) {
	// read app.conf file to AppConfigProperties object
	configFilePath := filepath.FromSlash("/usr/mep/conf/app.conf")
	appConfig, err := readPropertiesFile(configFilePath)
	return appConfig, err
}

// Send post request
func SendPostRequest(consumerURL string, jsonStr []byte) error {

	req := httplib.Post(consumerURL)
	req.Header("Content-Type", "application/json; charset=utf-8")
	config, err := TLSConfig("apigw_cacert")
	if err != nil {
		log.Error("unable to read certificate")
		return err
	}
	req.SetTLSClientConfig(config)
	req.Body(jsonStr)
	res, err := req.String()
	if err != nil {
		log.Error("send Post Request Failed")
		return err
	}
	log.Infof("request=%s", res)
	return nil
}

// Update tls configuration
func TLSConfig(crtName string) (*tls.Config, error) {
	appConfig, err := GetAppConfig()
	if err != nil {
		log.Error("get app config error")
		return nil, err
	}
	certNameConfig := string(appConfig[crtName])
	if len(certNameConfig) == 0 {
		log.Error(crtName + " configuration is not set")
		return nil, errors.New("cert name configuration is not set")
	}

	crt, err := ioutil.ReadFile(certNameConfig)
	if err != nil {
		log.Error("unable to read certificate")
		return nil, err
	}

	rootCAs := x509.NewCertPool()
	ok := rootCAs.AppendCertsFromPEM(crt)
	if !ok {
		log.Error("failed to decode cert file")
		return nil, errors.New("failed to decode cert file")
	}

	serverName := string(appConfig["server_name"])
	serverNameIsValid, validateServerNameErr := ValidateServerName(serverName)
	if validateServerNameErr != nil || !serverNameIsValid {
		log.Error("validate server name error")
		return nil, validateServerNameErr
	}
	sslCiphers := string(appConfig["ssl_ciphers"])
	if len(sslCiphers) == 0 {
		return nil, errors.New("TLS cipher configuration is not recommended or invalid")
	}
	cipherSuites := getCipherSuites(sslCiphers)
	if cipherSuites == nil {
		return nil, errors.New("TLS cipher configuration is not recommended or invalid")
	}
	return &tls.Config{
		RootCAs:      rootCAs,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: cipherSuites,
	}, nil
}

// Validate Server Name
func ValidateServerName(serverName string) (bool, error) {
	if len(serverName) > maxHostNameLen {
		return false, errors.New("server or host name validation failed")
	}
	return regexp.MatchString(ServerNameRegex, serverName)
}

func getCipherSuites(sslCiphers string) []uint16 {
	cipherSuiteArr := make([]uint16, 0, 5)
	cipherSuiteNameList := strings.Split(sslCiphers, ",")
	for _, cipherName := range cipherSuiteNameList {
		cipherName = strings.TrimSpace(cipherName)
		if len(cipherName) == 0 {
			continue
		}
		mapValue, ok := cipherSuiteMap[cipherName]
		if !ok {
			log.Error("not recommended cipher suite")
			return nil
		}
		cipherSuiteArr = append(cipherSuiteArr, mapValue)
	}
	if len(cipherSuiteArr) > 0 {
		return cipherSuiteArr
	}
	return nil
}