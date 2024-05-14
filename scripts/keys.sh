#!/usr/bin/env bash

#
# Copyright (c) 2024 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

set -e

if [ "${COMPUTERNAME}" == "" ]
then
    COMPUTERNAME="127.0.0.1"
fi

CERTS_DIR=$1
if [ -z "${CERTS_DIR}" ] ; then
  echo "Please provide certs_dir"
  exit 1
fi

mkdir -p ${CERTS_DIR}

# Generate random passwords for each run
PASS="${RANDOM}_${RANDOM}"
KEYPASS="${PASS}"
STOREPASS="${PASS}"
CAPASS="${RANDOM}"
TRUSTPASS="${RANDOM}_${RANDOM}"

echo Generate Guardians CA key:
echo "${CAPASS}" | openssl genrsa -passout stdin -aes256 \
    -out ${CERTS_DIR}/guardians-ca.key 4096

echo Generate Guardians CA certificate:
echo "${CAPASS}" | openssl req -passin stdin -new -x509 -days 3650 \
    -reqexts SAN \
    -config <(cat /etc/ssl/openssl.cnf \
        <(printf "\n[SAN]\nsubjectAltName=IP:127.0.0.1")) \
    -key ${CERTS_DIR}/guardians-ca.key \
    -out ${CERTS_DIR}/guardians-ca.crt \
    -subj "/CN=${COMPUTERNAME}" # guardians-ca.crt is a trustCertCollectionFile

echo Generate Ravagers CA key:
echo "${CAPASS}" | openssl genrsa -passout stdin -aes256 \
    -out ${CERTS_DIR}/ravagers-ca.key 4096

echo Generate Ravagers CA certificate:
echo "${CAPASS}" | openssl req -passin stdin -new -x509 -days 3650 \
    -reqexts SAN \
    -config <(cat /etc/ssl/openssl.cnf \
        <(printf "\n[SAN]\nsubjectAltName=IP:127.0.0.1")) \
    -key ${CERTS_DIR}/ravagers-ca.key \
    -out ${CERTS_DIR}/ravagers-ca.crt \
    -subj "/CN=${COMPUTERNAME}" # ravagers-ca.crt is a trustCertCollectionFile

echo Generate Icarus key:
echo "${CAPASS}" |  openssl genrsa -passout stdin -aes256 \
    -out ${CERTS_DIR}/icarus.key 4096

echo Generate Icarus signing request:
echo "${CAPASS}" | openssl req -passin stdin -new -key \
    ${CERTS_DIR}/icarus.key \
    -out ${CERTS_DIR}/icarus.csr \
    -subj "/CN=${COMPUTERNAME}"

echo Self-signed Icarus certificate:
echo "${CAPASS}" | openssl x509 -req -passin stdin -days 3650 \
    -in ${CERTS_DIR}/icarus.csr \
    -CA ${CERTS_DIR}/guardians-ca.crt \
    -CAkey ${CERTS_DIR}/guardians-ca.key \
    -set_serial 01 \
    -out ${CERTS_DIR}/icarus.crt # icarus.crt is the certChainFile for the server

echo Remove passphrase from Icarus key:
echo "${CAPASS}" | openssl rsa -passin stdin \
    -in ${CERTS_DIR}/icarus.key \
    -out ${CERTS_DIR}/icarus.key

echo Generate client Star-Lord key
echo "${CAPASS}" | openssl genrsa -passout stdin -aes256 \
    -out ${CERTS_DIR}/star-lord.key 4096

echo Generate client Star-Lord signing request:
echo "${CAPASS}" | openssl req -passin stdin -new \
    -key ${CERTS_DIR}/star-lord.key \
    -out ${CERTS_DIR}/star-lord.csr -subj "/CN=Star-Lord"

echo Self-signed client Star-Lord certificate:
echo "${CAPASS}" | openssl x509 -passin stdin -req -days 3650 \
    -in ${CERTS_DIR}/star-lord.csr \
    -CA ${CERTS_DIR}/guardians-ca.crt \
    -CAkey ${CERTS_DIR}/guardians-ca.key \
    -set_serial 01 \
    -out ${CERTS_DIR}/star-lord.crt # star-lord.crt is the certChainFile for the client (Mutual TLS only)

echo Remove passphrase from Star-Lord key:
echo "${CAPASS}" | openssl rsa -passin stdin \
    -in ${CERTS_DIR}/star-lord.key \
    -out ${CERTS_DIR}/star-lord.key

echo Generate client Groot key
echo "${CAPASS}" | openssl genrsa -passout stdin -aes256 \
    -out ${CERTS_DIR}/groot.key 4096

echo Generate client Groot signing request:
echo "${CAPASS}" | openssl req -passin stdin -new \
    -key ${CERTS_DIR}/groot.key \
    -out ${CERTS_DIR}/groot.csr \
    -subj "/CN=Groot"

echo Self-signed client Groot certificate:
echo "${CAPASS}" | openssl x509 -passin stdin -req -days 3650 \
    -in ${CERTS_DIR}/groot.csr -CA ${CERTS_DIR}/guardians-ca.crt \
    -CAkey ${CERTS_DIR}/guardians-ca.key \
    -set_serial 01 \
    -out ${CERTS_DIR}/groot.crt # groot.crt is the certChainFile for the client (Mutual TLS only)

echo Remove passphrase from client Groot key:
echo "${CAPASS}" |openssl rsa -passin stdin \
    -in ${CERTS_DIR}/groot.key \
    -out ${CERTS_DIR}/groot.key

