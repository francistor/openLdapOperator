package main

import (
	"fmt"
	"strings"
	"testing"
)

var currentLdif = `
# extended LDIF
#
# LDAPv3
# base <cn=config> with scope subtree
# filter: (!(objectClass=olcSchemaConfig))
# requesting: * 
#

# config
dn: cn=config
objectClass: olcGlobal
cn: config
olcConfigFile: slapd.conf
olcConfigDir: slapd.d
olcArgsFile: /usr/local/var/run/slapd.args
olcAttributeOptions: lang-
olcAuthzPolicy: none
olcConcurrency: 0
olcConnMaxPending: 100
olcConnMaxPendingAuth: 1000
olcGentleHUP: FALSE
olcIdleTimeout: 0
olcIndexSubstrIfMaxLen: 4
olcIndexSubstrIfMinLen: 2
olcIndexSubstrAnyLen: 4
olcIndexSubstrAnyStep: 2
olcIndexHash64: FALSE
olcIndexIntLen: 4
olcListenerThreads: 1
olcLocalSSF: 71
olcLogLevel: 0
olcMaxFilterDepth: 1000
olcPidFile: /usr/local/var/run/slapd.pid
olcReadOnly: FALSE
olcSaslAuxpropsDontUseCopyIgnore: FALSE
olcSaslSecProps: noplain,noanonymous
olcSockbufMaxIncoming: 262143
olcSockbufMaxIncomingAuth: 16777215
olcThreads: 16
olcThreadQueues: 1
olcTLSCACertificateFile: /usr/local/etc/openldap/cacerts/ca.cert.pem
olcTLSCertificateFile: /usr/local/etc/openldap/certs/myldap.crt
olcTLSCertificateKeyFile: /usr/local/etc/openldap/certs/myldap.key
olcTLSCRLCheck: none
olcTLSVerifyClient: allow
olcTLSProtocolMin: 0.0
olcToolThreads: 1
olcWriteTimeout: 0

# {0}config, config
dn: olcDatabase={0}config,cn=config
objectClass: olcDatabaseConfig
olcDatabase: {0}config
olcAccess: {0}to *  by * none
olcAddContentAcl: TRUE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 15
olcReadOnly: FALSE
olcRootDN: cn=admin,cn=config
olcRootPW: secret
olcSyncUseSubentry: FALSE
olcMonitoring: FALSE

# {1}mdb, config
dn: olcDatabase={1}mdb,cn=config
objectClass: olcDatabaseConfig
objectClass: olcMdbConfig
olcDatabase: {1}mdb
olcDbDirectory: /usr/local/var/openldap-data
olcSuffix: dc=minsait,dc=com
olcAddContentAcl: FALSE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 15
olcReadOnly: FALSE
olcRootDN: cn=Manager,dc=minsait,dc=com
olcRootPW: secret
olcSyncUseSubentry: FALSE
olcMonitoring: TRUE
olcDbNoSync: FALSE
olcDbIndex: objectClass eq
olcDbIndex: default eq,pres
olcDbIndex: mail
olcDbMaxReaders: 0
olcDbMaxSize: 1073741824
olcDbMode: 0600
olcDbSearchStack: 16
olcDbMaxEntrySize: 0
olcDbRtxnSize: 10000

# {-1}frontend, config
dn: olcDatabase={-1}frontend,cn=config
objectClass: olcDatabaseConfig
objectClass: olcFrontendConfig
olcDatabase: {-1}frontend
olcAddContentAcl: FALSE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 0
olcReadOnly: FALSE
olcSchemaDN: cn=Subschema
olcSyncUseSubentry: FALSE
olcMonitoring: FALSE


# {2}monitor, config
dn: olcDatabase={2}monitor,cn=config
objectClass: olcDatabaseConfig
olcDatabase: {2}monitor
olcAddContentAcl: FALSE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 15
olcReadOnly: FALSE
olcSyncUseSubentry: FALSE
olcMonitoring: FALSE

# search result
search: 2
result: 0 Success

# numResponses: 6
# numEntries: 5
`

