#!/bin/sh
# Copyright 2020 Huawei Technologies Co., Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Contain at most 63 characters
# Contain only lowercase alphanumeric characters or '-'
# Start with an alphanumeric character
# End with an alphanumeric character
validate_host_name()
{
 hostname="$1"
 len="${#hostname}"
 if [ "${len}" -gt "253" ] ; then
   return 1
 fi
 if ! echo "$hostname" | grep -qE '^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$' ; then
   return 1
 fi
 return 0
}

# Validating if port is > 1 and < 65535 , not validating reserved port.
validate_port_num()
{
 portnum="$1"
 len="${#portnum}"
 if [ "${len}" -gt "5" ] ; then
   return 1
 fi
 if ! echo "$portnum" | grep -qE '^-?[0-9]+$' ; then
   return 1
 fi
 if [ "$portnum" -gt "65535" ] || [ "$portnum" -lt "1" ] ; then
   return 1
 fi
 return 0
}

validate_host_name "$MEPAUTH_APIGW_HOST"
valid_api_host_name="$?"
if [ ! "$valid_api_host_name" -eq "0" ] ; then
   echo "invalid apigw host name"
   exit 1
fi

validate_port_num "$MEPAUTH_APIGW_PORT"
valid_api_port="$?"
if [ ! "$valid_api_port" -eq "0" ] ; then
   echo "invalid apigw portnumber"
   exit 1
fi

validate_host_name "$MEPAUTH_CERT_DOMAIN_NAME"
valid_cert_host_name="$?"
if [ ! "$valid_cert_host_name" -eq "0" ] ; then
   echo "invalid cert host name"
   exit 1
fi

MEPSERVER_HOST="${MEPSERVER_HOST:-$(hostname -i)}"
validate_host_name "$MEPSERVER_HOST"
valid_mepserver_host_name="$?"
if [ ! "$valid_mepserver_host_name" -eq "0" ] ; then
   echo "invalid mep server host name"
   exit 1
fi

cd /usr/mep

set +e

sed -i "s/^apigw_host.*=.*$/apigw_host = ${MEPAUTH_APIGW_HOST}/g" conf/app.conf
sed -i "s/^apigw_port.*=.*$/apigw_port = ${MEPAUTH_APIGW_PORT}/g" conf/app.conf

sed -i "s/^server_name.*=.*$/server_name = ${MEPAUTH_CERT_DOMAIN_NAME}/g" conf/app.conf

sed -i "s/^HTTPSAddr.*=.*$/HTTPSAddr = $(hostname -i)/g" conf/app.conf
sed -i "s/^mepserver_host.*=.*$/mepserver_host = ${MEPSERVER_HOST}/g" conf/app.conf

# config db
sed -i "s/^db_name.*=.*$/db_name = ${MEPAUTH_DB_NAME}/g" conf/app.conf
sed -i "s/^db_user.*=.*$/db_user = ${MEPAUTH_DB_USER}/g" conf/app.conf
sed -i "s/^db_passwd.*=.*$/db_passwd = ${MEPAUTH_DB_PASSWD}/g" conf/app.conf
sed -i "s/^db_host.*=.*$/db_host = ${MEPAUTH_DB_HOST}/g" conf/app.conf

set -e

umask 0027

$HOME/bin/app
