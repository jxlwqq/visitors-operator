package controllers

import (
	"context"
	appv1alpha1 "github.com/jxlwqq/visitors-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const backendImage = "jdob/visitors-service:1.0.0"
const backendPort = 8000
const backendServicePort = 30685

func backendDeploymentName(v *appv1alpha1.VisitorsApp) string {
	return v.Name + "-backend"
}

func backendServiceName(v *appv1alpha1.VisitorsApp) string {
	return v.Name + "-backend-svc"
}

func (r *VisitorsAppReconciler) backendDeployment(v *appv1alpha1.VisitorsApp) *appsv1.Deployment {
	size := v.Spec.Size
	labels := labels(v, "backend")

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "username",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "password",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      backendDeploymentName(v),
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
						Name:  "visitors-service",
						Image: backendImage,
						Ports: []corev1.ContainerPort{{
							ContainerPort: backendPort,
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors",
							},
							{
								Name:  "MYSQL_SERVICE_HOST",
								Value: mysqlServiceName(),
							},
							{
								Name:      "MYSQL_USERNAME",
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

func (r *VisitorsAppReconciler) backendService(v *appv1alpha1.VisitorsApp) *corev1.Service {
	labels := labels(v, "backend")

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      backendServiceName(v),
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       backendPort,
				TargetPort: intstr.FromInt(backendPort),
				NodePort:   backendServicePort,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	_ = controllerutil.SetControllerReference(v, svc, r.Scheme)

	return svc
}

func (r *VisitorsAppReconciler) updateBackendStatus(v *appv1alpha1.VisitorsApp) error {
	v.Status.BackendImage = backendImage
	err := r.Client.Status().Update(context.TODO(), v)
	return err
}

func (r *VisitorsAppReconciler) handleBackendChanges(v *appv1alpha1.VisitorsApp) (*ctrl.Result, error) {
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: v.Namespace,
		Name:      backendDeploymentName(v),
	}, found)

	if err != nil {
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	size := v.Spec.Size
	if size != *found.Spec.Replicas {
		found.Spec.Replicas = &size
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			return &ctrl.Result{}, err
		}
	}

	return nil, nil
}
