/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

	"stash.appscode.dev/cli/pkg"

	"github.com/appscode/go/runtime"
	"github.com/spf13/cobra/doc"
)

var (
	tplFrontMatter = template.Must(template.New("index").Parse(`---
title: Stash kubectl plugin
description: Stash kubectl plugin Reference
menu:
  product_stash_{{ "{{.version}}" }}:
    identifier: stash-cli-references-{{ "{{ .subproject_version }}" }}
    name: {{ "{{ .subproject_version }}" }}
    parent: stash-cli-references
    weight: 20
menu_name: product_stash_{{ "{{.version}}" }}
---
`))

	_ = template.Must(tplFrontMatter.New("cmd").Parse(`---
title: {{ .Name }}
menu:
  product_stash_{{ "{{.version}}" }}:
    identifier: {{ .ID }}-{{ "{{ .subproject_version }}" }}
    name: {{ .Name }}
    parent: stash-cli-references-{{ "{{ .subproject_version }}" }}
{{- if .RootCmd }}
    weight: 0
{{- end }}
product_name: stash
section_menu_id: guides
menu_name: product_stash_{{ "{{.version}}" }}
{{- if .RootCmd }}
aliases:
  - /products/stash/{{ "{{.version}}" }}/guides/latest/cli/reference/{{ "{{ .subproject_version }}" }}
{{ end }}
---
`))
)

// ref: https://github.com/spf13/cobra/blob/master/doc/md_docs.md
func main() {
	rootCmd := pkg.NewRootCmd()
	dir := runtime.GOPath() + "/src/stash.appscode.dev/cli/docs/"
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
			RootCmd bool
		}{
			strings.Replace(base, "_", "-", -1),
			name,
			!strings.ContainsRune(base, '_'),
		}
		var buf bytes.Buffer
		if err := tplFrontMatter.ExecuteTemplate(&buf, "cmd", data); err != nil {
			log.Fatalln(err)
		}
		return buf.String()
	}

	linkHandler := func(name string) string {
		return `/docs/guides/latest/cli/reference/{{< param "info.subproject_version" >}}/` + name
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
	err = tplFrontMatter.ExecuteTemplate(f, "index", struct{ Version string }{""})
	if err != nil {
		log.Fatalln(err)
	}
	if err := f.Close(); err != nil {
		log.Fatalln(err)
	}
}
