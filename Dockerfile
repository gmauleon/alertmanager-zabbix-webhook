FROM alpine:latest

RUN adduser webhook -s /bin/false -D webhook

RUN mkdir -p /etc/webhook
COPY config.yaml /etc/webhook

COPY alertmanager-zabbix-webhook  /usr/bin
RUN chmod +x /usr/bin/alertmanager-zabbix-webhook

USER webhook

ENTRYPOINT ["/usr/bin/alertmanager-zabbix-webhook"]
CMD ["-config", "/etc/webhook/config.yaml"]
