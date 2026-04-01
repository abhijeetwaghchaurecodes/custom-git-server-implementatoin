package templategen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gonja "github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"service-injector/db"
)

// RenderToString renders a .gonja (Jinja2-style) template with DB records as context.
// Gonja syntax is identical to Python Jinja2:
//
//	{{ variable }}         — output variable
//	{% for x in list %}   — loop
//	{% if condition %}    — conditional
//	{{ list | length }}   — filter
func RenderToString(templatePath string, records []db.ImplementationRecord) (string, error) {
	tpl, err := gonja.FromFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("gonja load %q: %w", templatePath, err)
	}

	ctx := exec.NewContext(buildContext(records))

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("gonja execute %q: %w", templatePath, err)
	}
	return buf.String(), nil
}

// WriteToFile writes rendered content to outputPath, creating parent dirs.
func WriteToFile(outputPath, content string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// DebugContext prints a summary of what the template will receive.
func DebugContext(records []db.ImplementationRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  Template context — %d implementations:\n", len(records)))
	for _, r := range records {
		sb.WriteString(fmt.Sprintf("    alias=%-15s interface=%-15s struct=%s\n",
			r.PackageName, r.Interface, r.StructName))
	}
	return sb.String()
}

// buildContext converts DB records into the Gonja context map.
// In Jinja2 terms: template.render(**context)
func buildContext(records []db.ImplementationRecord) map[string]interface{} {
	var impls []map[string]interface{}
	for _, r := range records {
		injAt := "pending"
		if r.InjectedAt != nil {
			injAt = r.InjectedAt.Format(time.RFC3339)
		}
		impls = append(impls, map[string]interface{}{
			"alias":       r.PackageName,
			"import_path": r.ImportPath,
			"interface":   r.Interface,
			"struct_name": r.StructName,
			"status":      r.Status,
			"checksum":    r.Checksum,
			"injected_at": injAt,
		})
	}
	return map[string]interface{}{
		"generated_at":    time.Now().Format(time.RFC3339),
		"implementations": impls,
		"project_name":    "project1-mf-backend",
	}
}
