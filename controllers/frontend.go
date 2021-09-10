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

const frontendPort = 3000
const frontendServicePort = 30686
const frontendImage = "jdob/visitors-webui:1.0.0"

func frontendDeploymentName(v *appv1alpha1.VisitorsApp) string {
	return v.Name + "-frontend"
}

func frontendServiceName(v *appv1alpha1.VisitorsApp) string {
	return v.Name + "-frontend-svc"
}

func (r *VisitorsAppReconciler) frontendDeployment(v *appv1alpha1.VisitorsApp) *appsv1.Deployment {
	size := int32(1)
	labels := labels(v, "frontend")
	var env []corev1.EnvVar
	if v.Spec.Title != "" {
		env = append(env, corev1.EnvVar{
			Name:  "REACT_APP_TITLE",
			Value: v.Spec.Title,
		})
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      frontendDeploymentName(v),
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
						Name:  "visitors-webui",
						Image: frontendImage,
						Ports: []corev1.ContainerPort{{
							ContainerPort: frontendPort,
						}},
						Env: env,
					}},
				},
			},
		},
	}

	_ = controllerutil.SetControllerReference(v, dep, r.Scheme)

	return dep
}

func (r *VisitorsAppReconciler) frontendService(v *appv1alpha1.VisitorsApp) *corev1.Service {
	labels := labels(v, "frontend")

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v.Namespace,
			Name:      frontendServiceName(v),
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       frontendPort,
				TargetPort: intstr.FromInt(frontendPort),
				NodePort:   frontendServicePort,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	_ = controllerutil.SetControllerReference(v, svc, r.Scheme)

	return svc
}

func (r *VisitorsAppReconciler) updateFrontendStatus(v *appv1alpha1.VisitorsApp) error {
	v.Status.FrontendImage = frontendImage
	err := r.Client.Update(context.TODO(), v)
	return err
}

func (r *VisitorsAppReconciler) handleFrontendChanges(v *appv1alpha1.VisitorsApp) (*ctrl.Result, error) {
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: v.Namespace,
		Name:      frontendDeploymentName(v),
	}, found)
	if err != nil {
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	title := v.Spec.Title
	existing := (*found).Spec.Template.Spec.Containers[0].Env[0].Value

	if title != existing {
		(*found).Spec.Template.Spec.Containers[0].Env[0].Value = title
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			return &ctrl.Result{}, err
		}

		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}
