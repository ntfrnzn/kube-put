package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ntfrnzn/kube-put/internal/util"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	var kubeconfig string
	// var file string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	// flag.StringVar(&file, "file", "", "json or yaml file describing kubernetes object")
	flag.Parse()

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	// if file == "" {
	// 	log.Fatal("must supply an input file")
	// }

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// objects, err := util.ReadObjects(file)

	objects, err := util.LoadObjects()
	if err != nil {
		panic(err.Error())
	}

	for _, a := range objects {

		var pause = 30*time.Second
		var installError error
		for i := 0; i < 10; i++ {
			installError = util.Put(a, config)
			if installError != nil {
				log.Printf("Error: %s, pausing %s", installError, pause)
				time.Sleep( pause )
			} else {
				break
			}
		}
		if installError != nil {
			log.Fatal("Error installing %s, %w", a.Runtime.GetObjectKind().GroupVersionKind().String, installError)
		}
	}

}
