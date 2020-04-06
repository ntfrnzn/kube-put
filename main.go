package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	// "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"

	//utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
)

// Scheme is the basic k8s scheme
var Scheme *runtime.Scheme

func init() {
	Scheme = scheme.Scheme //runtime.NewScheme()
}

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

	obj, err := loadObject(file)
	if err != nil {
		panic(err.Error())
	}
	if obj == nil {
		log.Fatal("oops, no object")
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	fmt.Printf("%s\n", gvk.GroupKind())

	// create the dynamic client from kubeconfig
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

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
	// create the object using the dynamic client
	resource := schema.GroupVersionResource{Version: gvk.Version, Resource: resourceName} // kindToResource(gvk.Kind)}

	createdUnstructuredObj, err := dynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	log.Println(createdUnstructuredObj.GetName())

}

func loadObject(filename string) (runtime.Object, error) {

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// fmt.Println(string(data))

	decoder := serializer.NewCodecFactory(Scheme, serializer.EnableStrict).UniversalDeserializer()
	obj, _ /* gvk */, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	// fmt.Println("obj: ", obj)
	// fmt.Println("gvk: ", gvk)

	return obj, nil
}

func kindToResource(kind string) string {
	return strings.ToLower(kind) + "s"
}
