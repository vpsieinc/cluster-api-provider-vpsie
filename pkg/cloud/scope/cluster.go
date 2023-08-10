package scope

import (
	"context"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	VpsieClients
	Client       client.Client
	Logger       logr.Logger
	Cluster      *clusterv1.Cluster
	VpsieCluster *infrav1.VpsieCluster
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	VpsieClients
	Cluster      *clusterv1.Cluster
	VpsieCluster *infrav1.VpsieCluster
}

func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {

	if params.Cluster == nil {
		return nil, errors.New("Cluster is required")
	}

	if params.VpsieCluster == nil {
		return nil, errors.New("VpsieCluster is required")
	}

	helper, err := patch.NewHelper(params.VpsieCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &ClusterScope{
		Logger:       params.Logger,
		client:       params.Client,
		VpsieClients: params.VpsieClients,
		Cluster:      params.Cluster,
		VpsieCluster: params.VpsieCluster,
		patchHelper:  helper,
	}, nil
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.PatchObject()
}

// PatchObject persists the cluster configuration and status
func (s *ClusterScope) PatchObject() error {
	return s.patchHelper.Patch(context.TODO(), s.VpsieCluster)
}
