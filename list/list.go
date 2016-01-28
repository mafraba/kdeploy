package list

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/codegangsta/cli"
	"github.com/flexiant/kdeploy/utils"
	"github.com/flexiant/kdeploy/webservice"
)

// Flags builds a spec of the flags available for the command
func Flags() []cli.Flag {
	return []cli.Flag{}
}

// PrepareFlags processes the flags
func PrepareFlags(c *cli.Context) error {
	return nil
}

// struct representing an item to be listed
type Kube struct {
	Services    []map[string]string
	Controllers []string
}

type resourceList struct {
	Items []struct {
		Metadata struct {
			Name   string
			Labels map[string]string
		}
	}
}

// struct for unmarshalling json representations of services
type serviceList struct {
	Items []serviceItem
}

type serviceItem struct {
	Metadata struct {
		CreationTimestamp string
		Name              string
		Labels            map[string]string
	}
	Spec struct {
		ClusterIP string
	}
	Status struct {
		LoadBalancer struct {
			Ingress []struct {
				Hostname string
			}
		}
	}
}

// CmdList implements 'list' command
func CmdList(c *cli.Context) {
	// Get all services to extract their kubeware labels
	serviceList, err := getServices()
	utils.CheckError(err)
	// Get all controllers to extract their kubeware labels
	controllersList, err := getControllers()
	utils.CheckError(err)
	// build the list to be printed
	kubeList := buildKubeList(serviceList, controllersList)
	// print the list
	kyml, err := yaml.Marshal(kubeList)
	utils.CheckError(err)
	fmt.Printf("%s", kyml)
}

func buildKubeList(svcList *serviceList, rcList *resourceList) map[string]Kube {
	kmap := map[string]Kube{}
	for _, service := range svcList.Items {
		if kubeName, ok := service.Metadata.Labels["kubeware"]; ok {
			// check if kube already in map
			if _, ok := kmap[kubeName]; !ok {
				// if not, create it
				kmap[kubeName] = Kube{}
			}
			// add the service to the kube's list of services
			kube := kmap[kubeName]
			kube.Services = append(kube.Services, buildServiceRecord(service))
			kmap[kubeName] = kube
		}
	}
	for _, controller := range rcList.Items {
		if kubeName, ok := controller.Metadata.Labels["kubeware"]; ok {
			// check if kube already in map
			if _, ok := kmap[kubeName]; !ok {
				// if not, create it
				kmap[kubeName] = Kube{}
			}
			// add the controller to the kube's list of controllers
			kube := kmap[kubeName]
			kube.Controllers = append(kube.Controllers, controller.Metadata.Name)
			kmap[kubeName] = kube
		}
	}
	return kmap
}

func buildServiceRecord(service serviceItem) map[string]string {
	sr := map[string]string{}
	sr["Name"] = service.Metadata.Name
	sr["CreationDate"] = service.Metadata.CreationTimestamp
	sr["ClusterIP"] = service.Spec.ClusterIP
	if len(service.Status.LoadBalancer.Ingress) > 0 {
		sr["ExternalFQDN"] = service.Status.LoadBalancer.Ingress[0].Hostname
	}
	return sr
}

func getServices() (*serviceList, error) {
	kube, err := webservice.NewKubeClient()
	if err != nil {
		return nil, err
	}
	jsonServices, err := kube.GetServices(nil)
	if err != nil {
		return nil, err
	}
	return unmarshalServices(jsonServices)
}

func getControllers() (*resourceList, error) {
	kube, err := webservice.NewKubeClient()
	if err != nil {
		return nil, err
	}
	jsonControllers, err := kube.GetControllers(nil)
	if err != nil {
		return nil, err
	}
	return unmarshalResources(jsonControllers)
}

func unmarshalResources(jsonStr string) (*resourceList, error) {
	var rl resourceList
	err := json.Unmarshal([]byte(jsonStr), &rl)
	if err != nil {
		return nil, err
	}
	return &rl, nil
}

func unmarshalServices(jsonStr string) (*serviceList, error) {
	var sl serviceList
	err := json.Unmarshal([]byte(jsonStr), &sl)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

func extractKubes(rl *resourceList) []string {
	knames := []string{}
	for _, r := range rl.Items {
		if name, ok := r.Metadata.Labels["kubeware"]; ok {
			knames = append(knames, name)
		}
	}
	return knames
}