package main

import (
	"encoding/json"
	"fmt"
	"github.com/alexeyco/simpletable"
	"github.com/tidwall/gjson"
	"os/exec"
	"strings"
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

func RenderLogMessage(s *SensorsTemperature) string {
	if len(s.Modules) > 0 && len(s.Modules[0].Data) > 0 {
		return fmt.Sprintf("Temperature: %s", s.Modules[0].Data[0].Input)
	}
	return "Temperature: N/A"
}

func RenderPlainMessage(s *SensorsTemperature) string {
	table := simpletable.New()
	table.SetStyle(simpletable.StyleCompactClassic)
	for _, module := range s.Modules {
		var text strings.Builder
		for j, data := range module.Data {
			if j > 0 {
				text.WriteString(" | ")
			}
			if t, err := data.Input.Float64(); err == nil {
				text.WriteString(fmt.Sprintf("%.1f°C", t))
			} else {
				text.WriteString("N/A")
			}
		}
		table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
			{Text: strings.ToUpper(strings.Split(module.Name, "-")[0])},
			{Text: text.String()},
		})
	}
	return table.String()
}

func RenderTableMessage(s *SensorsTemperature) string {
	var text strings.Builder
	for _, module := range s.Modules {
		table := simpletable.New()
		table.SetStyle(simpletable.StyleCompactLite)
		table.Header = &simpletable.Header{
			Cells: []*simpletable.Cell{
				{Text: strings.ToUpper(strings.Split(module.Name, "-")[0])},
				{Text: "Temp"},
				{Text: "Max"},
				{Text: "Min"},
			},
		}
		for _, data := range module.Data {
			row := []*simpletable.Cell{
				{Text: data.Name},
				{Text: data.Input.String()},
				{Text: data.Max.String()},
				{Text: data.Min.String()},
			}
			table.Body.Cells = append(table.Body.Cells, row)
		}
		text.WriteString("\n")
		text.WriteString(table.String())
		text.WriteString("\n\n")
	}
	return text.String()
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
