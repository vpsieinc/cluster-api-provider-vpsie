package loadbalancers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
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

	loadbalancer, err := s.GetLB(ctx, lbUUID)
	if err != nil {
		return err
	}

	if loadbalancer == nil {
		loadbalancer, err = s.CreateLB(ctx, apiServerLoadbalancer)
		if err != nil {
			return errors.Wrapf(err, "failed to create load balancers for VpsieCluster %s/%s", s.scope.VpsieCluster.Namespace, s.scope.VpsieCluster.Name)
		}

		record.Eventf(s.scope.VpsieCluster, corev1.EventTypeNormal, "LoadBalancerCreated", "Created new load balancers - %s", loadbalancer.LBName)
	}

	apiServerLoadbalancerRef.ID = loadbalancer.Identifier
	// apiServerLoadbalancerRef.Status = infrav1.VpsieClusterStatus(loadbalancer.Status)
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
		return err
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
	request := govpsie.CreateLBReq{

		Rule:      rules,
		Algorithm: lbSpec.Algorithm,
		// HealthCheckPath:    lbSpec.HealthCheck.HealthyPath,
		LBName:             lbSpec.LbName,
		DcIdentifier:       lbSpec.DcIdentifier,
		ResourceIdentifier: lbSpec.ResourceIdentifier,
		RedirectHTTP:       lbSpec.RedirectHTTP,
		// CheckInterval:      lbSpec.HealthCheck.Interval,
		// FastInterval:       lbSpec.HealthCheck.Timeout,
		// Rise:               lbSpec.HealthCheck.HealthyThreshold,
		// Fall:               lbSpec.HealthCheck.UnhealthyThreshold,
	}

	err := s.scope.VpsieClients.Services.LB.CreateLB(ctx, &request)
	if err != nil {
		return nil, err
	}

	allLBs, err := s.scope.Services.LB.ListLBs(ctx, &govpsie.ListOptions{})
	if err != nil {
		return nil, err
	}

	var CreatedLB *govpsie.LB
	for _, lb := range allLBs {
		if lb.LBName == lbSpec.LbName {
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
