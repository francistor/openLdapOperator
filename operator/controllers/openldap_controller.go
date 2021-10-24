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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
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

	// Added for execution of commands
	RESTClient rest.Interface
	RESTConfig *rest.Config
}

//+kubebuilder:rbac:groups=openldap.minsait.com,resources=openldaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.minsait.com,resources=openldaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.minsait.com,resources=openldaps/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

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

	// Create or update the Configmap with LDAP configuration
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: "openldap-" + openldap.Name, Namespace: openldap.Namespace}, existingConfigMap)
	if err != nil && errors.IsNotFound(err) {
		// ConfigMap does not exist. Create it
		configMap := r.configMapForOpenldap((openldap))
		log.Info("About to create ConfigMap for Openldap")
		if err := r.Create(ctx, configMap); err != nil {
			log.Error(err, "Error creating Configmap")
			return ctrl.Result{}, err
		}
		// Configmap created. Return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Configmap")
		return ctrl.Result{}, err
	} else {
		// Update configuration if it has changed in CR
		if existingConfigMap.Data["slapd.conf"] != openldap.Spec.Config {
			existingConfigMap.Data["slapd.conf"] = openldap.Spec.Config
			log.Info("About to change configmap")
			err := r.Update(ctx, existingConfigMap)

			if err != nil {
				log.Error(err, "Could not update configuration")
				return ctrl.Result{}, err
			}

			// Force update of configuration, since configmap contents will not be applied as configuration in the Openldap image
			// This is done by executing a command in the pod, which takes the new config to apply through stdin
			// https://github.com/kubernetes-sigs/kubebuilder/issues/803
			existingPod := &corev1.Pod{}
			err = r.Get(ctx, types.NamespacedName{Name: "openldap-" + openldap.Name, Namespace: openldap.Namespace}, existingPod)
			if err != nil && errors.IsNotFound(err) {
				log.Info("The openldap pod does not exist yet")
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			} else if err != nil {
				log.Error(err, "Could not get the openLdap pod")
				return ctrl.Result{}, err
			}

			req := r.RESTClient.Post().
				Namespace(existingPod.Namespace).
				Resource("pods").
				Name(existingPod.Name).
				SubResource("exec").
				VersionedParams(&corev1.PodExecOptions{
					Container: existingPod.Spec.Containers[0].Name,
					Command:   []string{"/ldifCompare/bin/updateLdapConfig.sh"},
					Stdin:     true,
					Stdout:    true,
					Stderr:    true,
					TTY:       false,
				}, runtime.NewParameterCodec(r.Scheme))

			exec, err := remotecommand.NewSPDYExecutor(r.RESTConfig, "POST", req.URL())
			if err != nil {
				log.Error(err, "Could not build the remote command executor")
				return ctrl.Result{}, err
			}

			in := strings.NewReader(openldap.Spec.Config)
			out := strings.Builder{}
			eout := strings.Builder{}

			// Connect this process' std{in,out,err} to the remote shell process.
			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  in,
				Stdout: &out,
				Stderr: &eout,
				Tty:    false,
			})

			if err != nil {
				log.Error(err, "Could not execute update comand in LDAP pod")
				return ctrl.Result{}, err
			}

			log.Info("Update Command executed")
			log.Info("Stdout: " + out.String())
			log.Info("Stderr: " + eout.String())

			// Give some time to have the configmap update
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	// Create PVC if it does not exit
	existingPVC := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: "openldap-" + openldap.Name, Namespace: openldap.Namespace}, existingPVC)
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
			log.Error(err, "Existing PVC size does not match the requested one. You should consider deleting the existing PVC")
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
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
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
		client.MatchingLabels(map[string]string{"app": "openldap", "openldap": openldap.Name}),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed listing pods", "Namespace:", openldap.Namespace, "Name", openldap.Name)
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
			log.Error(err, "Could not update status", "Namespace:", openldap.Namespace, "Name", openldap.Name)
			return ctrl.Result{}, err
		}

	}

	log.Info("Nothing to Reconcile")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenldapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openldapv1alpha1.Openldap{}).
		// TODO: Check what happens if I remove some of the Owns
		Owns(&corev1.Pod{}).Owns(&corev1.Service{}).Owns(&corev1.PersistentVolumeClaim{}).Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// Creates the configmap
func (r *OpenldapReconciler) configMapForOpenldap(openldap *openldapv1alpha1.Openldap) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openldap-" + openldap.Name,
			Namespace: openldap.Namespace,
		},
		Data: map[string]string{
			"slapd.conf": openldap.Spec.Config,
		},
	}
	ctrl.SetControllerReference(openldap, configMap, r.Scheme)
	return configMap
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
				Name:  "openldap-" + openldap.Name,
				Image: openldap.Spec.Image,
				Command: []string{
					"/bin/sh",
					"-c",
					"slaptest -n 0 -f /usr/local/etc/openldap/slapd.conf -F /usr/local/etc/openldap/slapd.d && /usr/local/libexec/slapd -F /usr/local/etc/openldap/slapd.d -h \"ldap:/// ldapi:///\" -d stats",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "ldap-database-volume",
						MountPath: "/usr/local/var/openldap-data",
					},
					{
						Name:      "ldap-config",
						MountPath: "/usr/local/etc/openldap/slapd.conf",
						SubPath:   "slapd.conf",
					},
				},
			}},
			Volumes: []corev1.Volume{
				{
					Name: "ldap-database-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "openldap-" + openldap.Name,
						},
					},
				},
				{
					Name: "ldap-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "openldap-" + openldap.Name,
							},
						},
					},
				},
			},
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
