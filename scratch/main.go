package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// https://www.youtube.com/watch?v=jiKwjnlc7Wk
func main() {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		panic(err)
	}
	clientset := kubernetes.NewForConfigOrDie(config)

	nodeList, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, node := range nodeList.Items {
		fmt.Println(node.Name)
	}

	/*
		newPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "busybox", Image: "busybox:latest", Command: []string{"sleep", "300"}},
				},
			},
		}

		pod, err := clientset.CoreV1().Pods("default").Create(context.Background(), newPod, metav1.CreateOptions{})

		if err != nil {
			panic(err)
		}

		fmt.Println(pod)
	*/

	newConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openldap-sample",
		},
		Data: map[string]string{
			"config": `
			bla 
			bla
			#
			bla
			`,
		},
	}

	configmap, err := clientset.CoreV1().ConfigMaps("default").Create(context.Background(), newConfigMap, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println(configmap)
}
