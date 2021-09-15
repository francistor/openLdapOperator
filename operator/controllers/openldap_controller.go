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
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	openldapv1alpha1 "ldapOperator/api/v1alpha1"

	"fmt"
)

// OpenldapReconciler reconciles a Openldap object
type OpenldapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openldap.minsait.com,resources=openldaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.minsait.com,resources=openldaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.minsait.com,resources=openldaps/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Openldap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *OpenldapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	log.Info("Reconcile", fmt.Sprintf("%v", ctx), fmt.Sprintf("%v", req))

	openldap := &openldapv1alpha1.Openldap{}
	if err := r.Get(ctx, req.NamespacedName, openldap); err != nil {
		// Ignore this type of errors
		if errors.IsNotFound(err) {
			log.Info("Openldap object not found. Ignoring, since it might be deleted")
			return ctrl.Result{}, nil
		}
		// Requeue
		log.Error(err, "Error looding for Openldap object")
		return ctrl.Result{}, err
	}

	// Create Deployment if it does not exist
	existingDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: openldap.Name, Namespace: openldap.Namespace}, existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		// Create a new Deployment
		deployment := r.deploymentForOpenldap(openldap)
		log.Info("About to create a deployment for Openldap", "Deployment.Namespace:", openldap.Namespace, "Deployment.Name", openldap.Name, "Image", openldap.Spec.Image)
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed creating deployment for Openldap", "Deployment.Namespace:", openldap.Namespace, "Deployment.Name", openldap.Name)
			return ctrl.Result{}, err
		}
		// Deployment created. Return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get deployment")
		return ctrl.Result{}, err
	}

	// Check and adjust size if necessary
	desiredSize := openldap.Spec.Size
	if *existingDeployment.Spec.Replicas != desiredSize {
		existingDeployment.Spec.Replicas = &desiredSize
		if err = r.Update(ctx, existingDeployment); err != nil {
			log.Error(err, "Failed updating deployment for ldap", "Deployment.Namespace:", openldap.Namespace, "Deployment.Name", openldap.Name)
			return ctrl.Result{}, err
		}

		// Requeue after some time
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Create service if it does not exist
	existingService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: openldap.Name + "-service", Namespace: openldap.Namespace}, existingService)
	if err != nil && errors.IsNotFound(err) {
		// Create the service
		service := r.serviceForOpenldap(openldap)
		log.Info("About to create service for Openldap")
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Failed creating service for Openldap")
			return ctrl.Result{}, err
		}

		// Service created. Return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get service")
		return ctrl.Result{}, err
	}

	// Update status with pod names
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(openldap.Namespace),
		client.MatchingLabels(map[string]string{"app": "openldap", "openldap_cr": openldap.Name}),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed listing pods", "Deployment.Namespace:", openldap.Namespace, "Deployment.Name", openldap.Name)
		return ctrl.Result{}, err
	}
	var podNames []string
	for _, pod := range podList.Items {
		podNames = append(podNames, pod.Name)
	}

	// Update if required
	if !reflect.DeepEqual(openldap.Status.Nodes, podNames) {
		openldap.Status.Nodes = podNames
		if err := r.Status().Update(ctx, openldap); err != nil {
			log.Error(err, "Could not update status", "Deployment.Namespace:", openldap.Namespace, "Deployment.Name", openldap.Name)
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenldapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openldapv1alpha1.Openldap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// Creates the deployment
func (r *OpenldapReconciler) deploymentForOpenldap(openldap *openldapv1alpha1.Openldap) *appsv1.Deployment {
	labels := map[string]string{"app": "openldap", "openldap_cr": openldap.Name}
	replicas := openldap.Spec.Size
	image := openldap.Spec.Image

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      openldap.Name,
			Namespace: openldap.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   image,
						Name:    "managedopenldap",
						Command: []string{"/usr/local/libexec/slapd", "-F", "/usr/local/etc/openldap/slapd.d", "-h", "ldap:/// ldapi:///", "-d", "stats"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 389,
							Name:          "ldap",
						}},
					}},
				},
			},
		},
	}

	ctrl.SetControllerReference(openldap, deployment, r.Scheme)
	return deployment
}

// Creates the service
func (r *OpenldapReconciler) serviceForOpenldap(openldap *openldapv1alpha1.Openldap) *corev1.Service {
	labels := map[string]string{"app": "openldap"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      openldap.Name + "-service",
			Namespace: openldap.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name:     "ldap",
				Protocol: "TCP",
				Port:     389,
			}},
		},
	}

	ctrl.SetControllerReference(openldap, service, r.Scheme)
	return service
}
