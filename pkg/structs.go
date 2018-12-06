package pkg

import (
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"strings"
	"fmt"
)

type Group struct {
	Name         string
	ParentGroups [] string
	Vars         map[string]interface{}
	Containers 	 []*Container
	ContainerDefaults ContainerDefaults
	Inventory 	*Inventory
}

type Host struct {
	Name   string
	Groups []Group
	Vars   map[string]interface{}
}

type Inventory struct {
	Groups map[string]*Group
	Hosts  map[string]*Host
	Vars   map[string]interface{}
	Limit string
}


func (inv Inventory) IsLimited(name string) bool {
	if inv.Limit == "" {
		return false
	}
	for _ ,limit := range strings.Split(inv.Limit, ":") {
		if limit == name {
			return false
		}
	}
	return true
}

func (inv Inventory) AddGroup(group Group) {
	if group.Vars == nil {
		group.Vars = make(map[string]interface{})
	}
	if group.Containers == nil {
		group.Containers = []*Container{}
	}
	defaults, ok := group.Vars["container_defaults"];
	if  ok {
		mapstructure.Decode(defaults,&group.ContainerDefaults )
		//FIXME .Decode for some reason doesn't pick up service_type
		service_type, ok  := defaults.(map[string]interface{})["service_type"]
		if ok {
			group.ContainerDefaults.ServiceType = service_type.(string)
		}

	}
	inv.Groups[group.Name] = &group
	group.Inventory = &inv
}


func (g Group) Get(key string) string {

	val, ok := g.Inventory.Vars[key]

	if ok {
		return fmt.Sprintf("%v", val)
	}
	val, ok = g.Vars[key]

	if ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}


func (inv Inventory) AddHost(host Host) {
	inv.Hosts[host.Name] = &host
}

func (inv Inventory) Containers() []*Container {
	var containers []*Container
	for _, group := range inv.Groups {
		containers = append(containers, group.Containers...)
	}
	return containers
}

func (inv Inventory) Merge() {

	groups := inv.Groups
	vars := inv.Vars
	all := groups["all"]

	PutAll(vars, all.Vars)

	for _, group := range groups {
		vars := make(map[string]interface{})
		PutAll(all.Vars, vars)
		for _, parent := range group.ParentGroups {
			if group, ok := groups[parent]; ok {
				PutAll(group.Vars, vars)
			} else {
				log.Warningf("Missing group %s", parent)
			}
		}
		PutAll(group.Vars, vars)
		PutAll(vars, group.Vars)
		//group.Vars = vars
		//println(fmt.Sprintf("%v",group.Vars))
	}
	//TODO default functions should only be available on 2nd pass
	InterpolateGroups(inv.Groups)
	// 2nd pass as no variable dependency ordering is done
	InterpolateGroups(inv.Groups)

	for group := range inv.Groups {
		if inv.IsLimited(group) {
			log.Infof("Excluding %s", group)
			delete(inv.Groups, group)
		}
	}
}

func NewInventory() Inventory {
	inv := Inventory{}
	inv.Vars = make(map[string]interface{})
	inv.Groups = make(map[string]*Group)
	inv.Hosts = make(map[string]*Host)
	return inv
}
