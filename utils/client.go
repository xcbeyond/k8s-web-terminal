package utils

import (
	"io/ioutil"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	// kubernetes config file path
	KubeConfigPath string
	k8sClient      *kubernetes.Clientset
	restConfig     *rest.Config
}

func (t *Client) NewClient() error {
	kubeconfig, err := ioutil.ReadFile(t.KubeConfigPath)
	if err != nil {
		return err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return err
	}
	t.restConfig = restConfig

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	t.k8sClient = clientset

	return nil
}

func (t *Client) K8sClient() *kubernetes.Clientset {
	return t.k8sClient
}

func (t *Client) RestConfig() *rest.Config {
	return t.restConfig
}
