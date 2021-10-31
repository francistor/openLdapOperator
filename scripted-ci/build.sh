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
envsubst < cr-ldap-sample.yaml | kubectl apply -f -

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

