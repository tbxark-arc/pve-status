package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/go-sphere/confstore"
	"github.com/go-sphere/confstore/codec"
	"github.com/go-sphere/confstore/provider"
	"github.com/go-sphere/confstore/provider/file"
	"github.com/go-sphere/confstore/provider/http"
)

var BuildVersion = "dev"

func main() {
	conf := flag.String("config", "config.json", "config file path")
	help := flag.Bool("help", false, "show help")
	flag.Parse()

	if *help {
		fmt.Printf("Version: %s\n", BuildVersion)
		flag.Usage()
		return
	}
	config, err := confstore.Load[Config](provider.NewSelect(*conf,
		provider.If(file.IsLocalPath, func(s string) provider.Provider {
			return file.New(s)
		}),
		provider.If(http.IsRemoteURL, func(s string) provider.Provider {
			return http.New(s, http.WithTimeout(10))
		}),
	), codec.JsonCodec())
	if err != nil {
		log.Fatal(err)
	}
	app, err := NewApplication(config)
	if err != nil {
		log.Fatal(err)
	}
	//app.tempLoader = MockLoadSensorsTemperature("mock_data.json")
	go app.startPolling(context.Background())
	app.startMonitoring(context.Background())
}
