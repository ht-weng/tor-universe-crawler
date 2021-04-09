#!/bin/bash

helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add elastic https://helm.elastic.co

kubectl create namespace TorUniverseCrawler

helm install --namespace TorUniverseCrawler redis -f deployments/k8s/helm/redis-values.yaml bitnami/redis
helm install --namespace TorUniverseCrawler rabbitmq -f deployments/k8s/helm/rabbitmq-values.yaml bitnami/rabbitmq
helm install --namespace TorUniverseCrawler elasticsearch elastic/elasticsearch
helm install --namespace TorUniverseCrawler kibana elastic/kibana

# Install our resources
kubectl -n TorUniverseCrawler apply -f deployments/k8s/torproxy.yaml
kubectl -n TorUniverseCrawler apply -f deployments/k8s/configapi.yaml
kubectl -n TorUniverseCrawler apply -f deployments/k8s/crawler.yaml
kubectl -n TorUniverseCrawler apply -f deployments/k8s/scheduler.yaml
kubectl -n TorUniverseCrawler apply -f deployments/k8s/blacklister.yaml
kubectl -n TorUniverseCrawler apply -f deployments/k8s/indexer-es.yaml
