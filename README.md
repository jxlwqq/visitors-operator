# visitors-operator

### 前置条件

* 安装 Docker Desktop，并启动内置的 Kubernetes 集群
* 注册一个 [hub.docker.com](https://hub.docker.com/) 账户，需要将本地构建好的镜像推送至公开仓库中
* 安装 operator SDK CLI: `brew install operator-sdk`
* 安装 Go: `brew install go`

本示例推荐的依赖版本：

* Docker Desktop: >= 4.0.0
* Kubernetes: >= 1.21.4
* Operator-SDK: >= 1.11.0
* Go: >= 1.17

### 创建项目

使用 Operator SDK CLI 创建名为 visitors-operator 的项目。
```shell
mkdir -p $HOME/projects/visitors-operator
cd $HOME/projects/visitors-operator
go env -w GOPROXY=https://goproxy.cn,direct

operator-sdk init \
--domain=jxlwqq.github.io \
--repo=github.com/jxlwqq/visitors-operator \
--skip-go-version-check
```


### 创建 API 和控制器

使用 Operator SDK CLI 创建自定义资源定义（CRD）API 和控制器。

运行以下命令创建带有组 cache、版本 v1alpha1 和种类 Memcached 的 API：

```shell
operator-sdk create api \
--resource=true \
--controller=true \
--group=app \
--version=v1alpha1 \
--kind=VisitorsApp
```

定义 VisitorsApp 自定义资源（CR）的 API。

修改 api/v1alpha1/visitorsapp_types.go 中的 Go 类型定义，使其具有以下 spec 和 status

```go
type VisitorsAppSpec struct {
	Size int32 `json:"size"`
	Title string `json:"title"`
}

type VisitorsAppStatus struct {
	BackendImage string `json:"backend_image"`
	FrontendImage string `json:"frontend_image"`
}
```