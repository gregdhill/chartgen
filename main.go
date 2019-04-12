package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"regexp"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	flags "github.com/jessevdk/go-flags"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/ignore"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

var opts struct {
	Args struct {
		Name string `description:"Name of the chart to generate."`
	} `positional-args:"yes" required:"yes"`
	Suite bool `short:"s" long:"suite" description:"This is a suite."`
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	meta := chart.Metadata{
		Name: opts.Args.Name,
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	err = CreateFrom(&meta, "./src", dir)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Created: %s", filepath.Join(dir, opts.Args.Name))
}

// LoadBox overrides https://godoc.org/k8s.io/helm/pkg/chartutil#Load
func LoadBox(dir string) (*chart.Chart, error) {
	box := packr.NewBox(dir)

	c := &chart.Chart{}
	rules := ignore.Empty()
	ifile := filepath.Join(dir, ignore.HelmIgnore)
	if _, err := os.Stat(ifile); err == nil {
		r, err := ignore.ParseFile(ifile)
		if err != nil {
			return c, err
		}
		rules = r
	}
	rules.AddDefaults()

	files := []*chartutil.BufferedFile{}
	dir += string(filepath.Separator)

	walk := func(name string, in packd.File) error {
		fi, _ := in.FileInfo()

		n := strings.TrimPrefix(name, dir)
		if n == "" {
			return nil
		}

		n = filepath.ToSlash(n)

		if fi.IsDir() {
			if rules.Ignore(n, fi) {
				return filepath.SkipDir
			}
			return nil
		}

		if rules.Ignore(n, fi) {
			return nil
		}

		data := in.String()
		match, _ := regexp.MatchString("templates\\/[0-9a-zA-Z]*\\.yaml", n)
		if match && opts.Suite {
			data = fmt.Sprintf("{{- if .Values.%s.enabled }}\n%s\n{{- end }}\n", opts.Args.Name, in.String())
			parts := strings.SplitAfter(n, "/")
			n = filepath.Join(parts[0], opts.Args.Name, parts[1])
		}

		files = append(files, &chartutil.BufferedFile{Name: n, Data: []byte(data)})
		return nil
	}
	if err := box.Walk(walk); err != nil {
		return c, err
	}

	return chartutil.LoadFiles(files)
}

// CreateFrom overrides https://godoc.org/k8s.io/helm/pkg/chartutil#CreateFrom
func CreateFrom(chartfile *chart.Metadata, src, dest string) error {
	schart, err := LoadBox(src)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", src, err)
	}

	// schart.Metadata = chartfile
	schart.Metadata.Name = chartfile.Name

	var updatedTemplates []*chart.Template

	for _, template := range schart.Templates {
		newData := chartutil.Transform(string(template.Data), "<CHARTNAME>", schart.Metadata.Name)
		updatedTemplates = append(updatedTemplates, &chart.Template{Name: template.Name, Data: newData})
	}

	schart.Templates = updatedTemplates
	if schart.Values != nil {
		schart.Values = &chart.Config{Raw: string(chartutil.Transform(schart.Values.Raw, "<CHARTNAME>", schart.Metadata.Name))}
	}
	return chartutil.SaveDir(schart, dest)
}
