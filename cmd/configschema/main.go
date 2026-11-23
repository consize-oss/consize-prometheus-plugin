// configschema generates the public JSON Schema from the same types the plugin decodes.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/consize-oss/consize-prometheus-plugin/prometheus"
)

func main() {
	check := flag.Bool("check", false, "check generated schema is current")
	flag.Parse()
	schema := schemaFor(reflect.TypeOf(prometheus.JSONConfig{}))
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	schema["$id"] = "https://raw.githubusercontent.com/consize-oss/consize-prometheus-plugin/main/config.schema.json"
	schema["title"] = "Consize Prometheus plugin configuration"
	schema["required"] = []string{"base_url"}
	properties := schema["properties"].(map[string]any)
	properties["base_url"].(map[string]any)["format"] = "uri"
	properties["window"].(map[string]any)["default"] = "24h"
	properties["step"].(map[string]any)["default"] = "5m"
	schema["description"] = "Strict subprocess configuration. Duration, endpoint and mapping constraints are also validated by the plugin. File credentials only; the host strips credential environment variables."
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *check {
		current, err := os.ReadFile("config.schema.json")
		if err != nil || !bytes.Equal(current, data) {
			fmt.Fprintln(os.Stderr, "config.schema.json is stale; run go run ./cmd/configschema")
			os.Exit(1)
		}
	} else if err := os.WriteFile("config.schema.json", data, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func schemaFor(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.Struct:
		properties := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" || strings.HasSuffix(name, "_env") {
				continue
			}
			properties[name] = schemaFor(field.Type)
		}
		return map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(t.Elem())}
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer", "minimum": 0}
	case reflect.String:
		return map[string]any{"type": "string"}
	default:
		panic("unsupported configuration type: " + t.String())
	}
}
