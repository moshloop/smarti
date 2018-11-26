package pkg

import (
	"regexp"
	"strings"
	"os"
	"github.com/knq/ini"
	"github.com/ghodss/yaml"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/go-getter"

	"encoding/json"
)

func FindImports(bytes []byte, inventory Inventory) map[string]interface{} {
	s := string(bytes[:])
	vars := make(map[string]interface{})

	for _, comment := range regexp.MustCompile("(?m)(# @import.*)").FindAllString(s, -1) {
		source := strings.Replace(comment, "# @import ", "", -1)
		path := "/"
		if strings.Contains(source, "#") {
			path += strings.Split(source, "#")[1]
			source = strings.Split(source, "#")[0]
		}
		log.Infof("Import %s / %s", source, path)
		pwd, _ := os.Getwd()
		dst := pwd + "/.getter/" + regexp.MustCompile("[^0-9a-zA-z]*").ReplaceAllString(source, "")
		err := getter.Get(dst, source)
		if err != nil {
			log.Fatal("Error retrieving %s: \n %s", source, err)
			return vars
		}

		dst = dst + path
		if stat, err := os.Stat(dst); stat.IsDir() {
			ParseInventory(dst, inventory)
		} else if err == nil {
			PutAll(ParseFile(dst, inventory), vars)
		} else {
			log.Fatal("Sub file does not exist: " + dst)

		}
	}
	return vars

}

func ParseFile(file string, inventory Inventory) map[string]interface{} {
	log.Debugf("Parsing %s", file)
	bytes, err := ioutil.ReadFile(file)
	vars := FindImports(bytes, inventory)
	if err != nil {
		log.Error(err)
		return vars
	}

	if strings.HasSuffix(file, "json") {
		json.Unmarshal(bytes, &vars)
	} else if strings.HasSuffix(file, "properties") || strings.HasSuffix(file, "ini") {

		cfg, err := ini.LoadString(string(bytes))
		if err != nil {
			panic(err)
		}
		for k,v := range cfg.GetMapFlat() {
			vars[k] = string(v)
		}
	} else {
		yaml.Unmarshal(bytes, &vars)

	}

	return vars
}
