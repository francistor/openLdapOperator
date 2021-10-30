#!/bin/bash

SCRIPT_DIR="$(dirname $0 )"

echo ' 
# Configuration from Custom Resource Definition
# This file should NOT be world readable.
#
include /usr/local/etc/openldap/schema/core.schema
include /usr/local/etc/openldap/schema/cosine.schema 
include /usr/local/etc/openldap/schema/inetorgperson.schema 

# Define global ACLs to disable default read access.

# Do not enable referrals until AFTER you have a working directory
# service AND an understanding of referrals.
#referral	ldap://root.openldap.org

pidfile /usr/local/var/run/slapd.pid
argsfile /usr/local/var/run/slapd.args

# Load dynamic backend modules:
# modulepath	/usr/local/libexec/openldap
# moduleload	back_mdb.la
# moduleload	back_ldap.la

# Sample security restrictions
#	Require integrity protection (prevent hijacking)
#	Require 112-bit (3DES or better) encryption for updates
#	Require 63-bit encryption for simple bind
# security ssf=1 update_ssf=112 simple_bind=64

# Sample access control policy:
#	Root DSE: allow anyone to read it
#	Subschema (sub)entry DSE: allow anyone to read it
#	Other DSEs:
#		Allow self write access
#		Allow authenticated users read access
#		Allow anonymous users to authenticate
#	Directives needed to implement policy:
# access to dn.base="" by * read
# access to dn.base="cn=Subschema" by * read
# access to *
#	by self write
#	by users read
#	by anonymous auth
#
# if no access controls are present, the default policy
# allows anyone and everyone to read anything but restricts
# updates to rootdn.  (e.g., "access to * by * read")
#
# rootdn can always read and write EVERYTHING!

#######################################################################
# config database definitions
#######################################################################
database config
# Uncomment the rootpw line to allow binding as the cn=config
# rootdn so that temporary modifications to the configuration can be made
# while slapd is running. They will not persist across a restart.
rootdn "cn=admin,cn=config"
rootpw secretcr
# This allows access using ldapi:/// and integrated authentication, which seems to be disabled by default for db config
access to * by dn.exact=gidNumber=0+uidNumber=0,cn=peercred,cn=external,cn=auth manage by * break

#######################################################################
# MDB database definitions
#######################################################################

database mdb
maxsize 1073741824
suffix "dc=minsait,dc=com"
rootdn "cn=Manager,dc=minsait,dc=com"
# Cleartext passwords, especially for the rootdn, should
# be avoid.  See slappasswd(8) and slapd.conf(5) for details.
# Use of strong authentication encouraged.
rootpw secretcr
# The database directory MUST exist prior to running slapd AND 
# should only be accessible by the slapd and slap tools.
# Mode 700 recommended.
directory /usr/local/var/openldap-data
# Indices to maintain
index objectClass	eq
# Asnync writes <------------------ Change here!
dbnosync FALSE

#######################################################################
# monitor database definitions
#######################################################################
database monitor
' | $SCRIPT_DIR/../bin/updateLdapConfig.sh -h 192.168.122.210