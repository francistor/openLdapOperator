#!/bin/bash

# Usage
# updateLdapConfig.sh [-h HOST] [-s LDAP_SECRET]
#
# Applies the configuration received in standard input to the OpenLdap server

# Produces an ldif file navigating through the directory structure in openldap configuration
# where each directory name is part of the dn path, and the files contain ldif

function dump_ldif_from_tree(){

    # Build the relative dn to append to the dn in entries as the sequence of directorie names traveled
    # separated by commas (according to the way Openldap stores the configuration in slapd.d directory)
    if [ -z ${relative_dn+dummy} ]
    then
        # The variable was unset, and we now set it to empty string
        relative_dn=""
    else
        # Get an array, using / as the separator. The () will force array creation. The space after the last / is
        # significant: replace / by <space>
        dirItems=(${1//\// })
        last_dir=${dirItems[-1]}
        if [ -z "$relative_dn" ]
        then
            relative_dn=,$last_dir
        else
            relative_dn=,$last_dir$relative_dn
        fi
    fi

    echo $relative_dn >> /tmp/debug.log

    # -p will append / to directories, and then we filter them
    for file in $(ls -p $1 | grep -v /)
    do
        cat $1/$file | awk -v postfix=$relative_dn '/^dn:/ {print $0postfix;next}; /.*/ {print $0}'
        # Separator line
        echo
    done

    for dir in $(ls -A -p $1 | grep /)
    do
        # sed to remove the last character, which is an /
        dump_ldif_from_tree $(echo $1/$dir |  sed 's/.$//')
    done
}

SCRIPT_DIR="$(dirname $0 )"

# Read command line
while getopts 'h:s:' opt; do
  case $opt in
    h) HOST=$OPTARG ;;
    s) SECRET=$OPTARG ;;
  esac
done

# Get the current configuration and store in file. Notice the use of ldif-wrap=no
# Another way to circunvect the wrapping is to pipe the output to sed -n '1 {h; $ !d}; $ {x; s/\n //g; p}; /^ / {H; d}; /^ /! {x; s/\n //g; p}'
# If host or secret are specified, use explicit authentication. Otherwise, use integrated authentication, asumming root
if [ -z "$HOST" ] && [ -z "$SECRET"]
then
  # ldapsearch -o ldif-wrap=no -H ldapi:/// -Y EXTERNAL -b "cn=config" '(!(objectClass=olcSchemaConfig))' "*" > /tmp/current.ldif
  ldapsearch -o ldif-wrap=no -H ldapi:/// -Y EXTERNAL -b "cn=config" "*" > /tmp/current.ldif
else
  ldapsearch -o ldif-wrap=no -h ${HOST:-127.0.0.1} -w ${SECRET:-secretcr} -D "cn=admin,cn=config" -b "cn=config" "*" > /tmp/current.ldif
fi

# The new configuration is read from standard input, in .conf format
# for f in $(find . -type f); do cat $f; echo; done
cat > /tmp/new.conf

# Generate the configuration in dynamic format
rm -rf /tmp/slapd.d && mkdir -p /tmp/slapd.d
slaptest -n 0 -f /tmp/new.conf -F /tmp/slapd.d

# Aggregate the .ldif files using function | unwrap lines | generate changes to apply
dump_ldif_from_tree /tmp/slapd.d | 
  sed -n '1 {h; $ !d}; $ {x; s/\n //g; p}; /^ / {H; d}; /^ /! {x; s/\n //g; p}' > /tmp/new.ldif

dump_ldif_from_tree /tmp/slapd.d | 
  sed -n '1 {h; $ !d}; $ {x; s/\n //g; p}; /^ / {H; d}; /^ /! {x; s/\n //g; p}' |
  awk '/modifyTimestamp:|modifiersName:|entryUUID:|entryCSN:|creatorsName:|createTimestamp:|structuralObjectClass:/ {next}; /.*/ {print $0}' |
  $SCRIPT_DIR/../ldifCompare --current /tmp/current.ldif > /tmp/diff.ldif

# Apply changes
if [ -z "$HOST" ] && [ -z "$SECRET"]
then
  ldapmodify -H ldapi:/// -Y EXTERNAL -D "cn=admin,cn=config" -f /tmp/diff.ldif
else
  ldapmodify -h ${HOST:-127.0.0.1} -w ${SECRET:-secretcr} -f /tmp/diff.ldif
fi
