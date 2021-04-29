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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
	"k8s.io/client-go/util/homedir"
)

type enableResponseCaching struct {
	rt            http.RoundTripper
	maxAgeSeconds int
}

func (rt *enableResponseCaching) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", rt.maxAgeSeconds)) // cache response for 5 minutes
	return resp, nil
}

func EnableResponseCaching(rt http.RoundTripper, maxAgeSeconds int) http.RoundTripper {
	return &enableResponseCaching{rt, maxAgeSeconds}
}

var _ http.RoundTripper = &enableResponseCaching{}

func CacheResponse(rt http.RoundTripper) http.RoundTripper {
	t := httpcache.NewMemoryCacheTransport()
	t.Transport = rt
	return t
}

func main() {
	masterURL := ""
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Fatalf("Could not get Kubernetes config: %s", err)
	}

	c2 := rest.CopyConfig(config)
	c2.Wrap(transport.Wrappers(EnableResponseCaching, CacheResponse))

	dc2 := dynamic.NewForConfigOrDie(c2)

	gvrNode := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}
	nodes, err := dc2.Resource(gvrNode).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, obj := range nodes.Items {
		fmt.Printf("%+v\n", obj.GetName())
	}
	nodes, err = dc2.Resource(gvrNode).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, obj := range nodes.Items {
		fmt.Printf("%+v\n", obj.GetName())
	}

	c3 := rest.CopyConfig(config)
	// c3.Wrap(transport.Wrappers(EnableResponseCaching, CacheResponse))
	dc3 := dynamic.NewForConfigOrDie(c3)

	nodes, err = dc3.Resource(gvrNode).List(context.TODO(), metav1.ListOptions{})
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
	result, err := dc2.Resource(gvrPod).Namespace("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: sel.String(),
	})
	if err != nil {
		panic(err)
	}
	for _, obj := range result.Items {
		fmt.Println(obj.GetName())
	}
}
