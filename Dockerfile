FROM alpine:latest

COPY alertmanager-zabbix-webhook  /usr/bin
RUN chmod +x /usr/bin/alertmanager-zabbix-webhook

RUN mkdir -p /etc/webhook
COPY config.yaml /etc/webhook

RUN adduser webhook -s /bin/false -D webhook
USER webhook

CMD ["/usr/bin/alertmanager-zabbix-webhook", "-config", "/etc/webhook/config.yaml"]
