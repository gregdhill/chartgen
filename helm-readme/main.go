package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	flags "github.com/jessevdk/go-flags"
	yaml "gopkg.in/yaml.v2"
)

var opts struct {
	Values   string `short:"v" long:"values" required:"yes" description:"Helm values file."`
	Template string `short:"t" long:"template" required:"yes" description:"Template to format."`
}

// explore the tree of values
func search(spec map[interface{}]interface{}, prefix string, fields map[string]string) {
	for key, value := range spec {
		target := fmt.Sprintf("%v%s", prefix, key)
		switch def := value.(type) {
		case bool, int, string:
			fields[target] = fmt.Sprintf("%v", def)
		case []interface{}:
			fields[target] = "[]"
		case map[interface{}]interface{}:
			search(def, fmt.Sprintf("%s.", target), fields)
		}
	}
}

func main() {

	// parse the flags (all must be set)
	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	// open the given values file
	data, err := ioutil.ReadFile(opts.Values)
	if err != nil {
		log.Fatalln(err)
	}

	// start with a generic map and build explicit string mapping
	spec := make(map[interface{}]interface{}, len(data))
	if err = yaml.Unmarshal(data, &spec); err != nil {
		log.Fatalln(err)
	}
	fields := make(map[string]string, len(spec))
	search(spec, "", fields)

	// template the markdown file
	buf := new(bytes.Buffer)
	t, err := template.New(opts.Template).ParseFiles(opts.Template)
	if err != nil {
		log.Fatalln(err)
	}

	err = t.Execute(buf, fields)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(buf.String())

}
