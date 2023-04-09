## 使用说明

### 导出全部资源

```
kubeymlex -export all
```

### 导出pv类型的资源

> k8s中pv卷没有namespace的概念，因此使用单独的命令直接导出全部pv卷

```
kubeymlex -export pv
```

### 配置说明

`exportModel`:资源的备份模式

1: 单个资源逐个备份方式
按照命名空间创建文件夹，按照类型+资源名的形式导出。
等效于：

```
kubectl get <resourcetype> --namespace xxx
```

eg:

```
├── namespace-1
│   ├── ConfigMap-kube-root-ca.crt.yaml
│   ├── Deployment-nfs-client-provisioner.yaml
│   ├── PersistentVolumeClaim-tbe-chaos-claim.yaml
│   ├── Service-kubernetes.yaml
│   └── Service-tfe-facemap.yaml
├── namespace-2
│   ├── ConfigMap-kube-root-ca.crt.yaml
│   ├── Deployment-tbe-chaos.yaml
│   ├── Deployment-tbe-chaos.yaml
│   ├── Service-tbe-chaos.yaml

```

2:按照类型备份方式
导出到一个文件夹，按照命名空间+资源类型的形式导出.此模式会将命名空间下某一类型的资源全部导出到一个文件中
eg：

```
├── k8s-export
│   ├── namespace-ConfigMap.yaml
│   ├── namespace-Deployment.yaml
│   ├── namespace-PersistentVolumeClaim.yaml
│   └── namespace-Service.yaml
```

## build

- amd64编译

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/kubeymlex-amd64-linux main.go
```

- arm编译

```bash
##注意根据 arm 的平台动态配置 GOARCH
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/kubeymlex-arm64-linux main.go
```

- win编译

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/kubeymlex-amd64-windows.exe main.go
```


