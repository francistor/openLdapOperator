Bla la Bla

*** Settings ***
Library           Process
Suite Setup       Clean data
Suite Teardown    Terminate All Processes    kill=True

*** Variables ***
# Requres the initial-data.ldif file in the current directory
${ldaphost}    openldapsample-service
${ldapport}    389
${number_of_clients}    999    
${number_of_clients_digits}    3    # 999 has 3 digits
 

*** Test Cases ***
Load data
    ${result} =    Run Process    ldclt   -h   ${ldaphost}   -p   ${ldapport}   -w   secret   -D   cn\=Manager,dc\=minsait,dc\=com   -e   object\=person.txt,rdn\=cn:Mr[A\=INCRNNOLOOP(0;${number_of_clients};${number_of_clients_digits})]   -b   ou\=people,dc\=minsait,dc\=com   -e   add,commoncounter   -n   10    stdout=output.txt
    log    ${result.stderr}    console=True
    Should Be Equal As Integers    ${result.rc}    0

Find the last created entry
    ${result} =    Run Process    ldapsearch   -H   ldap://${ldaphost}:${ldapport}   -x   -wsecret   -D   cn\=Manager,dc\=minsait,dc\=com   -b   cn\=Mr${number_of_clients},ou\=people,dc\=minsait,dc\=com   -s   base   -LLL
    log    ${result.stderr}    console=True
    Should Be Equal As Integers    ${result.rc}    0

*** Keywords ***
Clean data
    ${result} =    Run Process    ldapdelete    -h    ${ldaphost}    -p    ${ldapport}    -x    -wsecret    -D    cn\=Manager,dc\=minsait,dc\=com    -r    dc\=minsait,dc\=com
    # log    ${result.stderr}    console=True
    ${result} =    Run Process    ldapadd       -h    ${ldaphost}    -p    ${ldapport}    -x    -wsecret    -D    cn\=Manager,dc\=minsait,dc\=com    -f    initial-data.ldif
    # log    ${result.stderr}    console=True
    Should Be Equal As Integers    ${result.rc}    0