package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Token    string `json:"token"`
	TargetId int64  `json:"target_id"`
}

func loadConfig(path string) (*Config, error) {
	if strings.HasPrefix(path, "http") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		config := &Config{}
		err = json.NewDecoder(resp.Body).Decode(config)
		if err != nil {
			return nil, err
		}
		return config, nil
	} else {
		file, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		config := &Config{}
		err = json.Unmarshal(file, config)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}
