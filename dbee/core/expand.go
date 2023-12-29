package core

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

func loadEnv() map[string]string {
	envMap := make(map[string]string)

	for _, v := range os.Environ() {
		spl := strings.Split(v, "=")
		envMap[spl[0]] = spl[1]
	}

	return envMap
}

func expand(value string) string {
	tmpl, err := template.New("expand_variables").Parse(value)
	if err != nil {
		return value
	}

	input := struct {
		Env map[string]string
	}{
		Env: loadEnv(),
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, input)
	if err != nil {
		return value
	}

	return out.String()
}
