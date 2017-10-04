all: go-deps go-build docker-build
go-deps:
	go get -t ./...
go-build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .
docker-build:
	docker build . -t alertmanager-zabbix-webhook
