package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Financial-Times/publish-availability-monitor/config"
	log "github.com/Sirupsen/logrus"
)

// ParseConfig opens the file at configFileName and unmarshals it into an AppConfig.
func ParseConfig(configFileName string) (*config.AppConfig, error) {
	file, err := ioutil.ReadFile(configFileName)
	if err != nil {
		log.Errorf("Error reading configuration file [%v]: [%v]", configFileName, err.Error())
		return nil, err
	}

	var conf config.AppConfig
	err = json.Unmarshal(file, &conf)
	if err != nil {
		log.Errorf("Error unmarshalling configuration file [%v]: [%v]", configFileName, err.Error())
		return nil, err
	}

	return &conf, nil
}
