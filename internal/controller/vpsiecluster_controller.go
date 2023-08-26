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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/predicates"

	"sigs.k8s.io/cluster-api/util/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	"github.com/vpsie/cluster-api-provider-vpsie/pkg/cloud/scope"
	"github.com/vpsie/cluster-api-provider-vpsie/pkg/cloud/service/compute/loadbalancers"
	"github.com/vpsie/cluster-api-provider-vpsie/util/reconciler"
)

// VpsieClusterReconciler reconciles a VpsieCluster object
type VpsieClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vpsieclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vpsieclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vpsieclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VpsieCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *VpsieClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	logger := log.FromContext(ctx)

	vpsieCluster := &infrav1.VpsieCluster{}
	if err := r.Get(ctx, req.NamespacedName, vpsieCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get the Cluster
	cluster, err := util.GetOwnerCluster(ctx, r.Client, vpsieCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		logger.Info("Waiting for Cluster Controller to set OwnerRef on VpsieCluster")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, logger)

	if annotations.IsPaused(cluster, vpsieCluster) {
		logger.Info("VpsieCluster or owning Cluster is marked as paused, not reconciling")

		return ctrl.Result{}, nil
	}

	// Create the Cluster Scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		VpsieCluster: vpsieCluster,
		Logger:       logger,
		Cluster:      cluster,
	})

	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any changes.
	defer func() {
		if err := clusterScope.Close(); err != nil && rerr == nil {
			rerr = err
		}
	}()

	// Handle deleted clusters
	if !vpsieCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	return r.reconcileNormal(ctx, clusterScope)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VpsieClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {

	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.VpsieCluster{}).
		//WithOptions(options).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Build(r)
	if err != nil {
		return err
	}

	return c.Watch(
		source.Kind(mgr.GetCache(), &clusterv1.Cluster{}),
		handler.EnqueueRequestsFromMapFunc(util.ClusterToInfrastructureMapFunc(ctx, infrav1.GroupVersion.WithKind("VpsieCluster"), mgr.GetClient(), &infrav1.VpsieCluster{})),
		predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx)),
	)
}

func (r *VpsieClusterReconciler) reconcileNormal(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling VpsieCluster")

	if !controllerutil.ContainsFinalizer(clusterScope.VpsieCluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(clusterScope.VpsieCluster, infrav1.ClusterFinalizer)

		return ctrl.Result{Requeue: true}, nil
	}

	loadbalancersvc := loadbalancers.NewService(clusterScope)
	err := loadbalancersvc.Reconcile(ctx)
	if err != nil {
		if err.Error() == "LoadBalancer Pending" {
			logger.Info("LOadBalancer is in Pending state")
			record.Event(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Waiting for load-balancer")
			return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		logger.Error(err, "Reconcile error")
		record.Warnf(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Reconcile error - %v", err)
		return ctrl.Result{}, err
	}

	controlPlaneEndpoint := clusterScope.ControlPlaneEndpoint()

	if controlPlaneEndpoint.Host == "" {
		logger.Info("VpsieCluster does not have control-plane endpoint yet. Reconciling")
		record.Event(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Waiting for control-plane endpoint")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	record.Eventf(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Got control-plane endpoint - %s", controlPlaneEndpoint.Host)
	clusterScope.SetReady()
	record.Event(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Reconciled")
	return ctrl.Result{}, nil
}

func (r *VpsieClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling VpsieCluster deletion")

	loadbalancersvc := loadbalancers.NewService(clusterScope)
	if err := loadbalancersvc.Delete(ctx); err != nil {
		logger.Error(err, "Reconcile error")
		record.Warnf(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Reconcile error - %v", err)
		return ctrl.Result{}, err
	}

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(clusterScope.VpsieCluster, infrav1.ClusterFinalizer)
	record.Event(clusterScope.VpsieCluster, "VpsieClusterReconcile", "Reconciled")
	return ctrl.Result{}, nil
}
