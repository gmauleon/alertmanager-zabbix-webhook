all: deps build docker
deps:
	go get -t ./...
build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .
docker:
	docker build . -t alertmanager-zabbix-webhook
