package main

import (
	"./webhook"
	"flag"
	"github.com/sirupsen/logrus"
	"os"
)

var log = logrus.WithField("context", "main")

func main() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})

	configFileName := flag.String("config", "./config.yaml", "path to the configuration file")
	flag.Parse()

	cfg, err := webhook.ConfigFromFile(*configFileName)
	if err != nil {
		log.Fatal(err)
	}

	err = webhook.New(cfg).Start()
	if err != nil {
		log.Fatal(err)
	}

}
