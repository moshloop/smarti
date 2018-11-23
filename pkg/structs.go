package pkg

import (

	log "github.com/sirupsen/logrus"
)

type Group struct {
	Name         string
	ParentGroups [] string
	Vars         map[string]interface{}
}

type Host struct {
	Name   string
	Groups [] Group
	Vars   map[string]interface{}
}

type Inventory struct {
	Groups map[string]Group
	Hosts  map[string]Host
	Vars   map[string]interface{}
}

func (inv Inventory) AddGroup(group Group) {
	if group.Vars == nil {
		group.Vars = make(map[string]interface{})
	}
	inv.Groups[group.Name] = group
}

func (inv Inventory) AddHost(host Host) {
	inv.Hosts[host.Name] = host
}

func (inv Inventory) Merge() {

	groups := inv.Groups
	vars := inv.Vars;
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
	//println(fmt.Sprintf("%v",inv.Groups))
}

func NewInventory() Inventory {
	inv := Inventory{}
	inv.Vars = make(map[string]interface{})
	inv.Groups = make(map[string]Group)
	inv.Hosts = make(map[string]Host)
	return inv
}
