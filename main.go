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
		return "", 0, err
	}

	var sensorData map[string]interface{}
	if e := json.Unmarshal(output, &sensorData); e != nil {
		return "", 0, e
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
		return "", 0, err
	}

	warning := ""
	if mainTemp > highTempThreshold {
		warning = "⚠️ High Temperature Warning"
	}

	var coreTemps []string
	coreTemps = append(coreTemps, fmt.Sprintf("%.1f", mainTemp))

	for key, value := range cpuData {
		if strings.HasPrefix(key, "Core") {
			coreData, exist := value.(map[string]interface{})
			if !exist {
				continue
			}
			for k, v := range coreData {
				if strings.HasSuffix(k, "_input") {
					coreTemp, e := parseJsonNumber(json.Number(fmt.Sprintf("%v", v)))
					if e == nil {
						coreTemps = append(coreTemps, fmt.Sprintf("%.1f", coreTemp))
					}
					break
				}
			}
		}
	}

	text := fmt.Sprintf("%s\nCPU: %s°C\n", warning, strings.Join(coreTemps, "°C | "))

	if acpiData, exist := sensorData["acpitz-acpi-0"].(map[string]interface{}); exist {
		if temp1, tExist := acpiData["temp1"].(map[string]interface{}); tExist {
			if acpiTemp, aExist := temp1["temp1_input"]; aExist {
				acpiTempFloat, _ := parseJsonNumber(json.Number(fmt.Sprintf("%v", acpiTemp)))
				text += fmt.Sprintf("\nACPI: %.1f°C", acpiTempFloat)
			}
		}
	}

	text += "\n"

	if nvmeData, exist := sensorData["nvme-pci-0400"].(map[string]interface{}); exist {
		if composite, cExist := nvmeData["Composite"].(map[string]interface{}); cExist {
			if nvmeTemp, nExist := composite["temp1_input"]; nExist {
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
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", conf.Token), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %s", resp.Status)
	}

	var respData struct {
		Result struct {
			MessageId int64 `json:"message_id"`
		}
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return err
	}

	err = pinMessageToTelegram(conf, respData.Result.MessageId)
	if err != nil {
		return err
	}

	log.Printf("%.1f°C", temp)
	return nil
}

func pinMessageToTelegram(conf *Config, messageId int64) error {
	// unpin all messages
	data := map[string]interface{}{
		"chat_id": conf.TargetId,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.telegram.org/bot%s/unpinAllChatMessages", conf.Token), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}

	// pin the message
	data = map[string]interface{}{
		"chat_id":              conf.TargetId,
		"message_id":           messageId,
		"disable_notification": true,
	}
	jsonData, err = json.Marshal(data)
	if err != nil {
		return err
	}
	req, err = http.NewRequest("POST", fmt.Sprintf("https://api.telegram.org/bot%s/pinChatMessage", conf.Token), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
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
