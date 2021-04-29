package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gregjones/httpcache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	masterURL := ""
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		t := httpcache.NewMemoryCacheTransport()
		t.Transport = rt
		return t
	})

	dc := dynamic.NewForConfigOrDie(config)

	nodes, err := dc.Resource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, obj := range nodes.Items {
		fmt.Printf("%+v\n", obj.GetName())
	}

	sel, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels:      nil,
		MatchExpressions: nil,
	})
	if err != nil {
		panic(err)
	}
	gvrPod := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	result, err := dc.Resource(gvrPod).Namespace("default").List(context.TODO(), metav1.ListOptions{
		LabelSelector: sel.String(),
	})
	if err != nil {
		panic(err)
	}
	for _, obj := range result.Items {
		fmt.Println(obj.GetName())
	}
}