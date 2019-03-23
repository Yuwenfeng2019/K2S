#!/bin/bash
set -e -x

while [ ! -e /etc/k2s/k2s.yaml ]; do
    echo waiting for config
    sleep 1
done

mkdir -p /root/.kube
sed 's/localhost/server/g' /etc/k2s/k2s.yaml > /root/.kube/config
export KUBECONFIG=/root/.kube/config
cat /etc/k2s/k2s.yaml
cat $KUBECONFIG
sonobuoy run
sleep 15
sonobuoy logs -f
