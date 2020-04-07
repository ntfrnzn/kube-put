package util

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/rest"
)


func Put(obj runtime.Object, config  *rest.Config ) error {

	// create the dynamic client from kubeconfig
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil { return err }

	gvk, resourceName, err := Discover(obj, config)
	if err != nil {
		return err
	}

	// convert the runtime.Object to unstructured.Unstructured
	unstructuredData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	unstructuredObj := &unstructured.Unstructured{
		Object: unstructuredData,
	}
	fmt.Printf("%s %s\n", gvk.GroupKind(), unstructuredObj.GetName())
	// create the object using the dynamic client
	resource := schema.GroupVersionResource{Version: gvk.Version, Resource: resourceName}

	createdUnstructuredObj, err := dynamicClient.Resource(resource).Namespace("default").Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	log.Println(createdUnstructuredObj.GetName())
	return nil
}

func Discover(obj runtime.Object, config  *rest.Config ) (*schema.GroupVersionKind, string, error){
	gvk := obj.GetObjectKind().GroupVersionKind()

	// --- get the resource name for the gvk
	client, err := discovery.NewDiscoveryClientForConfig(config)
	groupResources, err := restmapper.GetAPIGroupResources(client)
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	m, err := mapper.RESTMappings(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, "", err
	}
	if len(m) == 0 {
		return nil, "", fmt.Errorf("No resource found for %s", gvk.String())
	}
	resourceName := m[0].Resource.Resource
	// ---
	return &gvk, resourceName, nil
}