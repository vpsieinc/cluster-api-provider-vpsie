package loadbalancers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	infrav1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	"github.com/vpsie/govpsie"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/record"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconcile reconcile cluster network components.
func (s *Service) Reconcile(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling loadbalancer resources")

	apiServerLoadbalancer := s.scope.APIServerLoadbalancers()
	apiServerLoadbalancer.ApplyDefaults()

	apiServerLoadbalancerRef := s.scope.APIServerLoadbalancersRef()
	lbUUID := apiServerLoadbalancerRef.ID

	if apiServerLoadbalancer.ID != "" {
		lbUUID = apiServerLoadbalancer.ID
	}

	loadbalancer, _ := s.GetLB(ctx, lbUUID)
	// if err != nil {
	// 	return err
	// }

	if loadbalancer == nil {

		if apiServerLoadbalancerRef.Name != "" {
			pendingLB, err := s.IsLBPending(ctx, apiServerLoadbalancerRef.Name)
			if err != nil {
				return errors.Wrap(err, "failed to get loadbalancer")
			}

			if pendingLB != nil {
				logger.Info("LoadBalancer Pending", "pendingLB", pendingLB)
				return errors.New("LoadBalancer Pending")
			}

			loadbalancer, err = s.GetLBByName(ctx, apiServerLoadbalancerRef.Name)
			if err != nil {
				return errors.Wrap(err, "failed to get loadbalancer")
			}
		} else {
			loadbalancer, err := s.CreateLB(ctx, apiServerLoadbalancer)
			if err != nil {
				return errors.Wrapf(err, "failed to create load balancers for VpsieCluster %s/%s", s.scope.VpsieCluster.Namespace, s.scope.VpsieCluster.Name)
			}

			if loadbalancer == nil {
				pendingLB , err := s.IsLBPending(ctx, "name")
				// update name, status
				if err != nil {
					return errors.Wrap(err, "failed to get loadbalancer")
				}
				apiServerLoadbalancerRef.Name = pendingLB.Data.LbName
				apiServerLoadbalancerRef.Status = "Pending"
				return errors.New("LoadBalancer Pending")
			}


			record.Eventf(s.scope.VpsieCluster, corev1.EventTypeNormal, "LoadBalancerCreated", "Created new load balancers - %s", loadbalancer.LBName)
		}
	}

	apiServerLoadbalancerRef.ID = loadbalancer.Identifier
	apiServerLoadbalancerRef.Status = "Running"
	apiServerLoadbalancer.ID = loadbalancer.Identifier

	record.Eventf(s.scope.VpsieCluster, corev1.EventTypeNormal, "LoadBalancerReady", "LoadBalancer got an IP Address - %s", loadbalancer.DefaultIP)

	endpoint := s.scope.ControlPlaneEndpoint()
	endpoint.Host = loadbalancer.DefaultIP
	s.scope.SetControlPlaneEndpoint(endpoint)

	// infrav1.LoadBalancer.Rules
	return nil

}

// Delete delete cluster control-plane loadbalancer compoenents.
func (s *Service) Delete(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Deleting loadbalancer resources")

	apiServerLoadbalancer := s.scope.APIServerLoadbalancers()

	loadbalancer, err := s.GetLB(ctx, apiServerLoadbalancer.ID)
	if err != nil {
		return nil
	}

	if loadbalancer == nil {
		logger.Info("Unable to locate load balancer")
		record.Eventf(s.scope.VpsieCluster, corev1.EventTypeWarning, "NoLoadBalancerFound", "Unable to find matching load balancer")
		return nil
	}

	if err := s.DeleteLoadBalancer(ctx, loadbalancer.Identifier); err != nil {
		return errors.Wrapf(err, "error deleting load balancer for DOCluster %s/%s", s.scope.VpsieCluster.Namespace, s.scope.VpsieCluster.Name)
	}

	return nil
}

func (s *Service) CreateLB(ctx context.Context, lbSpec *v1alpha1.LoadBalancer) (*govpsie.LB, error) {
	logger := log.FromContext(ctx)
	logger.Info("Creating loadbalancer resources")

	logger.Info("before creation")
	logger.Info("before creation", "rules", lbSpec.Rules)

	clusterName := infrav1.SafeName(s.scope.Name())
	name := clusterName + "-" + s.scope.UID()

	rules := make([]govpsie.Rule, len(lbSpec.Rules))
	for i, obj := range lbSpec.Rules {

		backends := make([]govpsie.Backend, len(obj.Backends))

		domains := make([]govpsie.LBDomain, len(obj.Domains))

		rules[i] = govpsie.Rule{
			Scheme:     obj.Scheme,
			FrontPort:  obj.FrontPort,
			BackPort:   obj.BackPort,
			Backends:   backends,
			DomainName: obj.DomainName,
			Domains:    domains,
		}
	}

	logger.Info("rules creation", "rules", rules)

	request := govpsie.CreateLBReq{

		Rule:      rules,
		Algorithm: lbSpec.Algorithm,
		// HealthCheckPath:    lbSpec.HealthCheck.HealthyPath,
		LBName:             name,
		DcIdentifier:       s.scope.VpsieCluster.Spec.DcIdentifier,
		ResourceIdentifier: lbSpec.ResourceIdentifier,
		// CheckInterval:      lbSpec.HealthCheck.Interval,
		// FastInterval:       lbSpec.HealthCheck.Timeout,
		// Rise:               lbSpec.HealthCheck.HealthyThreshold,
		// Fall:               lbSpec.HealthCheck.UnhealthyThreshold,
	}

	logger.Info("request payload", "request", request)

	err := s.scope.VpsieClients.Services.LB.CreateLB(ctx, &request)
	if err != nil {
		logger.Info("during creation", "err", err)
		return nil, err
	}

	logger.Info("after creation")
	allLBs, err := s.scope.Services.LB.ListLBs(ctx, &govpsie.ListOptions{})
	if err != nil {
		return nil, err
	}

	var CreatedLB *govpsie.LB
	for _, lb := range allLBs {
		logger.Info("during list all", "lb", lb)

		if lb.LBName == name {
			CreatedLB = &lb
			break
		}
	}

	return CreatedLB, nil
}

// DeleteLoadBalancer delete a LB by ID.
func (s *Service) DeleteLoadBalancer(ctx context.Context, id string) error {
	if err := s.scope.VpsieClients.Services.LB.DeleteLB(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetLB(ctx context.Context, id string) (*govpsie.LB, error) {
	return s.scope.Services.LB.GetLB(ctx, id)
}

func (s *Service) GetLBByName(ctx context.Context, name string) (*govpsie.LB, error) {
	// list first and find by name
	lbs, err := s.scope.Services.LB.ListLBs(ctx, &govpsie.ListOptions{})
	if err != nil {
		return nil, err
	}

	var lb *govpsie.LB
	for _, v := range lbs {
		if v.LBName == name {
			lb = &v
			break
		}
	}

	return lb, nil
}

func (s *Service) IsLBPending(ctx context.Context, lbname string) (*govpsie.PendingLB, error) {
	logger := log.FromContext(ctx)
	logger.Info("Check if LB is pending")

	if lbname == "" {
		s.scope.Info("VpsieMachine does not have an lb lbname")
	}

	pendingVms, err := s.scope.VpsieClients.Services.LB.ListPendingLBs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get loadbalancer")
	}

	for _, v := range pendingVms {
		if v.Data.LbName == lbname {
			return &v, nil
		}
	}

	return nil, nil
}
