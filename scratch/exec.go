package main

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
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

	pod, err := clientset.CoreV1().Pods("default").Get(context.Background(), "openldap-sample", metav1.GetOptions{})

	if err != nil {
		panic(err)
	}

	restClient := clientset.CoreV1().RESTClient()
	req := restClient.Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   []string{"/bin/sh"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		panic(err)
	}

	in := strings.NewReader("ls")
	out := strings.Builder{}
	eout := strings.Builder{}

	// Connect this process' std{in,out,err} to the remote shell process.
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  in,
		Stdout: &out,
		Stderr: &eout,
		Tty:    false,
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("The stdout of the comand is: " + out.String())
	fmt.Println("The stderr of the comand is: " + out.String())

}
