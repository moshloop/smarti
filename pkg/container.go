package pkg

import (
	"fmt"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	log "github.com/sirupsen/logrus"
	"github.com/flosch/pongo2"
	"strconv"
	"strings"
	"errors"
	"path"
	"io/ioutil"
)

type StringOrInt interface{}

type Container struct {
	Image          string            `json:"image,omitempty"`
	ImageName      string
	ImageTag       string
	ImageDigest    string
	Ingress        string            `json:"ingress,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Command        []string          `json:"command,omitempty"`
	Entrypoint     []string          `json:"entrypoint,omitempty"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	Hostname       string            `json:"hostname,omitempty"`
	User           string            `json:"user,omitempty"`
	Privileged     bool              `json:"privileged,omitempty"`
	Service        string            `json:"service,omitempty"`
	ServiceType    string            `json:"service_type,omitempty"`
	Mem            int               `json:"mem,omitempty"`
	Cpu            StringOrInt       `json:"cpu,omitempty"`
	Replicas       int32             `json:"replicas,omitempty"`
	Commands       []string          `json:"commands,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Files          map[string]string `json:"files,omitempty"`
	Templates      map[string]string `json:"templates,omitempty"`
	Mounts         map[string]string `json:"mounts,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	ContainerName  string            `json:"container_name,omitempty"`
	Ports          []ContainerPort   `json:"ports,omitempty"`
	Source         interface{}       `json:"source,omitempty"`
	ReadinessProbe *HealthCheck      `json:"readinessProbe,omitempty"`
	LivenessProbe  *HealthCheck      `json:"livenessProbe,omitempty"`
	Group          Group
	Specs          []interface{}
	K8VolumeMounts []v1.VolumeMount
	K8Volumes      []v1.Volume
}

type ContainerPort struct {
	Published int
	Target    int
	Name      string
	Protocol  string
}

type HealthCheck struct {
	Cmd     string
	Url     string
	Port    int
	Period  int32 `json:"period,omitempty"`
	Timeout int32 `json:"timeout,omitempty"`
	Delay   int32 `json:"delay,omitempty"`
}

