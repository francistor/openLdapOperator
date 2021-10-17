#!/bin/bash

# Dependencies
# kubectl sudo snap install kubectl. Clpy kubectl file
# kustomize curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
# Golang curl -L -o go.tar.gz https://golang.org/dl/go1.17.1.linux-amd64.tar.gz && tar -C /usr/local -xzf /home/francisco/go.tar.gz && echo "export PATH=$PATH:/usr/local/go/bin" >> /etc/environment
# Docker sudo apt install docker.io
# pip sudo apt install python3-venv python3-pip

# This one will make the script to exit in case of error in any command
set -e

export OPENLDAP_IMAGE=harbor.jativa:443/francisco/openldap:latest
export CONTROLLER_IMAGE=harbor.jativa:443/francisco/openldapoperator:latest
export LOADBALANCER_IP_ADDRESS=192.168.122.210

# Build the Docker Image locally. The last parameter is the context
sudo docker build -f ../docker/dockerfile -t $OPENLDAP_IMAGE ..

# if [[ $? -ne 0 ]]; then echo "Docker build for openldap failed"; exit; fi

# Push to the registry
# Default harbor login admin:Harbor12345
# If getting unknown certificate authority, copy the CA.crt to /etc/docker/certs.d/<repo-name> and systemctl restart docker
sudo docker push $OPENLDAP_IMAGE

# Build operator
pushd ../operator
make test build
sudo docker build -t $CONTROLLER_IMAGE .
sudo docker push $CONTROLLER_IMAGE
pushd config/manager && kustomize edit set image controller=$CONTROLLER_IMAGE
popd && kustomize build config/default | kubectl apply -f -
popd

# Deploy one openldap instance
cat <<EOF | kubectl apply -f -
apiVersion: openldap.minsait.com/v1alpha1
kind: Openldap
metadata:
  name: sample 
spec:
  image: $OPENLDAP_IMAGE
  storage-size: 1Gi
  dispose-pvc: true
  loadbalancer-ip-address: $LOADBALANCER_IP_ADDRESS
  config: |
    # Configuration from Custom Resource Definition
    # This file should NOT be world readable.
    #
    include		/usr/local/etc/openldap/schema/core.schema
    include 	/usr/local/etc/openldap/schema/cosine.schema 
    include 	/usr/local/etc/openldap/schema/inetorgperson.schema 

    # Define global ACLs to disable default read access.

    # Do not enable referrals until AFTER you have a working directory
    # service AND an understanding of referrals.
    #referral	ldap://root.openldap.org

    pidfile		/usr/local/var/run/slapd.pid
    argsfile	/usr/local/var/run/slapd.args

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

    database	mdb
    maxsize		1073741824
    suffix		"dc=minsait,dc=com"
    rootdn		"cn=Manager,dc=minsait,dc=com"
    # Cleartext passwords, especially for the rootdn, should
    # be avoid.  See slappasswd(8) and slapd.conf(5) for details.
    # Use of strong authentication encouraged.
    rootpw		secretcr
    # The database directory MUST exist prior to running slapd AND 
    # should only be accessible by the slapd and slap tools.
    # Mode 700 recommended.
    directory	/usr/local/var/openldap-data
    # Indices to maintain
    index	objectClass	eq

    #######################################################################
    # monitor database definitions
    #######################################################################
    database monitor

EOF

# Test
sudo apt update

# Upgrade pip
sudo -H pip3 install --upgrade pip

# Install Robot
python3 -m pip install --ignore-installed haikunator requests pyvcloud progressbar pathlib robotframework robotframework-seleniumlibrary robotframework-requests robotframework-SSHLibrary
      
# Install ldapsearch et al
sudo apt install -y ldap-utils
      
# Install ldclt
sudo apt install -y 389-ds-base

# Wait until openldap-sample pod available
while ! kubectl get pods -n default|grep Running; do
  echo "LDAP pod not running yet"
  sleep 10; 
done

# Execute tests
pushd ../tests
robot -d output ldap.robot 

