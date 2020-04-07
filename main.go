package main

import (
	"flag"
	"log"
	"os"

	"github.com/ntfrnzn/kube-put/internal/util"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	var kubeconfig string
	var file string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&file, "file", "", "json or yaml file describing kubernetes object")
	flag.Parse()

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	if file == "" {
		log.Fatal("must supply an input file")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	objects, err := util.ReadObjects(file)
	if err != nil {
		panic(err.Error())
	}

	for _, obj := range objects {
		err := util.Put(obj, config)
		if err != nil {
			panic(err.Error())
		}
	}

}
