#!/bin/bash

# Usage
# updateLdapConfig.sh [-h HOST] [-s LDAP_SECRET]
#
# Applies the configuration received in standard input to the OpenLdap server

SCRIPT_DIR="$(dirname $0 )"

# Default values
HOST=127.0.0.1
SECRET=secretcr

# Read command line
while getopts 'h:s:' opt; do
  case $opt in
    h) HOST=$OPTARG ;;
    s) SECRET=$OPTARG ;;
  esac
done

# Get the current configuration and store in file. Notice the use of ldif-wrap=no
# Another way to circunvect the wrapping is to pipe the output to sed -n '1 {h; $ !d}; $ {x; s/\n //g; p}; /^ / {H; d}; /^ /! {x; s/\n //g; p}'
# ldapsearch -o ldif-wrap=no -H ldapi:/// -Y EXTERNAL -b "cn=config" '(!(objectClass=olcSchemaConfig))' "*" > /tmp/current.ldif
# Use this version of the command for testing
ldapsearch -h $HOST -w $SECRET -D "cn=admin,cn=config" -b "cn=config" '(!(objectClass=olcSchemaConfig))' "*" > /tmp/current.ldif

# The new configuration is read from standard input, in .conf format
# for f in $(find . -type f); do cat $f; echo; done
cat > /tmp/new.conf

# Generate the configuration in dynamic format
rm -rf /tmp/slapd.d && mkdir -p /tmp/slapd.d
slaptest -n 0 -f /tmp/new.conf -F /tmp/slapd.d

# Aggregate the .ldif files and pipe to ldifCompare
for f in $(find /tmp/slapd.d -type f); do cat $f; echo; done | $SCRIPT_DIR/../ldifCompare --current /tmp/current.ldif > /tmp/diff.ldif

# Apply changes
# ldapmodify -H ldapi:/// -Y EXTERNAL -D "cn=admin,cn=config" -f /tmp/diff.ldif
# ldapmodify -h $HOST -w $SECRET -D "cn=admin,cn=config" -f /tmp/diff.ldif
