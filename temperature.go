package main

import (
	"encoding/json"
	"fmt"
	"github.com/alexeyco/simpletable"
	"github.com/tidwall/gjson"
	"os"
	"os/exec"
	"strings"
	"time"
)

type TemperatureData struct {
	Name      string
	Input     json.Number
	Max       json.Number
	Min       json.Number
	Crit      json.Number
	CritAlarm json.Number
	Alarm     json.Number
}

type Module struct {
	Name    string            `json:"-"`
	Adapter string            `json:"Adapter"`
	Data    []TemperatureData `json:"-"`
}

type SensorsTemperature struct {
	Modules []Module `json:"-"`
}

type TemperatureLoader func() (*SensorsTemperature, error)

func LoadSensorsTemperature() (*SensorsTemperature, error) {
	output, err := exec.Command("sensors", "-j").Output()
	if err != nil {
		return nil, err
	}
	var temp SensorsTemperature
	err = json.Unmarshal(output, &temp)
	if err != nil {
		return nil, err
	}
	return &temp, nil
}

func MockLoadSensorsTemperature(path string) TemperatureLoader {
	return func() (*SensorsTemperature, error) {
		file, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var temp SensorsTemperature
		err = json.Unmarshal(file, &temp)
		if err != nil {
			return nil, err
		}
		return &temp, nil
	}
}

func RenderLogMessage(s *SensorsTemperature) string {
	return fmt.Sprintf("Temperature: %s", s.HighestTemperature())
}

func RenderTableMessage(s *SensorsTemperature) string {
	table := simpletable.New()
	table.SetStyle(simpletable.StyleCompactLite)
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Text: "Type"},
			{Text: "Index"},
			{Text: "Temp"},
		},
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: Temperature: <strong>%s</strong>\n\n", time.Now().Format(time.DateTime), s.HighestTemperature()))
	for _, module := range s.Modules {
		for _, data := range module.Data {
			temp, err := data.Input.Float64()
			if err != nil {
				continue
			}
			tempStr := fmt.Sprintf("%.2f°C", temp)
			row := []*simpletable.Cell{
				{Text: module.Name},
				{Text: data.Name},
				{Text: tempStr},
			}
			table.Body.Cells = append(table.Body.Cells, row)
		}
	}
	sb.WriteString("<pre>")
	sb.WriteString(table.String())
	sb.WriteString("</pre>")
	return sb.String()
}

func (s *SensorsTemperature) HighestTemperature() string {
	var maxTemp float64 = -1000
	for _, module := range s.Modules {
		for _, data := range module.Data {
			t, err := data.Input.Float64()
			if err != nil {
				continue
			}
			if t > maxTemp {
				maxTemp = t
			}
		}
	}
	if maxTemp < -999 {
		return "N/A"
	}
	return fmt.Sprintf("%.2f°C", maxTemp)
}

func (s *SensorsTemperature) IsHigherThanThreshold(threshold float64) bool {
	for _, module := range s.Modules {
		for _, data := range module.Data {
			if t, err := data.Input.Float64(); err == nil && t >= threshold {
				return true
			}
		}
	}
	return false
}

func (s *SensorsTemperature) UnmarshalJSON(data []byte) error {
	result := gjson.ParseBytes(data)
	result.ForEach(func(key, value gjson.Result) bool {
		temp := Module{
			Name: key.String(),
		}
		if err := json.Unmarshal([]byte(value.Raw), &temp); err != nil {
			return false
		}
		s.Modules = append(s.Modules, temp)
		return true
	})
	return nil
}

func (m *Module) UnmarshalJSON(data []byte) error {
	m.Adapter = ""
	m.Data = nil
	result := gjson.ParseBytes(data)
	result.ForEach(func(key, value gjson.Result) bool {
		if key.String() == "Adapter" {
			m.Adapter = value.String()
			return true
		}
		if !value.IsObject() {
			return true
		}
		tempData := &TemperatureData{
			Name: key.String(),
		}
		value.ForEach(func(k, v gjson.Result) bool {
			if strings.HasSuffix(k.String(), "_input") {
				tempData.Input = json.Number(v.Raw)
			} else if strings.HasSuffix(k.String(), "_max") {
				tempData.Max = json.Number(v.Raw)
			} else if strings.HasSuffix(k.String(), "_min") {
				tempData.Min = json.Number(v.Raw)
			} else if strings.HasSuffix(k.String(), "_crit") {
				tempData.Crit = json.Number(v.Raw)
			} else if strings.HasSuffix(k.String(), "_crit_alarm") {
				tempData.CritAlarm = json.Number(v.Raw)
			} else if strings.HasSuffix(k.String(), "_alarm") {
				tempData.Alarm = json.Number(v.Raw)
			}
			return true
		})
		m.Data = append(m.Data, *tempData)
		return true
	})
	return nil
}
