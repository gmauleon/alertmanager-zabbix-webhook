all: go docker

go:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .
docker:
	docker build . -t alertmanager-zabbix-webhook
