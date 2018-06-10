package pkg

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"os"
	"github.com/spf13/cobra"
	"path"
	"strings"
	"github.com/knq/ini"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/go-getter"
)

func Parse(cmd *cobra.Command) Inventory {
	dir := cmd.Flag("inventory").Value.String()

	inventory := NewInventory()

	inventory.Vars["inventory_name"] = path.Base(dir)
	inventory.Vars["inventory_dir"] = dir
	inventory.Vars["inventory_file"] = dir + "/hosts"
	extra, _ := cmd.Flags().GetStringSlice("extra-vars")
	ParseExtraVars(extra, inventory)

	ParseInventory(dir, inventory)

	//var hosts = make(map[string]*Host)

	inventory.Merge()
	return inventory

}

func ParseInventory(dir string, inventory Inventory) {
	log.Infof("Parsing inventory: " + dir)

	ParseGroups(dir+"/group_vars", inventory)
	ParseGroupIni(dir, inventory)
}

func ParseExtraVars(extra []string, inventory Inventory) {
	for _, val := range extra {
		parts := strings.Split(val, "=")
		inventory.Vars[parts[0]] = parts[1]
	}
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

func FindImports(bytes []byte, inventory Inventory) map[string]interface{} {
	s := string(bytes[:])
	vars := make(map[string]interface{})

	for _, comment := range regexp.MustCompile("(?m)(# @import.*)").FindAllString(s, -1) {
		source := strings.Replace(comment, "# @import ", "", -1)
		path  := "/"
		if strings.Contains( source,"#") {
			path += strings.Split(source, "#")[1]
			source = strings.Split(source, "#")[0]
		}
		log.Infof("Import %s / %s", source,path)
		pwd, _ := os.Getwd()
		dst := pwd + "/.getter/" + regexp.MustCompile("[^0-9a-zA-z]*").ReplaceAllString(source, "")
		err := getter.Get(dst,source)
		if err != nil {
			log.Fatal("Error retrieving %s: \n %s", source, err)
			return vars
		}

		dst = dst + path
		if stat, err := os.Stat(dst); stat.IsDir() {
			ParseInventory(dst, inventory)
		} else if err == nil{
			PutAll(ParseFile(dst, inventory), vars)
		} else {
			log.Fatal("Sub file does not exist: " + dst  )

		}
	}
	return vars

}

func ParseFile(file string, inventory Inventory) map[string]interface{} {
	log.Infof("Parsing %s", file)
	bytes, err := ioutil.ReadFile(file)
	vars := FindImports(bytes, inventory)
	if err != nil {
		log.Error(err)
		return vars
	}
	yaml.Unmarshal(bytes, &vars)
	return vars
}