var newLdif = `
# extended LDIF
#
# LDAPv3
# base <cn=config> with scope subtree
# filter: (!(objectClass=olcSchemaConfig))
# requesting: * 
#

# config
dn: cn=config
objectClass: olcGlobal
cn: config
olcConfigFile: slapd.conf
olcConfigDir: slapd.d
olcArgsFile: /usr/local/var/run/slapd.args
olcAttributeOptions: lang-
olcAuthzPolicy: none
olcConcurrency: 0
olcConnMaxPending: 100
olcConnMaxPendingAuth: 1000
olcGentleHUP: FALSE
olcIdleTimeout: 0
olcIndexSubstrIfMaxLen: 4
olcIndexSubstrIfMinLen: 2
olcIndexSubstrAnyLen: 4
olcIndexSubstrAnyStep: 2
olcIndexHash64: FALSE
olcIndexIntLen: 4
olcListenerThreads: 1
olcLocalSSF: 71
olcLogLevel: 0
olcMaxFilterDepth: 1000
olcPidFile: /usr/local/var/run/slapd.pid
olcReadOnly: FALSE
olcSaslAuxpropsDontUseCopyIgnore: FALSE
olcSaslSecProps: noplain,noanonymous
olcSockbufMaxIncoming: 262143
olcSockbufMaxIncomingAuth: 16777215
olcThreads: 16
olcThreadQueues: 1
olcTLSCACertificateFile: /usr/local/etc/openldap/cacerts/ca.cert.pem
olcTLSCertificateFile: /usr/local/etc/openldap/certs/myldap.crt
olcTLSCertificateKeyFile: /usr/local/etc/openldap/certs/myldap.key
olcTLSCRLCheck: none
olcTLSVerifyClient: allow
olcTLSProtocolMin: 0.0
olcToolThreads: 1
olcWriteTimeout: 0

# {0}config, config
dn: olcDatabase={0}config,cn=config
objectClass: olcDatabaseConfig
olcDatabase: {0}config
olcAccess: {0}to *  by * none
olcAddContentAcl: TRUE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 15
olcReadOnly: FALSE
olcRootDN: cn=admin,cn=config
olcRootPW: secret
olcSyncUseSubentry: FALSE
olcMonitoring: FALSE

# {1}mdb, config
dn: olcDatabase={1}mdb,cn=config
objectClass: olcDatabaseConfig
objectClass: olcMdbConfig
olcDatabase: {1}mdb
olcDbDirectory: /usr/local/var/openldap-data
olcSuffix: dc=minsait,dc=com
olcAddContentAcl: FALSE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 15
olcReadOnly: FALSE
olcRootDN: cn=Manager,dc=minsait,dc=com
olcRootPW: secret
olcSyncUseSubentry: FALSE
olcMonitoring: TRUE
olcDbNoSync: FALSE
olcDbIndex: objectClass eq
olcDbIndex: default eq,pres
olcDbIndex: mail
olcDbMaxReaders: 0
olcDbMaxSize: 1073741824
olcDbMode: 0600
olcDbSearchStack: 16
olcDbMaxEntrySize: 0
olcDbRtxnSize: 10000

# {-1}frontend, config
dn: olcDatabase={-1}frontend,cn=config
objectClass: olcDatabaseConfig
objectClass: olcFrontendConfig
olcDatabase: {-1}frontend
olcAddContentAcl: FALSE
olcLastMod: TRUE
olcLastBind: TRUE
olcMaxDerefDepth: 0
olcReadOnly: FALSE
olcSchemaDN: cn=Subschema
olcSyncUseSubentry: FALSE
# This attribute is missing and has to be deleted
# olcMonitoring: FALSE
# This attribute has to be added
newAttribute: newAttribute!


# {2}monitor, config
# Delete this entry

# Add this entry
dn: newEntry
theAttr: theValue

# search result
search: 2
result: 0 Success

# numResponses: 6
# numEntries: 5
`

func TestParseLdif(t *testing.T) {

}

func TestLdifCompare(t *testing.T) {
	ldapModify := compareLdif(parseLdif(newLdif), parseLdif(currentLdif))
	fmt.Println("\n-------------------")
	fmt.Println(ldapModify)

	/*
		dn: newEntry
		changetype: add
		theAttr: theValue

		dn: olcDatabase={-1}frontend,cn=config
		changetype: modify
		-
		add: newAttribute
		newAttribute: newAttribute!
		-
		delete: olcMonitoring
		olcMonitoring: FALSE

		dn: olcDatabase={2}monitor,cn=config
		changetype: delete
	*/

	if !strings.Contains(ldapModify, "dn: newEntry\nchangetype: add\ntheAttr: theValue") {
		t.Fatal("Missing", "dn: newEntry\nchangetype: add\ntheAttr: theValue")
	}
	if !strings.Contains(ldapModify, "dn: olcDatabase={-1}frontend,cn=config\nchangetype: modify\n-\nadd: newAttribute\nnewAttribute: newAttribute!") {
		t.Fatal("Missing", "dn: olcDatabase={-1}frontend,cn=config\nchangetype: modify\n-\nadd: newAttribute\nnewAttribute: newAttribute!")
	}
	if !strings.Contains(ldapModify, "delete: olcMonitoring\nolcMonitoring: FALSE") {
		t.Fatal("Missing", "delete: olcMonitoring\nolcMonitoring: FALSE")
	}
	if !strings.Contains(ldapModify, "dn: olcDatabase={2}monitor,cn=config\nchangetype: delete") {
		t.Fatal("Missing", "dn: olcDatabase={2}monitor,cn=config\nchangetype: delete")
	}

}
