package main

import (
	"context"
	"flag"
	"fmt"
	"log"
)

var BuildVersion = "dev"

func main() {
	config := flag.String("config", "config.json", "config file path")
	help := flag.Bool("help", false, "show help")
	flag.Parse()

	if *help {
		fmt.Printf("Version: %s\n", BuildVersion)
		flag.Usage()
		return
	}
	conf, err := loadConfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	app, err := NewApplication(conf)
	if err != nil {
		log.Fatal(err)
	}
	//app.tempLoader = MockLoadSensorsTemperature("mock_data.json")
	go app.startPolling(context.Background())
	app.startMonitoring(context.Background())
}
