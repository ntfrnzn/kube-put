package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ntfrnzn/kube-put/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
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


	// create the dynamic client from kubeconfig
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()

		// --- get the resource name for the gvk
		client, err := discovery.NewDiscoveryClientForConfig(config)
		groupResources, err := restmapper.GetAPIGroupResources(client)
		mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

		m, err := mapper.RESTMappings(gvk.GroupKind(), gvk.Version)
		if err != nil {
			panic(err.Error())
		}
		if len(m) == 0 {
			panic("no resources")
		}
		resourceName := m[0].Resource.Resource
		// ---

		// convert the runtime.Object to unstructured.Unstructured
		unstructuredData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			panic(err.Error())
		}
		unstructuredObj := &unstructured.Unstructured{
			Object: unstructuredData,
		}
		fmt.Printf("%s %s\n", gvk.GroupKind(), unstructuredObj.GetName())
		// create the object using the dynamic client
		resource := schema.GroupVersionResource{Version: gvk.Version, Resource: resourceName}

		createdUnstructuredObj, err := dynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			panic(err.Error())
		}

		log.Println(createdUnstructuredObj.GetName())
	}

}
