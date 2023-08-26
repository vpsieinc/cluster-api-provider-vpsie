package scope

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/utils/pointer"

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

	params.VpsieClients.Services, err = params.VpsieClients.Session()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Vpsie session")
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

func (s *ClusterScope) Region() string {
	return s.VpsieCluster.Spec.DcIdentifier
}

func (s *ClusterScope) Name() string {
	return s.Cluster.GetName()
}

// UID returns the cluster UID.
func (s *ClusterScope) UID() string {
	return string(s.Cluster.UID)
}

func (s *ClusterScope) NameSpace() string {
	return s.Cluster.GetNamespace()
}

// Network returns the cluster network object.
func (s *ClusterScope) Network() *infrav1.VpsieNetworkResource {
	return &s.VpsieCluster.Status.Network
}

// SetReady sets cluster ready status.
func (s *ClusterScope) SetReady() {
	s.VpsieCluster.Status.Ready = true
}

// APIServerLoadbalancers get the VpsieCluster Spec Network APIServerLoadbalancers.
func (s *ClusterScope) APIServerLoadbalancers() *infrav1.LoadBalancer {
	return &s.VpsieCluster.Spec.Network.APIServerLoadbalancers
}

// APIServerLoadbalancersRef get the VpsieCluster status Network APIServerLoadbalancersRef.
func (s *ClusterScope) APIServerLoadbalancersRef() *infrav1.VpsieResourceReference {
	return &s.VpsieCluster.Status.Network.APIServerLoadbalancersRef
}

// ControlPlaneEndpoint returns the cluster control-plane endpoint.
func (s *ClusterScope) ControlPlaneEndpoint() clusterv1.APIEndpoint {
	endpoint := s.VpsieCluster.Spec.ControlPlaneEndpoint
	endpoint.Port = 443
	if c := s.Cluster.Spec.ClusterNetwork; c != nil {
		endpoint.Port = pointer.Int32Deref(c.APIServerPort, 443)
	}
	return endpoint
}

// SetControlPlaneEndpoint sets cluster control-plane endpoint.
func (s *ClusterScope) SetControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) {
	s.VpsieCluster.Spec.ControlPlaneEndpoint = endpoint
}
