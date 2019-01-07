package pkg

import (
	"io/ioutil"
	"os"
	"github.com/spf13/cobra"
	"path"
	"strings"
	"github.com/knq/ini"
	"regexp"
	"github.com/getlantern/deepcopy"
	log "github.com/sirupsen/logrus"
	"sync"
)

func Parse(cmd *cobra.Command) Inventory {
	dir := cmd.Flag("inventory").Value.String()
	inventory := NewInventory()
	inventory.Limit = cmd.Flag("limit").Value.String()

	inventory.Vars["inventory_name"] = path.Base(dir)
	inventory.Vars["inventory_dir"] = dir
	inventory.Vars["inventory_file"] = dir + "/hosts"
	extra, _ := cmd.Flags().GetStringSlice("extra-vars")
	ParseExtraVars(extra, inventory)

	ParseInventory(dir, inventory)
	ParseContainers(inventory)
	//var hosts = make(map[string]*Host)


	imageVersions, _ := cmd.Flags().GetString("image-versions")
	log.Infof("Using image versions from: %s", imageVersions)
	if imageVersions != "" {
		versions := ParseFile(imageVersions, inventory)
		log.Debugf("Parsed image versions: %v", versions)
		inventory.Vars["image_versions"] = versions
	}
	inventory.Merge()

	wg := new(sync.WaitGroup)
	for _, group := range inventory.Groups {
		for _, c := range group.Containers {
			wg.Add(1)
			go func(c *Container) {
				c.PostProcess()
				log.Debugf("Finished processing %s", c.Image)
				wg.Done()
			}(c)
		}
	}
	wg.Wait()
	return inventory
}



func ParseContainers(inv Inventory) {

	for _, group := range inv.Groups {
		containers := group.Vars["containers"]

		if containers != nil {
			for _, container := range containers.([]interface{}) {
				c := new(Container)
				c.Replicas = 1
				deepcopy.Copy(c, container)
				c.Group = *group
				group.Containers = append(group.Containers, c)
			}
		}

		compose_files := group.Vars["docker_compose_v3"]
		if compose_files != nil {
			for _, file := range compose_files.([]interface{}) {
				containers := NewContainerFromCompose(file.(string), *group)
				group.Containers = append(group.Containers, containers...)
			}
		}
	}

}


func ParseInventory(dir string, inventory Inventory) {
	if _, err := os.Stat(dir); err != nil {
		log.Debugf("Parsing inventory from string: " + dir)

		for _, host := range strings.Split(dir, ",") {
			inventory.AddHost(Host{
				Name: host,
			})

		}
	} else {
		log.Debugf("Parsing inventory from file: " + dir)
		ParseGroups(dir+"/group_vars", inventory)
		ParseGroupIni(dir, inventory)
	}

	if _, present := inventory.Groups["all"]; !present {
		inventory.AddGroup(Group{
			Name: "all",
		})
	}
	log.Debugf("Groups: %s", inventory.Groups)
	log.Debugf("Hosts: %s", inventory.Hosts)

}

func ParseExtraVars(extra []string, inventory Inventory) {
	for _, val := range extra {
		if strings.HasPrefix(val, "@") {
			vars := ParseFile(val[1:], inventory)
			for key := range vars {
				inventory.Vars[key] = vars[key]
			}
		} else {
			parts := strings.Split(val, "=")
			inventory.Vars[parts[0]] = parts[1]
		}
	}
}


func ParseGroupIni(dir string, inventory Inventory) {

	content := SafeRead(dir + "/groups")

	// ini syntax doesn't support just a key value so we replace all names with name=name
	re := regexp.MustCompile("(?m)(^\\w+)(.*)")
	content = re.ReplaceAllString(content, "$1=$1 $2")
	cfg, err := ini.LoadString(content)
	if err != nil {
		log.Fatalf("%s\n%s", err, content)
	}

	for _, section := range cfg.SectionNames() {
		if strings.HasSuffix(section, ":children") {
			for _, key := range cfg.GetSection(section).Keys() {
				//FIXME what happens if the group has not been declared yet?
				//FIXME what happens if the group has not been declared yet?
				group := inventory.Groups[key]
				group.ParentGroups = append(group.ParentGroups, strings.Split(section, ":")[0])
			}
		}
	}

}

func ParseGroups(dir string, inventory Inventory) {
	if _, err := os.Stat(dir); err != nil {
		return
	}

	log.Infof("Parsing groups from: %s", dir)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Error(err)
	}

	for _, f := range files {
		group := Group{Name: f.Name()}
		if _, err := os.Stat(dir + "/" + f.Name()); os.IsNotExist(err) {
			log.Warningf("Invalid symlink: %s", f.Name())
			continue
		}
		if f.IsDir() {
			children, _ := ioutil.ReadDir(dir + "/" + f.Name())
			for _, c := range children {
				vars := ParseFile(dir+"/"+f.Name()+"/"+c.Name(), inventory)
				if group.Vars != nil {
					for k, v := range vars {
						group.Vars[k] = v
					}
				} else {
					group.Vars = vars
				}
			}
		} else {
			group.Vars = ParseFile(dir+"/"+f.Name(), inventory)
		}
		inventory.AddGroup(group)
	}
}
