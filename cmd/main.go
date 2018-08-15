package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"local/controller"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type Application struct {
	RuntimePath string
	SpecPath    string
}

var app Application

func init() {
	flag.StringVar(&app.RuntimePath, "runtime", "./bins/shellout.so", "The path to the runtime plugin library")
	flag.StringVar(&app.SpecPath, "spec", "/spec.json", "The path to the podspec to start")
}

func main() {
	flag.Parse()

	var spec controller.PodSpec
	specContents, err := ioutil.ReadFile(app.SpecPath)
	if err != nil {
		log.Fatalf("failed to read contents of spec path %s: %v", app.SpecPath, err)
	} else if err := unmarshal(specContents, &spec); err != nil {
		log.Fatalf("failed to parse spec contents: %v", err)
	}

	log.Printf("PodSpec: %+v\n", spec)
	ctrl, err := controller.NewPodController(spec, app.RuntimePath)
	if err != nil {
		log.Fatalf("failed to initialize pod controller: %v", err)
	}
	ctrl.Start()
	log.Println("pod controller started")
	select {}
}

func unmarshal(contents []byte, spec *controller.PodSpec) error {
	jsonErr := json.Unmarshal(contents, spec)
	if jsonErr == nil {
		return nil
	}
	yamlErr := yaml.Unmarshal(contents, spec)
	if yamlErr == nil {
		return nil
	}
	return fmt.Errorf("failed to unmarshal spec contents into podspec struct: %v (json) | %v (yaml)", jsonErr, yamlErr)
}
