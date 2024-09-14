package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var BuildVersion = "dev"

const (
	highTempThreshold = 50.0
	sleepDuration     = 10 * time.Minute
)

func parseJsonNumber(n json.Number) (float64, error) {
	if n == "" {
		return 0, nil
	}
	return strconv.ParseFloat(string(n), 64)
}

func renderMessage() (string, float64, error) {
	output, err := exec.Command("sensors", "-j").Output()
	if err != nil {
		return "", 0, fmt.Errorf("failed to run sensors: %w", err)
	}

	var sensorData map[string]interface{}
	if err := json.Unmarshal(output, &sensorData); err != nil {
		return "", 0, fmt.Errorf("failed to parse output: %w", err)
	}

	cpuData, ok := sensorData["coretemp-isa-0000"].(map[string]interface{})
	if !ok {
		return "", 0, fmt.Errorf("failed to find CPU data")
	}

	packageData, ok := cpuData["Package id 0"].(map[string]interface{})
	if !ok {
		return "", 0, fmt.Errorf("failed to find Package id 0 data")
	}

	mainTemp, err := parseJsonNumber(json.Number(fmt.Sprintf("%v", packageData["temp1_input"])))
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse main temperature: %w", err)
	}

	warning := ""
	if mainTemp > highTempThreshold {
		warning = "⚠️ High Temperature Warning"
	}

	var coreTemps []string
	coreTemps = append(coreTemps, fmt.Sprintf("%.1f", mainTemp))

	for key, value := range cpuData {
		if strings.HasPrefix(key, "Core") {
			coreData, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			for k, v := range coreData {
				if strings.HasSuffix(k, "_input") {
					coreTemp, err := parseJsonNumber(json.Number(fmt.Sprintf("%v", v)))
					if err == nil {
						coreTemps = append(coreTemps, fmt.Sprintf("%.1f", coreTemp))
					}
					break
				}
			}
		}
	}

	text := fmt.Sprintf("%s\nCPU: %s°C\n", warning, strings.Join(coreTemps, "°C | "))

	if acpiData, ok := sensorData["acpitz-acpi-0"].(map[string]interface{}); ok {
		if temp1, ok := acpiData["temp1"].(map[string]interface{}); ok {
			if acpiTemp, ok := temp1["temp1_input"]; ok {
				acpiTempFloat, _ := parseJsonNumber(json.Number(fmt.Sprintf("%v", acpiTemp)))
				text += fmt.Sprintf("\nACPI: %.1f°C", acpiTempFloat)
			}
		}
	}

	text += "\n"

	if nvmeData, ok := sensorData["nvme-pci-0400"].(map[string]interface{}); ok {
		if composite, ok := nvmeData["Composite"].(map[string]interface{}); ok {
			if nvmeTemp, ok := composite["temp1_input"]; ok {
				nvmeTempFloat, _ := parseJsonNumber(json.Number(fmt.Sprintf("%v", nvmeTemp)))
				text += fmt.Sprintf("\nNVME: %.1f°C", nvmeTempFloat)
			}
		}
	}

	return text, mainTemp, nil
}

func sendPVEStatusToTelegram(text string, temp float64, conf *Config) error {
	data := map[string]interface{}{
		"chat_id":              conf.TargetId,
		"text":                 text,
		"disable_notification": temp <= 50,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize JSON data: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", conf.Token), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %s", resp.Status)
	}

	log.Println("sendPVEStatusToTelegram: true")
	return nil
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
		bytes, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		config := &Config{}
		err = json.Unmarshal(bytes, config)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}

type Config struct {
	Token    string `json:"token"`
	TargetId int64  `json:"target_id"`
}

func main() {
	conf := flag.String("config", "config.json", "config file path")
	help := flag.Bool("help", false, "show help")
	flag.Parse()

	if *help {
		fmt.Printf("Version: %s\n", BuildVersion)
		flag.Usage()
		return
	}

	config, err := loadConfig(*conf)
	if err != nil {
		log.Fatal(err)
	}
	for {
		text, temp, e := renderMessage()
		if e != nil {
			log.Printf("Error rendering message: %v", e)
			continue
		}
		if se := sendPVEStatusToTelegram(text, temp, config); se != nil {
			log.Printf("Error sending message to Telegram: %v", se)
		}
		time.Sleep(sleepDuration)
	}
}
