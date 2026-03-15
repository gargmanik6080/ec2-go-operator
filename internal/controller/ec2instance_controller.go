/*
Copyright 2026.

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

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	computev1 "github.com/gargmanik6080/ec2-go-operator/api/v1"
)

// EC2InstanceReconciler reconciles a EC2Instance object
type EC2InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.mycloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.mycloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.mycloud.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EC2Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *EC2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	// TODO(user): your logic here
	// r.Get(ctx, req.NamespacedName, ec2Instance)
	
	// l.Info("Reconciling EC2Instance", "Name", ec2Instance.Name)
	
	// fmt.Println("EC2 name is: ", ec2Instance.Name)
	
	// l.Info("EC2 reconciled", "Name", ec2Instance.Name)
	l.Info("=== RECONCILATION LOOP STARTED ===", "namespace", req.Namespace, "name", req.Name)

	ec2Instance := &computev1.EC2Instance{}
	if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
		if errors.IsNotFound(err) {
			l.Info("Instance Deleted, No need to reconcile")
			return ctrl.Result{}, nil
		}

		// If another error, backoff and retry
		return ctrl.Result{}, err
	}

	// checking if DeletionTimestamp is not zero
	if !ec2Instance.DeletionTimestamp.IsZero() {
		l.Info("Has deletionTimestamp, Instance is being deleted")
		_, err := deleteInstance(ctx, ec2Instance)
		if err != nil {
			l.Error(err, "Failed to delete EC2 instance")

			return ctrl.Result{Requeue: true}, err
		}

		// Remove the finaliser
		controllerutil.RemoveFinalizer(ec2Instance, "ec2instance.compute.mycloud.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			l.Error(err, "Failed to remove finalizer")

			return ctrl.Result{Requeue: true}, err
		}

		// Instance has been terminated and finalizer is removed
		return ctrl.Result{}, nil
	}

	// If InstanceID is in status
	if ec2Instance.Status.InstanceID != "" {
		l.Info("Requesting object already exists in the namespace. Not creating a new instance", "instanceID", ec2Instance.Status.InstanceID)

		instanceExists, instanceState, err := checkEC2InstanceExists(ctx, ec2Instance.Status.InstanceID, ec2Instance)
		if err != nil {
			// Instance might be terminated, clearing the status to trigger the reconcilation loop. to create a new instance
			ec2Instance.Status.InstanceID = ""
			ec2Instance.Status.State = ""
			ec2Instance.Status.PublicIP = ""
			ec2Instance.Status.PrivateIP = ""
			ec2Instance.Status.PublicDNS = ""
			ec2Instance.Status.PrivateDNS = ""

			err = r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{Requeue: true}, err
		}

		if !instanceExists {
			l.Info("Instance does not exist or is not in running state", 
				"instanceID", ec2Instance.Status.InstanceID,
				"state", instanceState)

			ec2Instance.Status.State = "Unknown"
			ec2Instance.Status.PublicIP = ""

			err = r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{}, err
		}

		l.Info("Instance already exists and is in Running state")
		if instanceExists && ec2Instance.Status.State == "Unknown" {
			// if the instance state was previously marked as unknown, updating it now. 
			ec2Instance.Status.State = string(instanceState.State.Name)
			ec2Instance.Status.PublicIP = *instanceState.PublicIpAddress

			err = r.Status().Update(ctx, ec2Instance)
			return ctrl.Result{}, err

		}
		return ctrl.Result{}, nil
	}

	// If instance is not being deleted and is not present, we are craeting a new instance
	l.Info("Creating new instance")

	l.Info("=== ADDING FINALIZER TO THE RESOURCE ===")
	ec2Instance.Finalizers = append(ec2Instance.Finalizers, "ec2instance.compute.mycloud.com")
	if err := r.Update(ctx,  ec2Instance); err != nil {
		l.Error(err, "Failed to add Finalizer")

		return ctrl.Result{Requeue: true}, err
	}
	l.Info("=== FINALIZER ADDED - This will trigger a new reconcilation loop, but the current loop continues ===")

	// Creating a new instance
	l.Info("=== PROCEEDING WITH INSTANCE CREATION ===")

	createdInstanceInfo, err := createEC2Instance(ec2Instance)
	if err != nil {
		l.Error(err, "Failed to create EC2 instance")
		return ctrl.Result{}, err
	}

	l.Info("=== INSTANCE CREATED ===")
	l.Info("=== ABOUT TO UPDATE THE STATUS - This will trigger the reconcilation loop again ===",
		"instanceID", createdInstanceInfo.InstanceID, 
		"state", createdInstanceInfo.State)

	ec2Instance.Status.InstanceID = createdInstanceInfo.InstanceID
	ec2Instance.Status.State = createdInstanceInfo.State
	ec2Instance.Status.PublicIP = createdInstanceInfo.PublicIP
	ec2Instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
	ec2Instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
	ec2Instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS

	err = r.Status().Update(ctx, ec2Instance)
	if err != nil {
		l.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	l.Info("=== STATUS UPDATED ===")


	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EC2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.EC2Instance{}).
		Named("ec2instance").
		Complete(r)
}
