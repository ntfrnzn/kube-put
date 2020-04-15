package util

import (
	"context"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)


var (
	gkIssuer = schema.GroupKind{Group: "cert-manager.io", Kind: "Issuer"}
	gkAPIServer = schema.GroupKind{Group: "apiregistration.k8s.io", Kind: "APIService"}
)


func Apply(ao ApplyObject, config *rest.Config) error {

	// create the dynamic client from kubeconfig
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	gvk, resourceName, err := Discover(ao.Runtime, config)
	if err != nil {
		return err
	}

	fmt.Printf("%s   %s/%s \n", gvk.GroupKind(), ao.Unstruc.GetNamespace(), ao.Unstruc.GetName())
	// create the object using the dynamic client
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resourceName,
	}

	log.Println(ao.Unstruc.GetName())
	ns := ao.Unstruc.GetNamespace()
	var patched *unstructured.Unstructured
	var iface dynamic.ResourceInterface
	if ns == "" {
		iface = dynamicClient.Resource(gvr)
	} else {
		iface = dynamicClient.Resource(gvr).Namespace(ns)
	}

	forceApply := true // https://github.com/kubernetes/kubernetes/issues/89954
	// see also https://github.com/kubernetes-sigs/structured-merge-diff/issues/130
	// :( :( https://github.com/kubernetes/kubernetes/issues/89264
	if gvk.GroupKind() == gkAPIServer {
		log.Printf("Can't \"apply\" APIService %s because of https://github.com/kubernetes/kubernetes/issues/89264", ao.Unstruc.GetName() )
		_, err = iface.Get(context.TODO(), ao.Unstruc.GetName(), metav1.GetOptions{TypeMeta: metav1.TypeMeta{Kind: gvk.Kind, APIVersion: gvk.Version}})
		if err != nil {
			log.Printf("Error in Get: %+v", err)
		}
		if err != nil && errors.IsNotFound(err) {
			patched, err = iface.Create(context.TODO(), ao.Unstruc, metav1.CreateOptions{})
			if err != nil {
				log.Printf("Error in Create: %+v", err)
			}
		} else {
			if err != nil {
				log.Printf("Error: %+v", err)
				return fmt.Errorf("cannot get resource %s %s, %w", gvr.String(), ao.Unstruc.GetName(), err)
			}
		}
	} else {
		patched, err = iface.Patch(
			context.TODO(),
			ao.Unstruc.GetName(),
			types.ApplyPatchType,
			ao.Raw,
			metav1.PatchOptions{
				Force:        &forceApply,
				FieldManager: "kube-put",
			},
		)
	}

	if err != nil {
		log.Printf("Error: %+v", err)
		return fmt.Errorf("cannot apply resource %s %s, %w", gvr.String(), ao.Unstruc.GetName(), err)
	}

	if patched == nil {
		log.Printf("skipped %v\n", ao.Unstruc.GetName())
	} else {
		log.Printf("applied %v\n", patched.GetName())
	}
	return nil
}