type ContainerDefaults struct {
	ReadinessProbe *HealthCheck      `json:"readinessProbe,omitempty"`
	LivenessProbe  *HealthCheck      `json:"livenessProbe,omitempty"`
	ServiceType    string            `json:"service_type,omitempty"`
	Replicas       int32             `json:"replicas,omitempty"`
	Mem            int               `json:"mem,omitempty"`
	Cpu            StringOrInt       `json:"cpu,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	Ingress        string            `json:"ingress,omitempty"`
}

func (port ContainerPort) String() string {
	return fmt.Sprintf("%d:%d", port.Published, port.Target)
}

func ToName(name string) string {
	path := strings.Split(name, "/")
	name = path[len(path)-1]
	return strings.Replace(strings.ToLower(name), ".", "", -1)
}

func (c *Container) PostProcess() {
	//{{_docker_registry}}/{{_image}}
	//force_sha
	//labels
	defaults := c.Group.ContainerDefaults

	env := make(map[string]string)

	for k, v := range defaults.Env {
		env[k] = v
	}
	for k, v := range c.Env {
		env[k] = v
	}
	c.Env = env

	if c.Mem == 0 {
		c.Mem = defaults.Mem
	}

	if replicas, ok := c.Group.Inventory.Vars["replicas"]; ok {
		_replicas, _ := strconv.Atoi(fmt.Sprintf("%s", replicas))
		c.Replicas = int32(_replicas)
	}

	if c.Replicas == 0 {
		c.Replicas = defaults.Replicas
	}

	if c.ServiceType == "" {
		c.ServiceType = defaults.ServiceType
	}

	if c.Cpu == "0" || c.Cpu == 0 || c.Cpu == nil {
		c.Cpu = ""
	}
	if c.Cpu == "" || c.Cpu == nil || c.Cpu == 0 {
		c.Cpu = defaults.Cpu
	}

	c.ImageName = strings.Split(c.Image, ":")[0]
	if strings.Contains(c.Image, ":") {
		c.ImageTag = strings.Split(c.Image, ":")[1]
	} else {
		c.ImageTag = "latest"
	}

	if c.ContainerName == "" {
		c.ContainerName = ToName(c.ImageName)
	}

	if c.Service == "" {
		c.Service = c.ImageName
	}
	c.Service = ToName(c.Service)

	if c.Ingress == "" && defaults.Ingress != "" {
		c.Ingress = c.Service + "." + defaults.Ingress
	}
	if len(c.Ports) > 0 && c.ReadinessProbe == nil {
		c.ReadinessProbe = &HealthCheck{
			Port: c.Ports[0].Target,
		}

		if defaults.ReadinessProbe != nil  {
			c.ReadinessProbe.Delay = defaults.ReadinessProbe.Delay
			c.ReadinessProbe.Timeout = defaults.ReadinessProbe.Timeout
			c.ReadinessProbe.Period = defaults.ReadinessProbe.Period

		}
	}

	if c.ReadinessProbe != nil && c.LivenessProbe == nil {
		c.LivenessProbe = c.ReadinessProbe
	}
	if c.LivenessProbe != nil && defaults.LivenessProbe != nil {
		if c.LivenessProbe.Timeout == 0 {
			c.LivenessProbe.Timeout = defaults.LivenessProbe.Timeout
		}
		if c.LivenessProbe.Period == 0 {
			c.LivenessProbe.Period = defaults.LivenessProbe.Period
		}
	} else if defaults.LivenessProbe != nil {
		c.LivenessProbe = defaults.LivenessProbe
	}

	versions := c.Group.Vars["image_versions"]
	versionsMap := make(map[string]interface{})
	if versions != nil {
		versionsMap = versions.(map[string]interface{})
	}
	if version, ok := versionsMap[c.ImageName]; ok {
		c.ImageTag = fmt.Sprintf("%s", version)
		c.Image = c.ImageName + ":" + c.ImageTag
		log.Infof("[%s] Using %s specified in version file", c.ImageName, c.ImageTag)
		return

	}

	if c.Group.Get("latest_to_tag") == "true" {
		c.LatestToTag()
	}

	if c.Group.Get("latest_to_tag_harbor") == "true" || c.Group.Get("latest_to_tag_harbor") == "all" {
		LatestToTagHarbor(c)
	}
	log.Infof("%-25s cpu=%.f, mem=%d, tag=%s ", c.ImageName, c.Cpu, c.Mem, c.ImageTag)

}

func (port *ContainerPort) UnmarshalJSON(b []byte) error {
	str, _ := strconv.Unquote(string(b))
	port.Published, _ = strconv.Atoi(strings.Split(str, ":")[0])
	if !strings.Contains(str, ":") {
		port.Target = port.Published
	} else {
		port.Target, _ = strconv.Atoi(strings.Split(str, ":")[1])
	}
	return nil
}

func (c Container) String() string {
	return fmt.Sprintf("%s/%s[%s, %dMb]env:%v, ports:%v", c.Service, c.Image, c.Cpu, c.Mem, c.Env, c.Ports)
}

func (c Container) ToMem() resource.Quantity {
	qty, _ := resource.ParseQuantity(strconv.Itoa(c.Mem) + "Mi")
	return qty
}

// Sub subtracts the provided quantity from the current value in place. If the current
// value is zero, the format of the quantity will be updated to the format of y.
func Ratio(q resource.Quantity, ratio int) resource.Quantity {
	qty :=  *resource.NewScaledQuantity(q.MilliValue() / int64(ratio), resource.Milli)
	qty.Format = q.Format
	return qty
}

func (c Container) ToCpu() (resource.Quantity, error) {
	var qty resource.Quantity
	switch v := c.Cpu.(type) {
	case int:
		qty, _ = resource.ParseQuantity(strconv.Itoa(v))
		break
	case float64:
		qty, _ = resource.ParseQuantity(strconv.Itoa(int(v)))
		break
	case string:
		qty, _ = resource.ParseQuantity(v)
		break
	default:
		return resource.Quantity{}, errors.New("Missing quantity")
	}
	return qty, nil
}

func (c Container) ToPorts() []v1.ServicePort {
	var ports []v1.ServicePort

	for _, port := range c.Ports {
		protocol := v1.ProtocolTCP
		if port.Protocol == "udp" {
			protocol = v1.ProtocolUDP
		}
		ports = append(ports, v1.ServicePort{
			Name:       fmt.Sprintf("%d", port.Published),
			Protocol:   protocol,
			Port:       int32(port.Published),
			TargetPort: intstr.FromInt(port.Target),
		})
	}
	return ports
}

func (c Container) ToContainerPorts() []v1.ContainerPort {
	var ports []v1.ContainerPort

	for _, port := range c.Ports {
		ports = append(ports, v1.ContainerPort{
			ContainerPort: int32(port.Target),
		})
	}
	return ports
}

func (c Container) ToResources() v1.ResourceRequirements {

	limits := v1.ResourceList{}
	requests := v1.ResourceList{}
	if c.Mem > 0 {
		limits[v1.ResourceMemory] = c.ToMem()
		requests[v1.ResourceMemory] = Ratio(c.ToMem(), 2)
	}
	cpu, err := c.ToCpu()
	if err == nil && cpu.Value() > 0 {
		limits[v1.ResourceCPU] = cpu
		requests[v1.ResourceCPU] = Ratio(cpu, 4);
	}

	return v1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}
}

func (c Container) ToEnvVars() []v1.EnvVar {

	var vars []v1.EnvVar

	for k, v := range c.Env {
		vars = append(vars, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return vars

}

func (c Container) ToContainer() v1.Container {
	registry := c.Group.Vars["docker_registry"]
	if registry == nil {
		registry = ""
	} else {
		registry = registry.(string) + "/"
	}
	container := v1.Container{
		Image:        registry.(string) + c.Image,
		Args:         c.Args,
		Command:      c.Command,
		Resources:    c.ToResources(),
		Ports:        c.ToContainerPorts(),
		WorkingDir:   c.WorkingDir,
		VolumeMounts: c.K8VolumeMounts,
	}

	if len(c.Commands) > 0 {
		var sh []string
		sh = []string{"sh", "-c", strings.Join(c.Commands, ";")}
		container.Lifecycle = &v1.Lifecycle{
			PostStart: &v1.Handler{
				Exec: &v1.ExecAction{
					Command: sh,
				},
			},
		}
	}
	if c.ContainerName != "" {
		container.Name = c.ContainerName
	}
	if c.LivenessProbe != nil {
		container.LivenessProbe = c.LivenessProbe.ToProbe()
	}
	if c.ReadinessProbe != nil {
		container.ReadinessProbe = c.ReadinessProbe.ToProbe()
	}
	container.Env = c.ToEnvVars()
	return container
}

func (c Container) ToServiceType() v1.ServiceType {

	if c.ServiceType == "dnsrr" || c.ServiceType == "NodePort" {
		return v1.ServiceTypeNodePort
	} else if strings.ToLower(c.ServiceType) == "loadbalancer" {
		return v1.ServiceTypeLoadBalancer
	}
	return v1.ServiceTypeClusterIP
}

func (c Container) ToIngress() v1beta1.Ingress {
	return v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Service + "-ing",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: c.Ingress,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Backend: v1beta1.IngressBackend{
										ServicePort: intstr.FromInt(c.Ports[0].Published),
										ServiceName: c.Service,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c Container) ToHttpIngressPaths() []v1beta1.HTTPIngressPath {
	paths := []v1beta1.HTTPIngressPath{}

	for _, port := range c.Ports {
		paths = append(paths, v1beta1.HTTPIngressPath{
			Backend: v1beta1.IngressBackend{
				ServicePort: intstr.FromInt(port.Published),
				ServiceName: c.Service,
			},
		})
	}
	return paths

}

func (c Container) ToDeployment() string {
	var specs []interface{}
	specs = append(specs, c.ToConfigMaps()...)

	specs = append(specs, v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1beta2",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Service,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &c.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": c.Service,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": c.Service,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{c.ToContainer()},
					Volumes:    c.K8Volumes,
				},
			},
		},
	})

	if len(c.Ports) > 0 {
		specs = append(specs, v1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.Service,
			},
			Spec: v1.ServiceSpec{
				Type: c.ToServiceType(),
				Selector: map[string]string{
					"app": c.Service,
				},
				Ports: c.ToPorts(),
			},
		})

		if c.Ingress != "" {
			specs = append(specs, c.ToIngress())
		}

	}

	out := ""
	for _, spec := range specs {
		data, _ := yaml.Marshal(spec)
		out = out + "\n---\n" + string(data)

	}

	return out
}
func (c Container) ToVolumeMounts() []v1.VolumeMount {
	var mounts []v1.VolumeMount
	return mounts
}

func ConfigMapName(name string) string {
	return strings.Replace(strings.Replace(name, "/", "", -1), ".", "-", -1)
}

func (c *Container) ToConfigMaps() []interface{} {
	var configs []interface{}

	if c.K8Volumes == nil {
		c.K8Volumes = []v1.Volume{}
	}

	if c.K8VolumeMounts == nil {
		c.K8VolumeMounts = []v1.VolumeMount{}
	}
	cms := make(map[string]map[string]string)
	for _path, file := range c.Files {
		dir := path.Dir(_path)
		str, _ := ioutil.ReadFile("files/" + file)
		cm, exists := cms[dir]

		if !exists {
			cm = make(map[string]string)
			cms[dir] = cm
		}

		cm[path.Base(_path)] = string(str)
	}

	for _path, file := range c.Templates {
		dir := path.Dir(_path)
		data, _ := ioutil.ReadFile("files/" + file)
		template := ConvertSyntaxFromJinjaToPongo(string(data))
		tpl, err := pongo2.FromString(template)
		if err != nil {
			log.Warnf("Error parsing: %s: %v", template, err)
			continue
		}
		out, err := tpl.Execute(c.Group.Vars)
		if err != nil {
			log.Warnf("Error parsing: %s: %v", template, err)
			continue
		}

		cm, exists := cms[dir]

		if !exists {
			cm = make(map[string]string)
			cms[dir] = cm
		}

		cm[path.Base(_path)] = string(out)
	}

	for name, cm := range cms {
		c.K8Volumes = append(c.K8Volumes, v1.Volume{
			Name: ConfigMapName(name),
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: ConfigMapName(name),
					},
				},
			},
		})
		c.K8VolumeMounts = append(c.K8VolumeMounts, v1.VolumeMount{
			Name:      ConfigMapName(name),
			MountPath: name,
		})
		configs = append(configs, NewConfigMap(name, cm))
	}

	return configs
}
func (c HealthCheck) ToProbe() *v1.Probe {

	probe := &v1.Probe{
		TimeoutSeconds:      c.Timeout,
		PeriodSeconds:       c.Period,
		InitialDelaySeconds: c.Delay,
	}

	if c.Url != "" {
		probe.Handler = v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: c.Url,
			},
		}
	} else if c.Port > 0 {
		probe.Handler = v1.Handler{
			TCPSocket: &v1.TCPSocketAction{
				Port: intstr.FromInt(c.Port),
			},
		}
	} else if c.Cmd != " " {
		probe.Handler = v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{c.Cmd},
			},
		}
	} else {
		return nil
	}

	return probe
}

func NewConfigMap(_path string, content map[string]string) v1.ConfigMap {
	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigMapName(_path),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		Data: content,
	}
}

func NewContainerFromCompose(file string, group Group) []*Container {
	containers := []*Container{}

	// Parse the Compose File
	data, _ := ioutil.ReadFile(file)
	parsedComposeFile, _ := loader.ParseYAML(data)

	// Config file
	configFile := types.ConfigFile{
		Filename: file,
		Config:   parsedComposeFile,
	}

	// Config details
	configDetails := types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{configFile},
	}

	// Actual config
	// We load it in order to retrieve the parsed output configuration!
	// This will output a github.com/docker/cli ServiceConfig
	// Which is similar to our version of ServiceConfig
	currentConfig, err := loader.Load(configDetails)

	if err != nil {
		log.Errorf("Error parsing compose file %s: %s", file, err)
	}

	if currentConfig != nil && currentConfig.Services != nil {
		for _, service := range currentConfig.Services {
			c := new(Container)
			c.Group = group
			c.Service = service.Name
			if len(service.Command) != 0 {
				c.Command = service.Command
			}

			if service.Deploy.Resources.Limits != nil {
				c.Mem = int(service.Deploy.Resources.Limits.MemoryBytes / 1024 / 1024)
				c.Cpu = service.Deploy.Resources.Limits.NanoCPUs
			}
			if service.Deploy.Replicas != nil {
				replicas := int32(*service.Deploy.Replicas)
				c.Replicas = replicas
			}

			if service.Deploy.EndpointMode != "" {
				c.ServiceType = service.Deploy.EndpointMode
			}
			//if len(service.Configs) != 0 {
			//	c.Configs = service.Configs
			//}
			if service.ContainerName != "" {
				c.ContainerName = service.ContainerName
			}
			if len(service.Entrypoint) != 0 {
				c.Entrypoint = service.Entrypoint
			}
			if len(service.Environment) != 0 {
				c.Env = make(map[string]string)
				for k, v := range service.Environment {
					c.Env[k] = *v
				}
			}
			//if len(service.EnvFile) != 0 {
			//	c.EnvFile = service.EnvFile
			//}
			//if len(service.Expose) != 0 {
			//	c.Expose = service.Expose.([]string)
			//}
			if service.Hostname != "" {
				c.Hostname = service.Hostname
			}
			//if service.HealthCheck != nil {
			//	c.HealthCheck = service.HealthCheck
			//}
			if service.Image != "" {
				c.Image = service.Image
			}
			if len(service.Labels) != 0 {
				c.Labels = service.Labels
			}
			if len(service.Ports) != 0 {
				//c.Ports =

				for _, port := range service.Ports {
					c.Ports = append(c.Ports, ContainerPort{
						Target:    int(port.Target),
						Protocol:  port.Protocol,
						Published: int(port.Published),
						Name:      fmt.Sprintf("%d", port.Published),
					})
				}
			}
			if service.Privileged != c.Privileged {
				c.Privileged = service.Privileged
			}

			if service.User != "" {
				c.User = service.User
			}
			//if len(service.Volumes) != 0 {
			//	c.Volumes = service.Volumes
			//}
			if service.WorkingDir != "" {
				c.WorkingDir = service.WorkingDir
			}

			log.Infof("New compose container: %s", c)
			containers = append(containers, c)
		}
	}

	return containers
}

func NewContainerFromVars(vars []map[string]interface{}) Container {
	return Container{}
}
