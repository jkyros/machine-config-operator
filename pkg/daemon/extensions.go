package daemon

import "io"
import "io/ioutil"
import "fmt"
import "encoding/json"

type SupportedExtensions struct {
	Extensions map[string]Extension `json:"extensions"`
	Repos      []string             `json:"repos"`
}

type Extension struct {
	Packages      []string `json:"packages"`
	Architectures []string `json:"architectures,omitEmpty"`
	Kind          string   `json:"kind"`
}

func parseSupportedExtensions(extensionsFile io.Reader) (map[string][]string, error) {

	if rawExtensions, err := ioutil.ReadAll(extensionsFile); err != nil {
		return nil, fmt.Errorf("failed to read extensions file %s : %s", extensionsFile, err)
	} else {
		var supportedExtensions *SupportedExtensions
	
		
		if err := json.Unmarshal(rawExtensions, &supportedExtensions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal extensions file %s : %s", extensionsFile, err)
		}

		//https://github.com/openshift/machine-config-operator/issues/2445 
		//mco should only pay attention to the  the os-extension kind 
		var extmap = make(map[string][]string)
		for extkey, ext := range supportedExtensions.Extensions {
                        if ext.Kind == "os-extension" {
                                extmap[extkey] = supportedExtensions.Extensions[extkey].Packages
                        }
                }
	

		return extmap, nil
	}
}

