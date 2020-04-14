package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"

	cmapi "github.com/jetstack/cert-manager/pkg/api"
	"github.com/ntfrnzn/kube-put/internal/box"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// Scheme is the basic k8s scheme
var Scheme *runtime.Scheme

func init() {
	Scheme = scheme.Scheme //runtime.NewScheme()
    apiextensions.AddToScheme(Scheme)
	cmapi.AddToScheme(Scheme)
}


const yamlSeparator = "\n---"
const separator = "---"

type ApplyObject struct {
	Raw []byte
	Unstruc *unstructured.Unstructured
	Runtime runtime.Object
}

func LoadObjects() ([]ApplyObject, error) {

	objects := []ApplyObject{}
	manifests := box.Boxed.List()
	for _, m := range manifests {
		log.Printf("Loading %s\n", m)
		data := box.Boxed.Get(m)
		
		scanner := bufio.NewScanner(bytes.NewReader(data))
		buf := make([]byte, 8*1024)
		scanner.Buffer(buf, 512 * 1024)

		scanner.Split(splitYAMLDocument)

		for scanner.Scan() {
			decoder := serializer.NewCodecFactory(Scheme, serializer.EnableStrict).UniversalDeserializer()
			a := ApplyObject{
				Raw: make([]byte, len(scanner.Bytes())),
			}
			copy(a.Raw,scanner.Bytes())
			obj, _ /* gvk */, err := decoder.Decode(a.Raw, nil, nil)
			if err != nil {
				return nil, err
			}
			a.Runtime = obj

			// convert the runtime.Object to unstructured.Unstructured
			unstructuredData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(a.Runtime)
			if err != nil {
				return nil, err
			}
			a.Unstruc = &unstructured.Unstructured{
				Object: unstructuredData,
			}

			objects = append(objects, a)
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Invalid input: %s", err)
		}
	}
	return objects, nil

}

func ReadObjects(filename string) ([]runtime.Object, error) {

	objects := []runtime.Object{}

	log.Printf("Loading %s\n", filename)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(splitYAMLDocument)

	for scanner.Scan() {
		decoder := serializer.NewCodecFactory(Scheme, serializer.EnableStrict).UniversalDeserializer()
		obj,  _ /* gvk */, err := decoder.Decode(scanner.Bytes(), nil, nil)
		if err != nil {
			return nil, err
		}
		objects = append(objects, obj)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Invalid input: %s", err)
	}
	return objects, nil
}

// FROM: https://github.com/kubernetes/apimachinery/blob/a98ff070d70e1d5c58428a86787e7a05a38cabe8/pkg/util/yaml/decoder.go#L142
// splitYAMLDocument is a bufio.SplitFunc for splitting YAML streams into individual documents.
func splitYAMLDocument(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
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

