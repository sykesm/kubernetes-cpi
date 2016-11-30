package config

import (
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/1.4/tools/clientcmd/api"
)

type Cluster struct {
	Server                   string `json:"server"`
	InsecureSkipTLSVerify    bool   `json:"insecure_skip_tls_verify,omitempty"`
	CertificateAuthorityData string `json:"certificate_authority_data"`
}

type AuthInfo struct {
	ClientCertificateData string `json:"client_certificate_data,omitempty"`
	ClientKeyData         string `json:"client_key_data,omitempty"`
	Token                 string `json:"token,omitempty"`
	Username              string `json:"username,omitempty"`
	Password              string `json:"password,omitempty"`
}

type Context struct {
	Cluster   string `json:"cluster"`
	AuthInfo  string `json:"user"`
	Namespace string `json:"namespace,omitempty"`
}

type Kubernetes struct {
	Clusters       map[string]*Cluster  `json:"clusters"`
	AuthInfos      map[string]*AuthInfo `json:"users"`
	Contexts       map[string]*Context  `json:"contexts"`
	CurrentContext string               `json:"current_context"`
}

func (k Kubernetes) NewClient(context string) (kubernetes.Interface, error) {
	rc, err := k.NonInteractiveClientConfig(context).ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(rc)
}

func (k Kubernetes) Context() string {
	return k.CurrentContext
}

func (k Kubernetes) Namespace() string {
	ns := "default"
	if dc := k.Contexts[k.CurrentContext]; dc != nil && len(dc.Namespace) != 0 {
		ns = dc.Namespace
	}
	return ns
}

func (k Kubernetes) ClientConfig() *clientcmdapi.Config {
	cc := clientcmdapi.NewConfig()
	for k, v := range k.Clusters {
		cc.Clusters[k] = v.api()
	}
	for k, v := range k.AuthInfos {
		cc.AuthInfos[k] = v.api()
	}
	for k, v := range k.Contexts {
		cc.Contexts[k] = v.api()
	}
	cc.CurrentContext = k.CurrentContext
	cc.Preferences = *clientcmdapi.NewPreferences()
	return cc
}

func (k Kubernetes) NonInteractiveClientConfig(context string) clientcmd.ClientConfig {
	if len(context) == 0 {
		context = k.CurrentContext
	}
	return clientcmd.NewNonInteractiveClientConfig(*k.ClientConfig(), context, &clientcmd.ConfigOverrides{}, &clientcmd.ClientConfigLoadingRules{})
}

func (a *AuthInfo) api() *clientcmdapi.AuthInfo {
	info := clientcmdapi.NewAuthInfo()
	info.Token = a.Token
	info.Username = a.Username
	info.Password = a.Password
	if len(a.ClientCertificateData) != 0 {
		info.ClientCertificateData = []byte(a.ClientCertificateData)
	}
	if len(a.ClientKeyData) != 0 {
		info.ClientKeyData = []byte(a.ClientKeyData)
	}
	return info
}

func (c *Cluster) api() *clientcmdapi.Cluster {
	cluster := clientcmdapi.NewCluster()
	cluster.Server = c.Server
	cluster.InsecureSkipTLSVerify = c.InsecureSkipTLSVerify
	if len(c.CertificateAuthorityData) != 0 {
		cluster.CertificateAuthorityData = []byte(c.CertificateAuthorityData)
	}
	return cluster
}

func (c *Context) api() *clientcmdapi.Context {
	context := clientcmdapi.NewContext()
	context.Cluster = c.Cluster
	context.AuthInfo = c.AuthInfo
	context.Namespace = c.Namespace
	return context
}
