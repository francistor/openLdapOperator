#!/bin/bash

# Produces an ldif file navigating through the directory structure in openldap configuration
# where each directory name is part of the dn path, and the files contain ldif

function dump_ldif_from_tree(){

    # Keep track of the directory passed in the first invocation, since it will be discarded
    # when building the path to append to the dn
    if [ -z "$initial_path" ]
    then
        initial_path=$1
    fi

    # The name of the directory is to be appended to the full dn of each entry
    # replacing the / by , and removing the single quotes
    remaining_path=${1#"$initial_path"}
    relativedn=$(echo $remaining_path | tr / , | tr -d "." | tr -d "'" | cut -c 2-)

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

    echo relative_dn $relative_dn

    # -p will append / to directories, and then we filter them
    for file in $(ls -p $1 | grep -v /)
    do
        # cat $1/$file | awk -v postfix=$relative_dn '/^dn:/ {print $0","postfix;next}; /.*/ {print $0}'
        # Separator line
        echo
    done

    for dir in $(ls -A -p $1 | grep /)
    do
        # sed to remove the last character, which is an /
        dump_ldif_from_tree $(echo $1/$dir |  sed 's/.$//')
    done
}

dump_ldif_from_tree $1


