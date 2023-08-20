package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	infrav1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	capierrors "sigs.k8s.io/cluster-api/errors"
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

	VpsieClients

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

func (m *MachineScope) SetReady() {
	m.VpsieMachine.Status.Ready = true
}

func (m *MachineScope) GetInstanceID() *string {
	parsed, err := noderefutil.NewProviderID(m.GetProviderID())
	if err != nil {
		return nil
	}

	return pointer.String(parsed.ID())
}

func (m *MachineScope) GetProviderID() string {
	if m.VpsieMachine.Spec.ProviderID != nil {
		return *m.VpsieMachine.Spec.ProviderID
	}

	return ""
}

func (m *MachineScope) SetProviderID(v string) {
	m.VpsieMachine.Spec.ProviderID = pointer.String(v)
}

func (m *MachineScope) Name() string {
	return m.Machine.Name
}

func (m *MachineScope) NameSpace() string {
	return m.Machine.Namespace
}

func (m *MachineScope) SetFailureMessage(v error) {
	m.VpsieMachine.Status.FailureMessage = pointer.StringPtr(v.Error())
}

func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.VpsieMachine.Status.FailureReason = &v
}

func (m *MachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.VpsieMachine.Status.Addresses = addrs
}

func (m *MachineScope) GetBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("Bootstrap.DataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.NameSpace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		err = errors.Wrapf(err, "failed to retrieve bootstrap data secret for VpsieMachine %s/%s", m.NameSpace(), m.Name())
		return "", err
	}

	data, ok := secret.Data["value"]
	if !ok {
		return "", errors.Errorf("failed to retrieve bootstrap data from secret %s/%s", secret.Namespace, secret.Name)
	}

	return string(data), nil
}

