package util

import (
	"context"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

func Put(obj runtime.Object, config *rest.Config) error {

	// create the dynamic client from kubeconfig
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

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
	fmt.Printf("%s   %s/%s \n", gvk.GroupKind(), unstructuredObj.GetNamespace(), unstructuredObj.GetName())
	// create the object using the dynamic client
	gvr := schema.GroupVersionResource{
		Group: gvk.Group,
		Version: gvk.Version,
		Resource: resourceName,
	}

	log.Println(unstructuredObj.GetName())
	ns := unstructuredObj.GetNamespace()
	var createdUnstructuredObj *unstructured.Unstructured
	var iface dynamic.ResourceInterface
	if ns == "" {
		iface = dynamicClient.Resource(gvr)
	} else {
		iface = dynamicClient.Resource(gvr).Namespace(ns)
	}

	_, err = iface.Get(context.TODO(), unstructuredObj.GetName(), metav1.GetOptions{}) //TypeMeta: metav1.TypeMeta{Kind: gvk.Kind, APIVersion: gvk.Version}})
	if err != nil && errors.IsNotFound(err) {
		createdUnstructuredObj, err = iface.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	} else {
		if err != nil {
			log.Printf("Error: %+v", err)
			return fmt.Errorf("cannot get resource %s %s, %w", gvr.String(), unstructuredObj.GetName(), err)
		}
	}

	if err != nil {
		log.Printf("Error: %+v", err)
		return fmt.Errorf("cannot create resource %s %s, %w", gvr.String(), unstructuredObj.GetName(), err)
	}

	if createdUnstructuredObj == nil {
		log.Printf("skipped %v\n", unstructuredObj.GetName())
	} else {
		log.Printf("created %v\n", createdUnstructuredObj.GetName())
	}
	return nil
}

func Discover(obj runtime.Object, config *rest.Config) (*schema.GroupVersionKind, string, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()

	// get the resource name for the gvk
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

	return &gvk, resourceName, nil
}
