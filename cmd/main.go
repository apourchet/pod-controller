package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"local/controller"
	"log"
	"net/http"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type Application struct {
	RuntimePath string
	SpecPath    string
	StatusPort  int
}

var app Application

func init() {
	flag.StringVar(&app.RuntimePath, "runtime", "./bins/shellout.so", "The path to the runtime plugin library")
	flag.StringVar(&app.SpecPath, "spec", "/spec.json", "The path to the podspec to start")
	flag.IntVar(&app.StatusPort, "port", 8888, "The port that we will listen on to report the status of the pod")
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

	createHandlers(ctrl)
	if err = http.ListenAndServe(fmt.Sprintf(":%d", app.StatusPort), nil); err != nil {
		log.Fatalf("failed to create listen on given port %d: %v", app.StatusPort, err)
	}
}

func createHandlers(ctrl controller.PodController) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		statuses := ctrl.Status()
		content, err := json.Marshal(statuses)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	http.HandleFunc("/healthy", func(w http.ResponseWriter, r *http.Request) {
		healthy := ctrl.Healthy()
		content := fmt.Sprintf(`{"healthy":%v}`, healthy)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	})
	http.HandleFunc("/kill", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(1)
		}()
	})
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