echo Generate client Yondu key
echo "${CAPASS}" | openssl genrsa -passout stdin -aes256 \
    -out ${CERTS_DIR}/yondu.key 4096

echo Generate client Yondu signing request:
echo "${CAPASS}" | openssl req -passin stdin -new \
    -key ${CERTS_DIR}/yondu.key \
    -out ${CERTS_DIR}/yondu.csr \
    -subj "/CN=Yondu"

echo Self-signed client Yondu certificate:
echo "${CAPASS}" | openssl x509 -passin stdin -req -days 3650 \
    -in ${CERTS_DIR}/yondu.csr \
    -CA ${CERTS_DIR}/ravagers-ca.crt \
    -CAkey ${CERTS_DIR}/ravagers-ca.key \
    -set_serial 01 \
    -out ${CERTS_DIR}/yondu.crt # yondu.crt is the certChainFile for the client (Mutual TLS only)

echo Remove passphrase from client Yondu key:
echo "${CAPASS}" | openssl rsa -passin stdin \
    -in ${CERTS_DIR}/yondu.key \
    -out ${CERTS_DIR}/yondu.key

openssl pkcs8 -topk8 -nocrypt \
    -in ${CERTS_DIR}/star-lord.key \
    -out ${CERTS_DIR}/star-lord.pem # star-lord.pem is the privateKey for the Client (mutual TLS only)

openssl pkcs8 -topk8 -nocrypt \
    -in ${CERTS_DIR}/groot.key \
    -out ${CERTS_DIR}/groot.pem # groot.pem is the privateKey for the Client (mutual TLS only)

openssl pkcs8 -topk8 -nocrypt \
    -in ${CERTS_DIR}/yondu.key \
    -out ${CERTS_DIR}/yondu.pem # yondu.pem is the privateKey for the Client (mutual TLS only)

openssl pkcs8 -topk8 -nocrypt \
    -in ${CERTS_DIR}/icarus.key \
    -out ${CERTS_DIR}/icarus.pem # icarus.pem is the privateKey for the Server

# Create the Java trust store
rm ${CERTS_DIR}/*.jks || true

(echo "${TRUSTPASS}" ; echo "${TRUSTPASS}") | keytool -import -noprompt -trustcacerts \
    -alias guardians -file ${CERTS_DIR}/guardians-ca.crt \
    -keystore ${CERTS_DIR}/truststore-guardians.jks \
    -deststoretype JKS

(echo "${TRUSTPASS}" ; echo "${TRUSTPASS}") | keytool -import -noprompt -trustcacerts \
    -alias ravagers -file ${CERTS_DIR}/ravagers-ca.crt \
    -keystore ${CERTS_DIR}/truststore-ravagers.jks \
    -deststoretype JKS

(echo "${TRUSTPASS}" ; echo "${TRUSTPASS}") | keytool -import -noprompt -trustcacerts \
    -alias guardians -file ${CERTS_DIR}/guardians-ca.crt \
    -keystore ${CERTS_DIR}/truststore-all.jks \
    -deststoretype JKS

(echo "${TRUSTPASS}" ; echo "${TRUSTPASS}") | keytool -import -noprompt -trustcacerts \
    -alias ravagers -file ${CERTS_DIR}/ravagers-ca.crt \
    -keystore ${CERTS_DIR}/truststore-all.jks \
    -deststoretype JKS

(echo "${KEYPASS}" ; echo "${KEYPASS}") | openssl pkcs12 -export -passout stdin \
    -inkey ${CERTS_DIR}/icarus.pem \
    -name test -in ${CERTS_DIR}/icarus.crt \
    -out ${CERTS_DIR}/icarus.p12

(echo "${STOREPASS}"; echo "${STOREPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}") | keytool -importkeystore -noprompt \
    -srckeystore ${CERTS_DIR}/icarus.p12 \
    -srcstoretype pkcs12 \
    -destkeystore ${CERTS_DIR}/icarus.jks

echo "${KEYPASS}" | openssl pkcs12 -export -passout stdin \
    -inkey ${CERTS_DIR}/star-lord.pem \
    -name test -in ${CERTS_DIR}/star-lord.crt \
    -out ${CERTS_DIR}/star-lord.p12

(echo "${STOREPASS}"; echo "${STOREPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}") | keytool -importkeystore -noprompt \
    -srckeystore ${CERTS_DIR}/star-lord.p12 \
    -srcstoretype pkcs12 \
    -destkeystore ${CERTS_DIR}/star-lord.jks

echo "${KEYPASS}" | openssl pkcs12 -export -passout stdin \
    -inkey ${CERTS_DIR}/groot.pem \
    -name test -in ${CERTS_DIR}/groot.crt \
    -out ${CERTS_DIR}/groot.p12

(echo "${STOREPASS}"; echo "${STOREPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}") | keytool -importkeystore -noprompt \
    -srckeystore ${CERTS_DIR}/groot.p12 \
    -srcstoretype pkcs12 \
    -destkeystore ${CERTS_DIR}/groot.jks

echo "${KEYPASS}" | openssl pkcs12 -export -passout stdin \
    -inkey ${CERTS_DIR}/yondu.pem \
    -name test -in ${CERTS_DIR}/yondu.crt \
    -out ${CERTS_DIR}/yondu.p12

(echo "${STOREPASS}"; echo "${STOREPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}"; echo "${KEYPASS}") | keytool -importkeystore -noprompt \
    -srckeystore ${CERTS_DIR}/yondu.p12 \
    -srcstoretype pkcs12 \
    -destkeystore ${CERTS_DIR}/yondu.jks
