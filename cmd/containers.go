package cmd

import (
	"github.com/spf13/cobra"
	"github.com/moshloop/smarti/pkg"
	"fmt"
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/core/v1"
	"strings"
)

var (

	Versions = cobra.Command{
		Use:  "versions",
		Short: "Print a list of all images and their resolved image tags",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			log.Infof("Exporting images versions from %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())

			var inv = pkg.Parse(cmd)

			for _, container := range inv.Containers() {
				fmt.Printf("%s\n", strings.Replace(container.Image, ":", ": ", 1))
			}
		},
	}

	Spec = cobra.Command{

		Use:  "spec",
		Short: "Generate a kubernetes spec for deployment",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			log.Infof("Running containers on %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())

			var inv = pkg.Parse(cmd)

			for _, container := range inv.Containers() {
				fmt.Printf("%s\n", container.ToDeployment())
			}

		},
	}

	Health = cobra.Command{
		Use: "health",
		Short: "Check the health of all services",
		Run: func(cmd *cobra.Command, args []string) {

			log.Infof("Checking the health of all services on %s/%s", cmd.Flag("inventory").Value.String(), cmd.Flag("limit").Value.String())
			print_pass, _ := cmd.Flags().GetBool("print")

			k8s, config := pkg.GetK8sClient("")

			nodes, _ := k8s.CoreV1().Nodes().List(meta_v1.ListOptions{})
			endpoint := ""
			for _, node := range nodes.Items {
				if endpoint != "" ||  node.Spec.Unschedulable {
					continue
				}
				for _, addr := range node.Status.Addresses {
					if addr.Type == v1.NodeInternalIP {
						endpoint = addr.Address
						log.Infof("Testing services using %s (%s)", node.Name, endpoint)
					}
				}
			}
			namespace := config.Contexts[config.CurrentContext].Namespace
			svcs, err := k8s.CoreV1().Services(namespace).List(meta_v1.ListOptions{})
			if err != nil {
				panic(err)
			}
			for _, svc := range svcs.Items {
				if svc.Spec.Type == v1.ServiceTypeNodePort {
					for _, port := range svc.Spec.Ports {
						if pkg.PingPort(endpoint, int(port.NodePort)) {
							log.Infof("%-20s %-5s -> %d = ✓", svc.Name, port.TargetPort.String(), port.NodePort)
							if print_pass {
								fmt.Printf("%s:%v\n", endpoint, port.NodePort)
							}
						} else {
							log.Errorf("%-20s %-5s -> %d = ❌", svc.Name, port.TargetPort.String(), port.NodePort)
						}
					}
				}

			}
		},

	}



	Containers = cobra.Command{

		Use:  "k8s",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			print("Must specify a sub-command, run `smarti k8s --help` for more details")
		},


	}


)


