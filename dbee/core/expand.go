package core

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func expand(value string) (string, error) {
	tmpl, err := template.New("expand_variables").
		Funcs(template.FuncMap{
			"env":  os.Getenv,
			"exec": execCommand,
		}).
		Parse(value)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, nil)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func execCommand(line string) (string, error) {
	if strings.Contains(line, " | ") {
		out, err := exec.Command("sh", "-c", line).Output()
		return strings.TrimSpace(string(out)), err
	}

	l := strings.Split(line, " ")
	if len(l) < 1 {
		return "", errors.New("no command provided")
	}
	cmd := l[0]
	args := l[1:]

	out, err := exec.Command(cmd, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// expandOrDefault silently suppresses errors.
func expandOrDefault(value string) string {
	ex, err := expand(value)
	if err != nil {
		return value
	}
	return ex
}
