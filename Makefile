all: deps build docker
deps:
	go get github.com/blacked/go-zabbix
	go get github.com/sirupsen/logrus
build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .
docker:
	docker build . -t alertmanager-zabbix-webhook
