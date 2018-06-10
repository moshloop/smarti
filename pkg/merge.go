package pkg

import (
	"github.com/flosch/pongo2"
	"strings"
	"reflect"
	log "github.com/sirupsen/logrus"
)

//http://docs.ansible.com/ansible/latest/user_guide/playbooks_variables.html
//

// all
// group
// extra varrs
//role defaults [1]
//inventory file or script group vars [2]
//inventory group_vars/all [3]
//playbook group_vars/all [3]
//inventory group_vars/* [3]
//playbook group_vars/* [3]
//inventory file or script host vars [2]
//inventory host_vars/*
//playbook host_vars/*
//host facts / cached set_facts [4]
//inventory host_vars/* [3]
//playbook host_vars/* [3]
//host facts
//play vars
//play vars_prompt
//play vars_files
//role vars (defined in role/vars/main.yml)
//block vars (only for tasks in block)
//task vars (only for the task)
//include_vars
//set_facts / registered vars
//role (and include_role) params
//include params
//extra vars (always win precedence)

func ToGenericMap(m map[string]string) map[string]interface{} {
	var out = map[string]interface{}{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func ConvertSyntaxFromJinjaToPongo(template string) string {
	// jinja used filter(arg), pongo uses filter:arg
	template = strings.Replace(template, "(", ":", -1)
	template = strings.Replace(template, ")", "", -1)
	return template
}

func InterpolateString(template string, vars map[string]interface{}) string {
	if strings.Contains(template, "lookup(") {
		log.Warningf("ansible lookups not supported %s", template)
		return template
	}

	template = ConvertSyntaxFromJinjaToPongo(template)
	tpl, err := pongo2.FromString(template)
	if err != nil {
		log.Debugf("Error parsing: %s: %v", template, err)
		return template
	}
	out, err := tpl.Execute(vars)
	if err != nil {
		log.Debugf("Error parsing: %s: %v", template, err)
		return template
	}
	//log.Errorf("%s => %s", template, out)
	return out
}

func Interpolate(key interface{}, vars map[string]interface{}) interface{} {
	switch v := key.(type) {
	case string:
		return InterpolateString(v, vars)
	case []interface{}:
		var out []interface{}
		for _, val := range v {
			out = append(out, Interpolate(val, vars))
		}
		return out
	case map[string]interface{}:
		for subkey, val := range v {
			v[subkey] = Interpolate(val, vars)
		}
		return v

	case map[string]string:
		var out = map[string]interface{}{}
		for subkey, val := range v {
			out[subkey] = Interpolate(val, vars)
		}
		return out

	case map[interface{}]interface{}:
		var out = map[string]interface{}{}
		for subkey, val := range v {
			out[subkey.(string)] = Interpolate(val, vars)
		}
		return out

	default:
		log.Warningf("Unknown type: %s", reflect.TypeOf(key))
		return key
	}
}

func PutAll(src map[string]interface{}, dst map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func InterpolateGroups(groups map[string]Group) {
	for _, group := range groups {
		for key, value := range group.Vars {
			group.Vars[key] = Interpolate(value, group.Vars)
		}
	}
}
