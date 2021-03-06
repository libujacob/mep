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

// Package works for the mep server entry
package main

import (
	"errors"
	"mepserver/mp1/plans"
	"os"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/server"
	_ "github.com/apache/servicecomb-service-center/server/bootstrap"
	_ "github.com/apache/servicecomb-service-center/server/init"
	_ "mepserver/common/tls"
	"mepserver/common/util"
	_ "mepserver/mm5"
	_ "mepserver/mp1"
	_ "mepserver/mp1/uuid"
	_ "mepserver/mp1/event"
)

func main() {

	err := initialEncryptComponent()
	if err != nil {
		log.Errorf(err, "initial encrypt component failed")
		return
	}
	if !util.IsFileOrDirExist(util.EncryptedCertSecFilePath) {
		err := encryptCertPwd()
		if err != nil {
			log.Errorf(err, "input cert pwd failed")
			return
		}

	}
	go plans.HeartbeatProcess()
	server.Run()
}

func encryptCertPwd() error {
	pwd := []byte(os.Getenv("TLS_KEY"))
	if len(os.Getenv("TLS_KEY")) == 0 {
		err := errors.New("tls password is not set in environment variable")
		log.Errorf(err, "read password failed")
		return err
	}
	os.Unsetenv("TLS_KEY")
	_, verifyErr := util.ValidatePassword(&pwd)
	if verifyErr != nil {
		log.Errorf(verifyErr, "Certificate password complexity validation failed")
		return verifyErr
	}
	encryptCertPwdErr := util.EncryptAndSaveCertPwd(&pwd)
	if encryptCertPwdErr != nil {
		log.Errorf(encryptCertPwdErr, "encrypt cert pwd failed")
		return encryptCertPwdErr
	}
	return nil
}

func initialEncryptComponent() error {
	keyComponentFromUser := []byte(os.Getenv("ROOT_KEY"))
	if len(os.Getenv("ROOT_KEY")) == 0 {
		err := errors.New("root key is not present inside environment variable")
		log.Errorf(err, "read root key component failed")
		return err
	}
	os.Unsetenv("ROOT_KEY")
	verifyErr := util.ValidateKeyComponentUserInput(&keyComponentFromUser)
	if verifyErr != nil {
		log.Errorf(verifyErr, "root key component from user validation failed")
		return verifyErr
	}
	util.KeyComponentFromUserStr = &keyComponentFromUser

	err := util.InitRootKeyAndWorkKey()
	if err != nil {
		log.Errorf(err, "failed to init root key and work key")
		return err
	}
	return nil
}
