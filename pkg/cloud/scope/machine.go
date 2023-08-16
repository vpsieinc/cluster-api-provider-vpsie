package scope

import (
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	infrav1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MachineScopeParams struct {
	Client client.Client
	Logger logr.Logger

	Machine      *clusterv1.Machine
	Cluster      *clusterv1.Cluster
	VpsieCluster *infrav1.VpsieCluster
	VpsieMachine *infrav1.VpsieMachine
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.VpsieCluster == nil {
		return nil, errors.New("DOCluster is required when creating a MachineScope")
	}
	if params.VpsieMachine == nil {
		return nil, errors.New("DOMachine is required when creating a MachineScope")
	}

	helper, err := patch.NewHelper(params.VpsieMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		client:       params.Client,
		Cluster:      params.Cluster,
		Machine:      params.Machine,
		VpsieCluster: params.VpsieCluster,
		VpsieMachine: params.VpsieMachine,
		Logger:       params.Logger,
		patchHelper:  helper,
	}, nil
}

type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Machine      *clusterv1.Machine
	Cluster      *clusterv1.Cluster
	VpsieCluster *infrav1.VpsieCluster
	VpsieMachine *infrav1.VpsieMachine
}

func (m *MachineScope) Close() error {
	return nil
}

// GetInstanceStatus returns the VpsieMachine instance status.
func (m *MachineScope) GetInstanceStatus() *infrav1.VpsieInstanceStatus {
	return m.VpsieMachine.Status.InstanceStatus
}

// SetInstanceStatus sets the VpsieMachine instance status.
func (m *MachineScope) SetInstanceStatus(v infrav1.VpsieInstanceStatus) {
	m.VpsieMachine.Status.InstanceStatus = &v
}
