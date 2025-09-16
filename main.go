package main

import (
	"context"
	"flag"
	"log"
	"time"

	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	labelKey   = "spectrocloud.com/nodeprep"
	taintKey   = "spectrocloud.com/nodeprep"
	taintEff   = v1.TaintEffectNoSchedule
)

func removeTaintIfComplete(ctx context.Context, c kubernetes.Interface, node *v1.Node) error {
	// Only act if label == "complete"
	if node.Labels[labelKey] != "complete" {
		return nil
	}
	// Check if taint exists
	taints := node.Spec.Taints
	keep := make([]v1.Taint, 0, len(taints))
	found := false
	for _, t := range taints {
		if t.Key == taintKey && t.Effect == taintEff {
			found = true
			continue
		}
		keep = append(keep, t)
	}
	if !found {
		return nil
	}

	// Patch via Update (simple & fine for this use case)
	// Retry a couple times in case of conflicts
	for i := 0; i < 3; i++ {
		n, err := c.CoreV1().Nodes().Get(ctx, node.Name, meta.GetOptions{})
		if err != nil {
			return err
		}
		newTaints := make([]v1.Taint, 0, len(n.Spec.Taints))
		for _, t := range n.Spec.Taints {
			if !(t.Key == taintKey && t.Effect == taintEff) {
				newTaints = append(newTaints, t)
			}
		}
		if len(newTaints) == len(n.Spec.Taints) {
			return nil // already removed
		}
		n.Spec.Taints = newTaints
		_, err = c.CoreV1().Nodes().Update(ctx, n, meta.UpdateOptions{})
		if err == nil {
			log.Printf("[nodeprep] removed taint from node %s", n.Name)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func main() {
	flag.Parse()
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	// Shared informer for Nodes; resync 0 = event-driven, no periodic list
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Nodes().Informer()

	ctx := context.Background()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if n, ok := obj.(*v1.Node); ok {
				_ = removeTaintIfComplete(ctx, clientset, n)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if n, ok := newObj.(*v1.Node); ok {
				_ = removeTaintIfComplete(ctx, clientset, n)
			}
		},
	})

	stop := make(chan struct{})
	defer close(stop)
	informer.Run(stop)
}
