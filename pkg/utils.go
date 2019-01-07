package pkg

import (
	"bytes"
	"syscall"
	"fmt"
	"os/exec"
	"os"
	"io/ioutil"
	"path/filepath"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"time"
)

// Exec is a helper that will run a command and capture the output
// in the case an error happens.
func Exec(cmd *exec.Cmd) error {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return fmt.Errorf(
				"%s exited with %d: %s",
				cmd.Path,
				status.ExitStatus(),
				buf.String())
		}
	}

	return fmt.Errorf("error running %s: %s", cmd.Path, buf.String())
}

func SafeRead(file string) string {
	if _, err := os.Stat(file); err != nil {
		return ""
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}

	return string(data[:])
}

func PingPort(ip string,  port int) bool{
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip,port),  time.Duration(10)*time.Second)
	if conn != nil {
		defer conn.Close()
	}
	if err == nil  {
		return true
	}
	return  false
}

func GetK8sClient(kubeconfig string) (*kubernetes.Clientset, *clientcmdapi.Config) {

	if home := homeDir(); home != ""  {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientconfig, _  := clientcmd.LoadFromFile(kubeconfig)
	//clientonfig.Contexts[clientonfig.CurrentContext].Namespace

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset, clientconfig

}

func homeDir() string {
	if h := os.Getenv("HOME");h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func PutAllIfAbsent(from map[string]string, into map[string]string) {
	for k,v  := range from {
		if _, ok := into[k]; !ok {
			into[k] = v
		}
	}
}