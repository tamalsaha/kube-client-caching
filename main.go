package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/transport"
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

type enableResponseCaching struct {
	rt http.RoundTripper
}

func (a *enableResponseCaching) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := a.rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Header.Set("Cache-Control", "max-age=300") // cache response for 5 minutes
	return resp, nil
}

func EnableResponseCaching(rt http.RoundTripper) http.RoundTripper {
	return &enableResponseCaching{rt}
}

var _ http.RoundTripper = &enableResponseCaching{}

type acceptCachedResponse struct {
	rt http.RoundTripper
}

func (u *acceptCachedResponse) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cache-Control", "only-if-cached")
	return u.rt.RoundTrip(req)
}

func AcceptCachedResponse(rt http.RoundTripper) http.RoundTripper {
	return &acceptCachedResponse{rt}
}

var _ http.RoundTripper = &acceptCachedResponse{}

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

	config.Wrap(transport.Wrappers(EnableResponseCaching, CacheResponse))

	//config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
	//	t := httpcache.NewMemoryCacheTransport()
	//
	//	t.Transport = &enableResponseCaching{rt}
	//	return t
	//})

	dc := dynamic.NewForConfigOrDie(config)

	gvrNode := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}
	nodes, err := dc.Resource(gvrNode).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, obj := range nodes.Items {
		fmt.Printf("%+v\n", obj.GetName())
	}
	nodes, err = dc.Resource(gvrNode).List(context.TODO(), metav1.ListOptions{})
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
	result, err := dc.Resource(gvrPod).Namespace("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: sel.String(),
	})
	if err != nil {
		panic(err)
	}
	for _, obj := range result.Items {
		fmt.Println(obj.GetName())
	}
}
