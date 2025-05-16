package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.gotmpl
var templates embed.FS

type ComponentType string

const (
	Receiver  ComponentType = "receiver"
	Processor ComponentType = "processor"
	Exporter  ComponentType = "exporter"
)

type Builder struct {
	WorkDir       string
	ComponentType ComponentType
	Package       string
	PackageName   string
	Output        string
}

func (b *Builder) Prepare() error {
	err := os.MkdirAll(b.WorkDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create workdir %s: %w", workDir, err)
	}

	err = b.exec("go", "mod", "init", b.PackageName)
	if err != nil {
		return fmt.Errorf("failed to init go module: %w", err)
	}

	err = b.exec("go", "get", b.Package)
	if err != nil {
		return fmt.Errorf("failed to get package %s: %w", b.Package, err)
	}

	err = b.exec("go", "mod", "edit", "-tool=github.com/otelwasm/wasibuilder")
	if err != nil {
		return fmt.Errorf("failed to add wasibuilder as tool: %w", err)
	}

	dst := filepath.Join(b.WorkDir, "main.go")
	tmplName := strings.ToLower(string(b.ComponentType)) + ".gotmpl"

	err = b.writeTemplate(dst, tmplName, map[string]any{
		"UpstreamPackage": b.Package,
	})
	if err != nil {
		return fmt.Errorf("failed to write template %s: %w", tmplName, err)
	}

	err = b.exec("go", "mod", "tidy")
	if err != nil {
		return fmt.Errorf("failed to tidy go module: %w", err)
	}

	return nil
}

func (b *Builder) Build() error {
	output, err := filepath.Abs(b.Output)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of output file %s: %w", b.Output, err)
	}

	err = b.exec("go", "tool", "wasibuilder", "go", "build", "-o", output, ".")
	if err != nil {
		return fmt.Errorf("failed to build package %s: %w", b.Package, err)
	}

	return nil
}

func (b *Builder) Clean() error {
	err := os.RemoveAll(b.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to remove workdir %s: %w", b.WorkDir, err)
	}

	return nil
}

func (b *Builder) exec(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = b.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (b *Builder) writeTemplate(dst, templateName string, data interface{}) error {
	templatePath := filepath.Join("templates", templateName)

	tmpl := template.Must(template.ParseFS(templates, templatePath))

	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, templateName, data)
	if err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	err = os.WriteFile(dst, buf.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write template %s: %w", dst, err)
	}

	return nil
}
