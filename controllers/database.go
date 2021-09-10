package controllers

import (
	"context"
	appv1alpha1 "github.com/jxlwqq/visitors-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func mysqlAuthName() string {
	return "mysql-auth"
}

func mysqlDeploymentName() string {
	return "mysql"
}

func mysqlServiceName() string {
	return "mysql-svc"
}

func (r *VisitorsAppReconciler) mysqlAuthSecret(v *appv1alpha1.VisitorsApp) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      mysqlAuthName(),
		},
		StringData: map[string]string{
			"username": "visitors-user",
			"password": "visitors-pass",
		},
		Type: corev1.SecretTypeOpaque,
	}

	_ = controllerutil.SetControllerReference(v, secret, r.Scheme)

	return secret
}

func (r *VisitorsAppReconciler) mysqlDeployment(v *appv1alpha1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "mysql")
	size := int32(1)

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "username",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mysqlAuthName(),
			},
			Key: "password",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      mysqlDeploymentName(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "mysql:5.7",
						Name:  "visitor-mysql",
						Ports: []corev1.ContainerPort{{
							Name:          "mysql",
							ContainerPort: 3306,
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_ROOT_PASSWORD",
								Value: "password",
							},
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors",
							},
							{
								Name:      "MYSQL_USER",
								ValueFrom: userSecret,
							},
							{
								Name:      "MYSQL_PASSWORD",
								ValueFrom: passwordSecret,
							},
						},
					}},
				},
			},
		},
	}

	_ = controllerutil.SetControllerReference(v, dep, r.Scheme)

	return dep
}

func (r *VisitorsAppReconciler) mysqlService(v *appv1alpha1.VisitorsApp) *corev1.Service {
	labels := labels(v, "mysql")

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      mysqlServiceName(),
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port: 3306,
			}},
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	_ = controllerutil.SetControllerReference(v, svc, r.Scheme)

	return svc
}

func (r *VisitorsAppReconciler) isMysqlUp(v *appv1alpha1.VisitorsApp) bool {
	dep := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: v.Namespace,
		Name:      mysqlDeploymentName(),
	}, dep)
	if err != nil {
		return false
	}

	if dep.Status.Replicas == 1 {
		return true
	}

	return false
}
