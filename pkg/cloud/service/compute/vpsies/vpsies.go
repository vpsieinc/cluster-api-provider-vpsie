package vpsies

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/vpsie/govpsie"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (s *Service) CreateVpsie(ctx context.Context) (*govpsie.VmData, error) {
	logger := log.FromContext(ctx)
	logger.Info("Create an instance of Vpsie")

	bootstrapdata, err := s.scope.GetBootstrapData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bootstrap data")
	}

	instancename := s.scope.Name()

	scriptRequest := govpsie.CreateScriptRequest{
		Name:          instancename,
		ScriptContent: bootstrapdata,
		ScriptType:    "bash",
		Tags: []string{
			"bootstraped-scripts",
		},
	}

	logger.Info("Create a script", "script", scriptRequest)
	err = s.scope.VpsieClients.Services.Scripts.CreateScript(ctx, &scriptRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create script")
	}

	logger.Info("create-vm-check-point-1")

	// list scripts and search for the one we just created
	scripts, err := s.scope.VpsieClients.Services.Scripts.GetScripts(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list scripts")
	}

	logger.Info("create-vm-check-point-2")

	var script *govpsie.Script
	for _, v := range scripts {
		if v.ScriptName == fmt.Sprintf("%s.sh", instancename) {
			script = &v
			break
		}
	}

	logger.Info("create-vm-check-point-3")

	projectid, _ := strconv.Atoi(s.scope.VpsieCluster.Spec.Project)
	request := &govpsie.CreateVpsieRequest{
		ResourceIdentifier: *s.scope.VpsieMachine.Spec.VpsiePlan,
		Hostname:           instancename,
		OsIdentifier:       *s.scope.VpsieMachine.Spec.OsIdentifier,
		DcIdentifier:       s.scope.VpsieCluster.Spec.DcIdentifier,
		ProjectID:          projectid,
		AddPublicIpV4:      1,
		AddPublicIpV6:      1,
		AddPrivateIp:       1,
		BackupEnabled:      1,
		ScriptIdentifier:   script.Identifier,
		Notes:              "creating vm for cluster api",
		Tags: []string{
			"cluster-api-vm",
		},
		SshKeyIdentifier: "6f992c35-daa9-11ed-8b5c-0050569c68dc",
	}

	logger.Info("create-vm-check-point-4")
	err = s.scope.VpsieClients.Services.Vpsie.CreateVpsie(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Vpsie")
	}

	var vpsie *govpsie.VmData
	// list all vpsies and search for the one we just created
	vpsies, err := s.scope.VpsieClients.Services.Vpsie.ListVpsie(ctx, &govpsie.ListOptions{}, s.scope.VpsieCluster.Spec.Project)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Vpsies")
	}

	for _, v := range vpsies {
		if v.Hostname == instancename {
			vpsie = &v
			break
		}
	}

	if vpsie == nil {
		return nil, errors.New("Vpsie not found")
	}

	return vpsie, nil
}

func (s *Service) GetVpsieAddress(vpsie *govpsie.VmData) ([]corev1.NodeAddress, error) {
	addresses := []corev1.NodeAddress{}

	privatev4 := vpsie.PrivateIP

	addresses = append(addresses, corev1.NodeAddress{
		Type:    corev1.NodeInternalIP,
		Address: privatev4,
	})

	publicv4 := vpsie.PublicIp

	addresses = append(addresses, corev1.NodeAddress{
		Type:    corev1.NodeExternalIP,
		Address: *publicv4,
	})

	return addresses, nil
}

func (s *Service) GetVpsie(ctx context.Context, id *string) (*govpsie.VmData, error) {
	if id == nil {
		s.scope.Info("VpsieMachine does not have an instance id")
		return nil, nil
	}

	logger := log.FromContext(ctx)
	logger.Info("Get an instance of Vpsie")

	vpsie, err := s.scope.VpsieClients.Services.Vpsie.GetVpsieByIdentifier(ctx, *id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Vpsie")
	}
	return vpsie, nil
}

func (s *Service) Delete(ctx context.Context, id *string) error {
	logger := log.FromContext(ctx)
	logger.Info("Delete an instance of Vpsie")

	if id == nil {
		s.scope.Info("VpsieMachine does not have an instance id")
		return nil
	}

	err := s.scope.VpsieClients.Services.Vpsie.DeleteVpsie(ctx, *id)
	if err != nil {
		return errors.Wrap(err, "failed to delete Vpsie")
	}

	logger.Info("Vpsie deleted successfully")

	return nil
}

func (s *Service) IsVmPending(ctx context.Context, hostname string) (*govpsie.PendingVm, error) {
	logger := log.FromContext(ctx)
	logger.Info("Check if Vpsie is pending")

	if hostname == "" {
		s.scope.Info("VpsieMachine does not have an instance hostname")
	}

	logger.Info("loggic before sending request", "vpsie-clients", s.scope.VpsieClients)
	logger.Info("loggic before sending request", "service", s.scope.VpsieClients.Services)
	logger.Info("loggic before sending request", "pending", s.scope.VpsieClients.Services.Pending)

	pendingVms, err := s.scope.VpsieClients.Services.Pending.GetPendingVms(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Vpsie")
	}

	logger.Info("All PendingVms", "PendingVms", pendingVms)

	for _, v := range pendingVms {
		logger.Info("PendingVm", "PendingVm", v)
		if v.Data.Hostname == hostname {
			return &v, nil
		}
	}

	logger.Info("NO Pending found")
	return nil, nil
}

func (s *Service) GetVpsieByName(ctx context.Context, hostName string) (*govpsie.VmData, error) {
	// list and find it by name
	logger := log.FromContext(ctx)
	logger.Info("Loging for vm by name", "Name", hostName)

	vpsies, err := s.scope.VpsieClients.Services.Vpsie.ListVpsie(ctx, &govpsie.ListOptions{}, s.scope.VpsieCluster.Spec.Project)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list Vpsies")
	}

	for _, v := range vpsies {
		logger.Info("Vpsie", "Vpsie", v)
		if v.Hostname == hostName {
			return &v, nil
		}
	}

	logger.Info("NO Vpsie found in Looking by name")
	return nil, nil
}
