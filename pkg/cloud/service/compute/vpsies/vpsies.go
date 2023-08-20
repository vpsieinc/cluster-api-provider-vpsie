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
	}

	err = s.scope.VpsieClients.Services.Scripts.CreateScript(ctx, &scriptRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create script")
	}

	// list scripts and search for the one we just created
	scripts, err := s.scope.VpsieClients.Services.Scripts.GetScripts(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list scripts")
	}

	var script *govpsie.Script
	for _, v := range scripts {
		if v.ScriptName == fmt.Sprintf("%s.sh", instancename) {
			script = &v
			break
		}
	}

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
	}

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
		// check if pending
		pending, err := s.IsVmPending(ctx, instancename)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if Vpsie is pending")
		}

		if pending {
			return nil, errors.New("Vpsie is pending")
		}

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
	}

	err := s.scope.VpsieClients.Services.Vpsie.DeleteVpsie(ctx, *id)
	if err != nil {
		return errors.Wrap(err, "failed to delete Vpsie")
	}

	logger.Info("Vpsie deleted successfully")

	return nil
}

func (s *Service) IsVmPending(ctx context.Context, hostname string) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("Check if Vpsie is pending")

	if hostname == "" {
		s.scope.Info("VpsieMachine does not have an instance hostname")
	}

	pendingVms, err := s.scope.VpsieClients.Services.Pending.GetPendingVms(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get Vpsie")
	}

	for _, v := range pendingVms {
		if v.Data.Hostname == hostname {
			return true, nil
		}
	}

	return false, nil
}
