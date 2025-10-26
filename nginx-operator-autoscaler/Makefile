IMG ?= rammurthymalisetti/nginx-operator-autoscaler:latest

.PHONY: build
build:
	go build -o bin/manager ./cmd/manager

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push:
	docker push $(IMG)

.PHONY: deploy
deploy:
	kubectl apply -f config/crd/nginxautoscalers.autoscaler.malisetti.dev.yaml
	kubectl apply -f config/rbac/rbac.yaml
	kubectl apply -f config/manager/deployment.yaml

.PHONY: undeploy
undeploy:
	-kubectl delete -f config/manager/deployment.yaml
	-kubectl delete -f config/rbac/rbac.yaml
	-kubectl delete -f config/crd/nginxautoscalers.autoscaler.malisetti.dev.yaml
