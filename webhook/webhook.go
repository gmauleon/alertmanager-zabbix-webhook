package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	zabbix "github.com/blacked/go-zabbix"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var log = logrus.WithField("context", "webhook")

type WebHook struct {
	channel chan *Alert
	config  WebHookConfig
}

type WebHookConfig struct {
	Port                 int    `yaml:"port"`
	QueueCapacity        int    `yaml:"queueCapacity"`
	ZabbixServerHost     string `yaml:"zabbixServerHost"`
	ZabbixServerPort     int    `yaml:"zabbixServerPort"`
	ZabbixHostDefault    string `yaml:"zabbixHostDefault"`
	ZabbixHostAnnotation string `yaml:"zabbixHostAnnotation"`
	ZabbixKeyPrefix      string `yaml:"zabbixKeyPrefix"`
}

type HookRequest struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt,omitempty"`
	EndsAt       string            `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL"`
}

func New(cfg *WebHookConfig) *WebHook {

	return &WebHook{
		channel: make(chan *Alert, cfg.QueueCapacity),
		config:  *cfg,
	}
}

func ConfigFromFile(filename string) (cfg *WebHookConfig, err error) {
	log.Infof("Loading configuration at '%s'", filename)
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open the config file: %s", err)
	}

	// Default values
	config := WebHookConfig{
		Port:                 8080,
		QueueCapacity:        500,
		ZabbixServerHost:     "127.0.0.1",
		ZabbixServerPort:     10051,
		ZabbixHostAnnotation: "zabbix_host",
		ZabbixKeyPrefix:      "prometheus",
		ZabbixHostDefault:    "",
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("can't read the config file: %s", err)
	}

	log.Info("Configuration loaded")
	return &config, nil
}

func (hook *WebHook) Start() error {

	// Launch the process thread
	go hook.processAlerts()

	// Launch the listening thread
	log.Println("Initializing HTTP server")
	http.HandleFunc("/alerts", hook.alertsHandler)
	err := http.ListenAndServe(":"+strconv.Itoa(hook.config.Port), nil)
	if err != nil {
		return fmt.Errorf("can't start the listening thread: %s", err)
	}

	log.Info("Exiting")
	close(hook.channel)

	return nil
}

func (hook *WebHook) alertsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		hook.postHandler(w, r)
	default:
		http.Error(w, "unsupported HTTP method", 400)
	}
}

func (hook *WebHook) postHandler(w http.ResponseWriter, r *http.Request) {

	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var m HookRequest
	if err := dec.Decode(&m); err != nil {
		log.Errorf("error decoding message: %v", err)
		http.Error(w, "request body is not valid json", 400)
		return
	}

	for index := range m.Alerts {
		hook.channel <- &m.Alerts[index]
	}

}

func (hook *WebHook) processAlerts() {

	log.Info("Alerts queue started")

	// While there are alerts in the queue, batch them and send them over to Zabbix
	var metrics []*zabbix.Metric
	for {
		select {
		case a := <-hook.channel:
			if a == nil {
				log.Info("Queue Closed")
				return
			}

			host, exists := a.Annotations[hook.config.ZabbixHostAnnotation]
			if !exists {
				host = hook.config.ZabbixHostDefault
			}

			// Send alerts only if a host annotation is present or configuration is not nill
			if host != "" || hook.config.ZabbixHostDefault != "" {
				key := fmt.Sprintf("%s.%s", hook.config.ZabbixKeyPrefix, strings.ToLower(a.Labels["alertname"]))
				value := "0"
				if a.Status == "firing" {
					value = "1"
				}

				log.Infof("added Zabbix metrics, host: '%s' key: '%s', value: '%s'", host, key, value)
				metrics = append(metrics, zabbix.NewMetric(host, key, value))
			}
		default:
			if len(metrics) != 0 {
				hook.zabbixSend(metrics)
				metrics = metrics[:0]
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (hook *WebHook) zabbixSend(metrics []*zabbix.Metric) {
	// Create instance of Packet class
	packet := zabbix.NewPacket(metrics)

	// Send packet to zabbix
	log.Infof("sending to zabbix '%s:%d'", hook.config.ZabbixServerHost, hook.config.ZabbixServerPort)
	z := zabbix.NewSender(hook.config.ZabbixServerHost, hook.config.ZabbixServerPort)
	_, err := z.Send(packet)
	if err != nil {
		log.Error(err)
	} else {
		log.Info("successfully sent")
	}

}
