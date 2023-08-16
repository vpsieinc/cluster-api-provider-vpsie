/*
Copyright 2023.

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
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	infrav1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	"github.com/vpsie/cluster-api-provider-vpsie/pkg/cloud/scope"
	"github.com/vpsie/cluster-api-provider-vpsie/pkg/cloud/service/compute/vpsies"
	"github.com/vpsie/cluster-api-provider-vpsie/util/reconciler"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// VpsieMachineReconciler reconciles a VpsieMachine object
type VpsieMachineReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vpsiemachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vpsiemachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vpsiemachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

func (r *VpsieMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Reconcile request received")

	// Fetch the VpsieMachine instance
	vpsieMachine := &infrav1.VpsieMachine{}
	err := r.Client.Get(ctx, req.NamespacedName, vpsieMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine that owns the VpsieMachine
	machine, err := util.GetOwnerMachine(ctx, r.Client, vpsieMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "failed to get Owner Machine")
		return ctrl.Result{}, err
	}

	if machine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on VpsieMachine")
		return ctrl.Result{}, nil
	}

	// Fetch the Cluster
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, vpsieMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "VpsieMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info(fmt.Sprintf("Please associate this machine with a cluster using the label %s: <name of cluster>", clusterv1.ClusterNameLabel))
		return ctrl.Result{}, nil
	}
	logger = logger.WithValues("cluster", cluster.Name)

	vpsieCluster := &infrav1.VpsieCluster{}
	vpsieClusterName := client.ObjectKey{
		Namespace: vpsieMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err = r.Client.Get(ctx, vpsieClusterName, vpsieCluster); err != nil {
		logger.Error(err, "failed to get vpsie cluster")
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused
	if annotations.IsPaused(cluster, vpsieMachine) {
		logger.Info("vpsieMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       logger,
		Cluster:      cluster,
		VpsieCluster: vpsieCluster,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:       logger,
		Client:       r.Client,
		Cluster:      cluster,
		Machine:      machine,
		VpsieCluster: vpsieCluster,
		VpsieMachine: vpsieMachine,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(vpsieMachine, infrav1.MachineFinalizer) {
		controllerutil.AddFinalizer(vpsieMachine, infrav1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	// Always close the scope when exiting this function so we can persist any DOMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !vpsieMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, machineScope, clusterScope)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VpsieMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VpsieMachine{}).
		Complete(r)
}

func (r *VpsieMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling VpsieMachine")

	if err := vpsies.NewService(machineScope).Reconcile(ctx); err != nil {
		logger.Error(err, "Error reconciling vpsie(vm) resources")
		record.Warnf(machineScope.VpsieMachine, "GCPMachineReconcile", "Reconcile error - %v", err)
		return ctrl.Result{}, err
	}

	vmState := *machineScope.GetInstanceStatus()

	switch vmState {
	case "pending":
		logger.Info("VpsieMachine instance is pending", "instance-id", *machineScope.GetInstanceID())
		record.Eventf(machineScope.VpsieMachine, "VpsieMachineReconcile", "VpsieMachine instance is pending - instance-id: %s", *machineScope.GetInstanceID())
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	case infrav1.InstanceStatusActive:
		logger.Info("VpsieMachine instance is running", "instance-id", *machineScope.GetInstanceID())
		record.Eventf(machineScope.VpsieMachine, "VpsieMachineReconcile", "VpsieMachine instance is running - instance-id: %s", *machineScope.GetInstanceID())
		record.Event(machineScope.VpsieMachine, "VpsieMachineReconcile", "Reconciled")
		machineScope.SetReady()
		return ctrl.Result{}, nil
	default:
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("Instance status %q is unexpected", vmState))
		return ctrl.Result{Requeue: true}, nil
	}
}

func (r *VpsieMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
