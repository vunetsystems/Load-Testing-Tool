package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type ModuleEPS struct {
	Name     string
	Unique   int
	Calc     string
	EPSValue float64
}

func getPeriodSeconds(period string) int {
	period = strings.TrimSpace(period)
	if strings.HasSuffix(period, "s") {
		p := strings.TrimSuffix(period, "s")
		if seconds, err := strconv.Atoi(p); err == nil && seconds > 0 {
			return seconds
		}
	}
	return 10
}

func readYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func getNumUniqKey(conf map[string]interface{}) int {
	if uk, ok := conf["uniquekey"].(map[interface{}]interface{}); ok {
		switch n := uk["NumUniqKey"].(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		case string:
			if val, err := strconv.Atoi(n); err == nil {
				return val
			}
		}
	}
	return 1
}

// Open file in default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", "", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}

func main() {
	tmpl := template.Must(template.New("dashboard").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SQL Server EPS Dashboard</title>
<style>
body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background:#f5f5f5; padding:20px; }
h1 { text-align:center; color:#333; }
table { width:100%; border-collapse:collapse; margin-top:20px; box-shadow:0 0 10px rgba(0,0,0,0.1);}
th, td { padding:12px; text-align:left; border-bottom:1px solid #ddd; }
th { background:#007BFF; color:white; }
tr:nth-child(even) { background:#e9ecef; }
tr:hover { background:#d1ecf1; }
tfoot td { font-weight:bold; background:#28a745; color:white; }
input { margin:5px; }
button { padding:10px; background:#007BFF; color:white; border:none; cursor:pointer; }
.form-section { background:#fff; padding:15px; margin:10px 0; border-radius:5px; box-shadow:0 2px 5px rgba(0,0,0,0.1); }
</style>
</head>
<body>
<h1>SQL Server EPS Dashboard</h1>

<div class="form-section">
<h2>Unique Key Configuration</h2>
<form id="uniqueKeyForm">
<label>Name: <input type="text" id="uniqueKeyName" value="{{.UniqueKeyName}}"></label><br>
<label>Data Type: <input type="text" id="uniqueKeyDataType" value="{{.UniqueKeyDataType}}"></label><br>
<label>Value Type: <input type="text" id="uniqueKeyValueType" value="{{.UniqueKeyValueType}}"></label><br>
<label>Value: <input type="text" id="uniqueKeyValue" value="{{.UniqueKeyValue}}"></label><br>
<label>Num Unique Key: <input type="number" id="uniqueKeyNum" value="{{.UniqueKeyNum}}"></label><br>
<button type="button" onclick="saveUniqueKey()">Save Unique Key Settings</button>
</form>
</div>

<div class="form-section">
<h2>Kafka Output Details</h2>
<form id="kafkaForm">
<label>Enabled: <input type="checkbox" id="kafkaEnabled" {{if .KafkaEnabled}}checked{{end}}></label><br>
<label>Topic: <input type="text" id="kafkaTopic" value="{{.KafkaTopic}}"></label><br>
<button type="button" onclick="saveKafka()">Save Kafka Settings</button>
</form>
</div>

<table>
<thead>
<tr><th>Module</th><th>UniqueKey</th><th>EPS Calculation</th><th>EPS Value</th></tr>
</thead>
<tbody>
{{range .Modules}}
<tr><td>{{.Name}}</td><td>{{.Unique}}</td><td>{{.Calc}}</td><td>{{.EPSValue}}</td></tr>
{{end}}
</tbody>
<tfoot>
<tr><td>TOTAL</td><td></td><td></td><td>{{.TotalEPS}}</td></tr>
</tfoot>
</table>
<script>
function saveUniqueKey() {
	   const name = document.getElementById('uniqueKeyName').value;
	   const dataType = document.getElementById('uniqueKeyDataType').value;
	   const valueType = document.getElementById('uniqueKeyValueType').value;
	   const value = document.getElementById('uniqueKeyValue').value;
	   const num = parseInt(document.getElementById('uniqueKeyNum').value);
	   fetch('/update', {
	       method: 'POST',
	       headers: {'Content-Type': 'application/json'},
	       body: JSON.stringify({
	           uniqueKeyName: name,
	           uniqueKeyDataType: dataType,
	           uniqueKeyValueType: valueType,
	           uniqueKeyValue: value,
	           uniqueKeyNum: num
	       })
	   }).then(() => location.reload());
}

function saveKafka() {
	   const enabled = document.getElementById('kafkaEnabled').checked;
	   const topic = document.getElementById('kafkaTopic').value;
	   console.log('Saving Kafka:', {enabled, topic});
	   fetch('/update', {
	       method: 'POST',
	       headers: {'Content-Type': 'application/json'},
	       body: JSON.stringify({kafkaEnabled: enabled, kafkaTopic: topic})
	   }).then(response => {
	       console.log('Response status:', response.status);
	       if (response.ok) {
	           location.reload();
	       } else {
	           alert('Error saving Kafka settings');
	       }
	   }).catch(error => {
	       console.error('Error:', error);
	       alert('Error saving Kafka settings');
	   });
}
</script>
</body>
</html>`))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		confPath := "conf.yml"
		modDir := "./"
		mainConf, err := readYAML(confPath)
		if err != nil {
			http.Error(w, "Error reading main config: "+err.Error(), 500)
			return
		}

		// Extract Kafka details
		var kafkaEnabled bool
		var kafkaTopic string
		if kafka, ok := mainConf["output.kafka"].(map[interface{}]interface{}); ok {
			log.Printf("Found Kafka config: %+v", kafka)
			if enabled, ok := kafka["enabled"].(bool); ok {
				kafkaEnabled = enabled
			}
			if topic, ok := kafka["topic"].(string); ok {
				kafkaTopic = topic
			}
		} else {
			log.Printf("No Kafka config found in mainConf")
		}
		log.Printf("Extracted Kafka - Enabled: %v, Topic: %s", kafkaEnabled, kafkaTopic)

		// Extract Unique Key details
		var uniqueKeyName, uniqueKeyDataType, uniqueKeyValueType, uniqueKeyValue string
		var uniqueKeyNum int
		if uk, ok := mainConf["uniquekey"].(map[interface{}]interface{}); ok {
			if name, ok := uk["name"].(string); ok {
				uniqueKeyName = name
			}
			if dataType, ok := uk["DataType"].(string); ok {
				uniqueKeyDataType = dataType
			}
			if valueType, ok := uk["ValueType"].(string); ok {
				uniqueKeyValueType = valueType
			}
			if value, ok := uk["Value"].(string); ok {
				uniqueKeyValue = value
			}
			switch n := uk["NumUniqKey"].(type) {
			case int:
				uniqueKeyNum = n
			case int64:
				uniqueKeyNum = int(n)
			case float64:
				uniqueKeyNum = int(n)
			case string:
				if val, err := strconv.Atoi(n); err == nil {
					uniqueKeyNum = val
				}
			}
		}

		mainKeys := getNumUniqKey(mainConf)
		period := "10s"
		if p, ok := mainConf["period"].(string); ok {
			period = p
		}
		periodSec := getPeriodSeconds(period)

		submodules := []string{}
		if sm, ok := mainConf["Include_sub_modules"].([]interface{}); ok {
			for _, s := range sm {
				submodules = append(submodules, fmt.Sprintf("%v", s))
			}
		} else if smStr, ok := mainConf["Include_sub_modules"].(string); ok {
			smStr = strings.Trim(smStr, "[]")
			for _, s := range strings.Split(smStr, ",") {
				submodules = append(submodules, strings.TrimSpace(s))
			}
		}

		var modules []ModuleEPS
		totalEPS := 0.0

		for _, sub := range submodules {
			subConfPath := fmt.Sprintf("%s/%s.yml", modDir, sub)
			subConf, err := readYAML(subConfPath)
			if err != nil {
				modules = append(modules, ModuleEPS{sub, 0, "Error", 0})
				continue
			}
			subKeys := getNumUniqKey(subConf)
			eps := float64(mainKeys*subKeys) / float64(periodSec)
			totalEPS += eps
			calc := fmt.Sprintf("%d × %d / %d", mainKeys, subKeys, periodSec)
			modules = append(modules, ModuleEPS{sub, subKeys, calc, eps})
		}

		data := struct {
			Modules            []ModuleEPS
			TotalEPS           float64
			KafkaEnabled       bool
			KafkaTopic         string
			UniqueKeyName      string
			UniqueKeyDataType  string
			UniqueKeyValueType string
			UniqueKeyValue     string
			UniqueKeyNum       int
		}{
			Modules:            modules,
			TotalEPS:           totalEPS,
			KafkaEnabled:       kafkaEnabled,
			KafkaTopic:         kafkaTopic,
			UniqueKeyName:      uniqueKeyName,
			UniqueKeyDataType:  uniqueKeyDataType,
			UniqueKeyValueType: uniqueKeyValueType,
			UniqueKeyValue:     uniqueKeyValue,
			UniqueKeyNum:       uniqueKeyNum,
		}
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		var update struct {
			KafkaEnabled       bool   `json:"kafkaEnabled"`
			KafkaTopic         string `json:"kafkaTopic"`
			UniqueKeyName      string `json:"uniqueKeyName"`
			UniqueKeyDataType  string `json:"uniqueKeyDataType"`
			UniqueKeyValueType string `json:"uniqueKeyValueType"`
			UniqueKeyValue     string `json:"uniqueKeyValue"`
			UniqueKeyNum       int    `json:"uniqueKeyNum"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, err.Error(), 400)
			return
		}
		log.Printf("Received update: %+v", update)
		confPath := "conf.yml"
		mainConf, err := readYAML(confPath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Update mainConf
		if kafka, ok := mainConf["output.kafka"].(map[interface{}]interface{}); ok {
			kafka["enabled"] = update.KafkaEnabled
			kafka["topic"] = update.KafkaTopic
		} else {
			mainConf["output.kafka"] = map[interface{}]interface{}{
				"enabled": update.KafkaEnabled,
				"topic":   update.KafkaTopic,
			}
		}

		// Update unique key settings
		if uk, ok := mainConf["uniquekey"].(map[interface{}]interface{}); ok {
			if update.UniqueKeyName != "" {
				uk["name"] = update.UniqueKeyName
			}
			if update.UniqueKeyDataType != "" {
				uk["DataType"] = update.UniqueKeyDataType
			}
			if update.UniqueKeyValueType != "" {
				uk["ValueType"] = update.UniqueKeyValueType
			}
			if update.UniqueKeyValue != "" {
				uk["Value"] = update.UniqueKeyValue
			}
			if update.UniqueKeyNum > 0 {
				uk["NumUniqKey"] = update.UniqueKeyNum
			}
		} else {
			mainConf["uniquekey"] = map[interface{}]interface{}{
				"name":       update.UniqueKeyName,
				"DataType":   update.UniqueKeyDataType,
				"ValueType":  update.UniqueKeyValueType,
				"Value":      update.UniqueKeyValue,
				"NumUniqKey": update.UniqueKeyNum,
			}
		}
		// Write back to YAML
		data, err := yaml.Marshal(mainConf)
		if err != nil {
			log.Printf("Error marshaling YAML: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		if err := os.WriteFile(confPath, data, 0644); err != nil {
			log.Printf("Error writing YAML file: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		log.Printf("Successfully updated YAML file")
		w.WriteHeader(200)
	})

	log.Println("Server starting on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
