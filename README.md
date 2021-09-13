# visitors-operator

> visitors-operator 是书籍《Kubernetes Operators》的经典示例，此书著于2019年，Operator-SDK 版本迭代较快，随书的代码仓库
> https://github.com/kubernetes-operators-book/chapters 已基本不可用。所以本示例基于最新版本的 Operator-SDK，做了大量的修改和调整。

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

运行以下命令创建带有组 cache、版本 v1alpha1 和种类 VisitorsApp 的 API：

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

为资源类型更新生成的代码：
```shell
make generate
```

运行以下命令以生成和更新 CRD 清单：
```shell
make manifests
```

### 实现控制器

> 由于逻辑较为复杂，代码较为庞大，所以无法在此全部展示，完整的操作器代码请参见 controllers 目录。

在本例中，将生成的控制器文件 controllers/visitors_controller.go 替换为以下示例实现：

```go
/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/jxlwqq/visitors-operator/api/v1alpha1"
)

// VisitorsAppReconciler reconciles a VisitorsApp object
type VisitorsAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=visitorsapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=visitorsapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=visitorsapps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VisitorsApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *VisitorsAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling VisitorsApp")

	// Fetch the VisitorsApp instance
	v := &appv1alpha1.VisitorsApp{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, v)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var result *ctrl.Result

	// Database
	result, err = r.ensureSecret(r.mysqlAuthSecret(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(r.mysqlDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(r.mysqlService(v))
	if result != nil {
		return *result, err
	}

	if !r.isMysqlUp(v) {
		delay := time.Second * time.Duration(5)
		log.Info(fmt.Sprintf("MySQL isn't running, waiting for %s", delay))
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// Backend
	result, err = r.ensureDeployment(r.backendDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(r.backendService(v))
	if result != nil {
		return *result, err
	}

	err = r.updateBackendStatus(v)
	if err != nil {
		return ctrl.Result{}, err
	}

	result, err = r.handleBackendChanges(v)
	if result != nil {
		return *result, err
	}

	// Frontend
	result, err = r.ensureDeployment(r.frontendDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(r.frontendService(v))
	if result != nil {
		return *result, err
	}

	err = r.updateFrontendStatus(v)
	if err != nil {
		return ctrl.Result{}, err
	}

	result, err = r.handleFrontendChanges(v)
	if result != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VisitorsAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.VisitorsApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
```


运行以下命令以生成和更新 CRD 清单：
```shell
make manifests
```

### 运行 Operator

捆绑 Operator，并使用 Operator Lifecycle Manager（OLM）在集群中部署。

修改 Makefile 中 IMAGE_TAG_BASE 和 IMG：

```makefile
IMAGE_TAG_BASE ?= docker.io/jxlwqq/visitors-operator
IMG ?= $(IMAGE_TAG_BASE):latest
```

构建镜像：

```shell
make docker-build
```

将镜像推送到镜像仓库：
```shell
make docker-push
```

成功后访问：https://hub.docker.com/r/jxlwqq/visitors-operator


运行 make bundle 命令创建 Operator 捆绑包清单，并依次填入名称、作者等必要信息:
```shell
make bundle
```

构建捆绑包镜像：
```shell
make bundle-build
```

推送捆绑包镜像：
```shell
make bundle-push
```

成功后访问：https://hub.docker.com/r/jxlwqq/visitors-operator-bundle


使用 Operator Lifecycle Manager 部署 Operator:

```shell
# 切换至本地集群
kubectl config use-context docker-desktop
# 安装 olm
operator-sdk olm install
# 使用 Operator SDK 中的 OLM 集成在集群中运行 Operator
operator-sdk run bundle docker.io/jxlwqq/visitors-operator-bundle:v0.0.1
```


### 创建自定义资源

编辑 config/samples/app_v1alpha1_visitorsapp.yaml 上的 VisitorsApp CR 清单示例，使其包含以下规格：

```yaml
apiVersion: app.jxlwqq.github.io/v1alpha1
kind: VisitorsApp
metadata:
  name: visitorsapp-sample
spec:
  # Add fields here
  size: 1
  title: Hello
```

创建 CR：
```shell
kubectl apply -f config/samples/app_v1alpha1_visitorsapp.yaml
```

查看 Pod：
```shell
NAME                                           READY   STATUS    RESTARTS   AGE
mysql-bbdc8b45f-mxt4l                          1/1     Running   0          11s
visitorsapp-sample-backend-5f954fcc56-mm8bp    1/1     Running   0          11s
visitorsapp-sample-frontend-6b666b55c7-hl4ch   1/1     Running   0          11s
```

查看 Service：
```shell
NAME                              TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
kubernetes                        ClusterIP   10.96.0.1        <none>        443/TCP          18h
mysql-svc                         ClusterIP   None             <none>        3306/TCP         52s
visitorsapp-sample-backend-svc    NodePort    10.105.131.165   <none>        8000:30685/TCP   52s
visitorsapp-sample-frontend-svc   NodePort    10.99.93.124     <none>        3000:30686/TCP   52s
```

浏览器访问：http://localhost:30686

网页上会显示以下内容，记录了访问者的访问记录：
```text
Hello

Service IP	Client IP	Timestamp
10.1.1.108	192.168.65.3	2021/9/10 上午11:06:40
10.1.1.108	192.168.65.3	2021/9/10 上午11:06:38
```


### 做好清理

```shell
operator-sdk cleanup visitors-operator
operator-sdk olm uninstall
```

### 更多

更多经典示例请参考：https://github.com/jxlwqq/kubernetes-examples
