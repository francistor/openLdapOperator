#!/bin/bash

SCRIPT_DIR="$(dirname $0 )"

go build $SCRIPT_DIR/../ldifcompare.go

$SCRIPT_DIR/../ldifcompare --current $SCRIPT_DIR/../resources/current_ldif.txt --new $SCRIPT_DIR/../resources/new_ldif.txt --debug