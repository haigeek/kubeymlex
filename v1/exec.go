package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type resource struct {
	name string
	yaml string
}

func main() {
	// 定义需要排除的命名空间名称
	excludedNamespaces := []string{"kube-system", "kube-public", "kube-node-lease"}
	exportDir := "./k8s-export"

	namespacesCmd := exec.Command("kubectl", "get", "namespaces", "-o=jsonpath='{range .items[*]}{.metadata.name}{\"\\n\"}{end}'")

	namespacesOutput, err := namespacesCmd.Output()
	if err != nil {
		panic(err)
	}

	namespacesString := strings.Trim(string(namespacesOutput), "'")
	namespaces := strings.Split(strings.TrimSpace(string(namespacesString)), "\n")

	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		os.Mkdir(exportDir, 0755)
	}

	for _, ns := range namespaces {

		// 检查当前命名空间是否需要被排除
		exclude := false
		for _, excludedNs := range excludedNamespaces {
			if ns == excludedNs {
				exclude = true
				break
			}
		}

		if exclude {
			continue
		}

		nsDir := fmt.Sprintf("%s/%s", exportDir, ns)
		if _, err := os.Stat(nsDir); os.IsNotExist(err) {
			os.Mkdir(nsDir, 0755)
		}

		resourcesCmd := exec.Command("kubectl", "-n", ns, "get", "pvc,configmap,service,secret,deployment,statefulset,job,cronjob", "-o", "json")
		resourcesOutput, err := resourcesCmd.Output()
		if err != nil {
			panic(err)
		}

		resources, err := parseResources(resourcesOutput)
		if err != nil {
			panic(err)
		}

		for _, res := range resources {
			fileName := fmt.Sprintf("%s/%s.yaml", nsDir, res.name)
			err := ioutil.WriteFile(fileName, []byte(res.yaml), 0644)
			if err != nil {
				panic(err)
			}
		}

		fmt.Printf("Exported %d resources from namespace %s\n", len(resources), ns)
	}
}

func parseResources(jsonData []byte) ([]resource, error) {
	var resources []resource

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	for _, item := range data["items"].([]interface{}) {
		itemData := item.(map[string]interface{})

		kind := itemData["kind"].(string)
		name := itemData["metadata"].(map[string]interface{})["name"].(string)
		namespace := itemData["metadata"].(map[string]interface{})["namespace"].(string)

		if namespace == "" {
			continue
		}

		yamlData, err := exec.Command("kubectl", "get", kind, name, "-n", namespace, "-o", "yaml").Output()
		if err != nil {
			return nil, err
		}

		resources = append(resources, resource{
			name: fmt.Sprintf("%s-%s", kind, name),
			yaml: string(yamlData),
		})
	}

	return resources, nil
}
