package pkg

import (
	"strings"
	log "github.com/sirupsen/logrus"

	"github.com/levigross/grequests"
	"fmt"
	"time"
	"sort"
)

func (c Container) TagToSha() {

}

func (c Container) LatestToTag() {

}

type HarborImage struct {
	Digest  string    `json:"digest"`
	Name    string    `json:"name"`
	Created time.Time `json:"created" time_format:"2018-11-05T17:24:53.207123795Z"`
}

func LatestToTagHarbor(c *Container) {
	all := c.Group.Vars["latest_to_tag_harbor"] == "all"
	var tag string
	image := strings.Split(c.Image, ":")[0]
	if strings.Contains(c.Image, ":") {
		tag = strings.Split(c.Image, ":")[1]
	} else {
		tag = "latest"
	}

	if !all && tag != "latest" {
		log.Debugf("[%s] Skipping non-latest tag: %s", image, tag)
		return
	} else {
		registry := c.Group.Vars["docker_registry"].(string)
		host := strings.Split(registry, ":")[0]

		if !strings.Contains(registry, "/") && !strings.Contains(image, "/") {
			log.Errorf("[%s] Missing project in registry or image path", c.Image)
		}

		var project string
		if strings.Contains(registry, "/") {
			project = strings.Split(registry, "/")[1]
			host = strings.Split(registry, "/")[0]
		} else {
			project = strings.Split(image, "/")[0]
			image = strings.Join(strings.Split(image, "/")[1:], "/")
		}

		api := fmt.Sprintf("https://%s/api/repositories/%s/%s/tags", host, project, image)
		log.Debugf("[%s] Looking up latest using %s", image, api)

		response, err := grequests.Get(api, nil)
		if err != nil {
			panic(err)
		}
		imgs := []HarborImage{}
		response.JSON(&imgs)

		if len(imgs) == 0 {
			log.Errorf("[%s] No tags found", c.Image)
		} else {
			sort.Slice(imgs, func(i int, j int) bool {
				return imgs[j].Created.Before(imgs[i].Created)
			})
			log.Infof("[%s] Found tag %s created %s", image, imgs[0].Name, imgs[0].Created)
			c.Image = strings.Split(image, ":")[0] + ":" + imgs[0].Name
		}
	}

}
