package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	ExportPath   string   `yaml:"exportPath"`
	PvExportPath string   `yaml:"pvExportPath"`
	ExportModel  int      `yaml:"exportModel"`
	ExcludedNs   []string `yaml:"excludedNamespaces"`
	IncludedNs   []string `yaml:"includedNamespaces"`
	Resource     []string `yaml:"resources"`
}

type resource struct {
	name string
	yaml string
}

var configPath = "config.yaml"

var (
	export = flag.String("export", "", "导出资源 all/pv")
	// start = flag.String("start", "", "启动镜像，全部-start all ,多个用空格隔开，example: -start mongo-gui pgadmin")
)

func main() {
	// 读取配置文件

	config := readConfig(configPath)
	// 获取需要导出的命名空间列表
	//namespaces := []string{"default", "bincloud"}
	namespaces := getNamespaces(config.IncludedNs, config.ExcludedNs)

	// 导出指定命名空间的资源到文件夹
	// 如果 exportPath 为空，则设置默认值
	if config.ExportPath == "" {
		config.ExportPath = "./k8s-export"
	}
	if config.PvExportPath == "" {
		config.PvExportPath = config.ExportPath + "/pv"
	}

	flag.Parse()

	//导出所有pv
	if *export == "pv" {
		exportPv(config.PvExportPath)
		return
	}
	//导出所有资源
	if *export == "all" {
		exportResources(config.Resource, namespaces, config.ExportPath, config.ExportModel)
		return
	}

}

func readConfig(configPath string) Config {
	// 读取配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	// 解析配置文件
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	return config
}

func getNamespaces(includedNs, excludedNs []string) []string {
	// 获取所有命名空间列表
	cmd := exec.Command("kubectl", "get", "namespaces", "-o=jsonpath='{range .items[*]}{.metadata.name}{\"\\n\"}{end}'")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("failed to get namespaces: %v", err)
	}

	// 解析命名空间列表
	namespacesString := strings.Trim(string(out), "'")
	allNs := strings.Split(strings.TrimSpace(string(namespacesString)), "\n")

	// 选取需要导出的命名空间
	var namespaces []string
	if len(includedNs) > 0 {
		// 如果配置文件中指定了需要导出的命名空间，则按照配置文件中的配置进行导出
		for _, ns := range includedNs {
			if contains(allNs, ns) {
				namespaces = append(namespaces, ns)
			} else {
				log.Printf("namespace %s does not exist, skipping...", ns)
			}
		}
	} else {
		// 如果配置文件中没有指定需要导出的命名空间，则导出所有命名空间
		for _, ns := range allNs {
			if !contains(excludedNs, ns) {
				namespaces = append(namespaces, ns)
			}
		}
	}

	return namespaces
}

func exportResources(Resource, namespaces []string, exportPath string, exportModel int) {
	// 创建保存资源的文件夹
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		err = os.MkdirAll(exportPath, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to create export directory: %v", err)
		}
	}
	//备份模式
	//单个资源逐个备份方式
	if exportModel == 1 {
		// 遍历命名空间和资源类型，导出指定资源到文件夹
		for _, ns := range namespaces {
			//创建命名空间文件夹
			nsDir := fmt.Sprintf("%s/%s", exportPath, ns)
			if _, err := os.Stat(nsDir); os.IsNotExist(err) {
				os.Mkdir(nsDir, 0755)
			}
			for _, obj := range Resource {
				// cmd := exec.Command("kubectl", "get", obj, "-n", ns, "-o=yaml")
				cmd := exec.Command("kubectl", "-n", ns, "get", obj, "-o", "json")
				out, err := cmd.Output()
				if err != nil {
					log.Printf("failed to get %s in namespace %s: %v", obj, ns, err)
					continue
				}

				resources, err := parseResources(out)
				if err != nil {
					panic(err)
				}

				// 将资源保存到文件
				for _, res := range resources {
					fileName := fmt.Sprintf("%s/%s.yaml", nsDir, res.name)
					err := ioutil.WriteFile(fileName, []byte(res.yaml), 0644)
					if err != nil {
						log.Printf("failed to write %s in namespace %s to file: %v", res.name, ns, err)
					}
					log.Printf("Exported %s in %s to %s", res.name, ns, fileName)
				}
			}
		}
	}

	//按照类型备份方式
	if exportModel == 2 {
		for _, ns := range namespaces {
			for _, obj := range Resource {
				cmd := exec.Command("kubectl", "get", obj, "-n", ns, "-o=yaml")
				out, err := cmd.Output()
				if err != nil {
					log.Printf("failed to get %s in namespace %s: %v", obj, ns, err)
					continue
				}
				// 将资源保存到文件
				filename := fmt.Sprintf("%s/%s-%s.yaml", exportPath, ns, obj)
				err = ioutil.WriteFile(filename, out, 0644)
				if err != nil {
					log.Printf("failed to write %s in namespace %s to file: %v", filename, ns, err)
				}
				log.Printf("Exported %s in %s to %s", obj, ns, filename)
			}
		}
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

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func exportPv(exportPath string) {
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		err = os.MkdirAll(exportPath, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to create export directory: %v", err)
		}
	}
	// 获取所有 PV 名称
	out, err := exec.Command("kubectl", "get", "pv", "-o", "jsonpath={range .items[*]}{.metadata.name}{'\\n'}{end}").Output()
	if err != nil {
		panic(err)
	}

	// 将 PV 名称存储到切片中
	pvNames := strings.Split(string(out), "\n")

	// 移除第一个和最后一个元素（可能是空字符串）
	if len(pvNames) > 0 {
		pvNames = pvNames[1:]
	}
	if len(pvNames) > 0 {
		pvNames = pvNames[:len(pvNames)-1]
	}

	// 遍历每个 PV，并导出到 YAML 文件中
	for _, pvName := range pvNames {
		// 使用 kubectl 导出 PV YAML
		out, err := exec.Command("kubectl", "get", "pv", pvName, "-o", "yaml").Output()
		if err != nil {
			panic(err)
		}

		// 将 YAML 内容存储到文件中
		filename := fmt.Sprintf("%s/%s.yaml", exportPath, pvName)
		err = ioutil.WriteFile(filename, out, 0644)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Exported PV %s to file %s.yaml\n", pvName, pvName)
	}
}
