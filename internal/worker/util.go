package worker

import "github.com/prometheus/alertmanager/template"

func FindFirstValue(alert template.Alert, defaultValue string, keys ...string) string {
	for _, key := range keys {
		if val, ok := alert.Annotations[key]; ok && val != "" {
			return val
		}
		if val, ok := alert.Labels[key]; ok && val != "" {
			return val
		}
	}
	return defaultValue
}
