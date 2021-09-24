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
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pvcs,verbs=get;list;watch;create;update;patch;delete

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

	// Get the openldap object
	openldap := &openldapv1alpha1.Openldap{}
	if err := r.Get(ctx, req.NamespacedName, openldap); err != nil {
		// Ignore this type of errors
		if errors.IsNotFound(err) {
			log.Info("Openldap object not found. Ignoring, since it might be deleted")
			return ctrl.Result{}, nil
		}
		// Requeue
		log.Error(err, "Error looking for Openldap object")
		return ctrl.Result{}, err
	}

	// Create PVC if it does not exit
	existingPVC := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: "openldap-" + openldap.Name, Namespace: openldap.Namespace}, existingPVC)
	if err != nil && errors.IsNotFound(err) {
		pvc := r.pvcForOpenLdap(openldap)
		log.Info("About to create a PVC for Openldap")
		if err := r.Create(ctx, pvc); err != nil {
			log.Error(err, "Error creating PVC")
			return ctrl.Result{}, err
		}
		// PVC created. Return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get PVC")
		return ctrl.Result{}, err
	} else {
		// Check sizeRequests
		if existingPVC.Spec.Resources.Requests["storage"] != openldap.Spec.StorageSize {
			log.Error(err, "Existing PVC size does not match the requested one")
			log.Info(fmt.Sprintf("#%v, #%v", existingPVC.Spec.Resources.Requests["storage"], openldap.Spec.StorageSize))
			return ctrl.Result{}, err
		}
	}

	// Create Pod if it does not exist
	existingPod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: "openldap-" + openldap.Name, Namespace: openldap.Namespace}, existingPod)
	if err != nil && errors.IsNotFound(err) {
		pod := r.podForOpenldap(openldap)
		log.Info("About to create a Pod for Openldap")
		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "Error creating Pod")
			return ctrl.Result{}, err
		}
		// Pod created. Return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	// Create service if it does not exist
	existingService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "openldap-" + openldap.Name, Namespace: openldap.Namespace}, existingService)
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
		Owns(&corev1.Pod{}).Owns(&corev1.Service{}).Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}

// Creates the ldap pod
func (r *OpenldapReconciler) podForOpenldap(openldap *openldapv1alpha1.Openldap) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openldap-" + openldap.Name,
			Namespace: openldap.Namespace,
			Labels:    map[string]string{"app": "openldap", "openldap": openldap.Name},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "openldap-" + openldap.Name,
				Image:   openldap.Spec.Image,
				Command: []string{"/usr/local/libexec/slapd", "-F", "/usr/local/etc/openldap/slapd.d", "-h", "ldap:/// ldapi:///", "-d", "stats"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "ldap-database-volume",
					MountPath: "/usr/local/var/openldap-data",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "ldap-database-volume",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "openldap-" + openldap.Name,
					},
				},
			}},
		},
	}
	ctrl.SetControllerReference(openldap, pod, r.Scheme)
	return pod
}

// Creates the PVC
func (r *OpenldapReconciler) pvcForOpenLdap(openldap *openldapv1alpha1.Openldap) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openldap-" + openldap.Name,
			Namespace: openldap.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": openldap.Spec.StorageSize,
				},
			},
		},
	}

	// Mark for deletion when main object is erased if so specified
	if openldap.Spec.DisposePVC {
		ctrl.SetControllerReference(openldap, pvc, r.Scheme)
	}

	return pvc
}

// Creates the load balancer service
func (r *OpenldapReconciler) serviceForOpenldap(openldap *openldapv1alpha1.Openldap) *corev1.Service {
	labels := map[string]string{"openldap": openldap.Name}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openldap-" + openldap.Name,
			Namespace: openldap.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name:     "ldap",
				Protocol: "TCP",
				Port:     389,
			}},
			Type: "LoadBalancer",
		},
	}

	if openldap.Spec.LoadBalancerIPAddress != "" {
		service.Spec.LoadBalancerIP = openldap.Spec.LoadBalancerIPAddress
	}

	ctrl.SetControllerReference(openldap, service, r.Scheme)
	return service
}
