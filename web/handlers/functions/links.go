package functions

import (
	"strings"
	"text/template"
)

func TraceURLFunc(eventTraceURLTemplate *template.Template) func(string) string {
	return func(traceID string) string {
		return traceIDToTraceURL(traceID, eventTraceURLTemplate)
	}
}

func traceIDToTraceURL(traceID string, eventTraceURLTemplate *template.Template) string {
	if eventTraceURLTemplate == nil {
		return ""
	}
	if traceID == "" {
		return ""
	}

	sb := new(strings.Builder)
	err := eventTraceURLTemplate.Execute(sb, map[string]string{
		"TraceID": traceID,
	})
	if err != nil {
		return err.Error()
	}
	return sb.String()
}
