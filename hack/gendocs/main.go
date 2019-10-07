package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/appscode/go/runtime"
	"github.com/spf13/cobra/doc"
	"stash.appscode.dev/cli/pkg"
)

const (
	version = "v0.1.0"
)

var (
	tplFrontMatter = template.Must(template.New("index").Parse(`---
title: Stash kubectl plugin
description: tash kubectl plugin Reference
menu:
  product_stash_{{ .Version }}:
    identifier: stash-cli
    name: Stash
    parent: reference
    weight: 20
menu_name: product_stash_{{ .Version }}
---
`))

	_ = template.Must(tplFrontMatter.New("cmd").Parse(`---
title: {{ .Name }}
menu:
  product_stash_{{ .Version }}:
    identifier: {{ .ID }}
    name: {{ .Name }}
    parent: stash-cli
{{- if .RootCmd }}
    weight: 0
{{ end }}
product_name: stash
section_menu_id: reference
menu_name: product_stash_{{ .Version }}
{{- if .RootCmd }}
url: /products/stash/{{ .Version }}/reference/cli/
aliases:
  - /products/stash/{{ .Version }}/reference/cli/cli/
{{ end }}
---
`))
)

// ref: https://github.com/spf13/cobra/blob/master/doc/md_docs.md
func main() {
	rootCmd := pkg.NewRootCmd()
	dir := runtime.GOPath() + "/src/stash.appscode.dev/docs/docs/reference/cli"
	fmt.Printf("Generating cli markdown tree in: %v\n", dir)
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatalln(err)
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatalln(err)
	}

	filePrepender := func(filename string) string {
		filename = filepath.Base(filename)
		base := strings.TrimSuffix(filename, path.Ext(filename))
		name := strings.Title(strings.Replace(base, "_", " ", -1))
		parts := strings.Split(name, " ")
		if len(parts) > 1 {
			name = strings.Join(parts[1:], " ")
		}
		data := struct {
			ID      string
			Name    string
			Version string
			RootCmd bool
		}{
			strings.Replace(base, "_", "-", -1),
			name,
			version,
			!strings.ContainsRune(base, '_'),
		}
		var buf bytes.Buffer
		if err := tplFrontMatter.ExecuteTemplate(&buf, "cmd", data); err != nil {
			log.Fatalln(err)
		}
		return buf.String()
	}

	linkHandler := func(name string) string {
		return "/docs/reference/stash/" + name
	}
	err = doc.GenMarkdownTreeCustom(rootCmd, dir, filePrepender, linkHandler)
	if err != nil {
		log.Fatalln(err)
	}

	index := filepath.Join(dir, "_index.md")
	f, err := os.OpenFile(index, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	err = tplFrontMatter.ExecuteTemplate(f, "index", struct{ Version string }{version})
	if err != nil {
		log.Fatalln(err)
	}
	if err := f.Close(); err != nil {
		log.Fatalln(err)
	}
}
