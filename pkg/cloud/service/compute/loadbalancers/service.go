package loadbalancers

import (
	"github.com/vpsie/cluster-api-provider-vpsie/pkg/cloud/scope"
)

// Service holds a collection of interfaces.
type Service struct {
	scope *scope.ClusterScope
}

// NewService returns a new service given the Vpsie api client.
func NewService(scope *scope.ClusterScope) *Service {
	return &Service{
		scope: scope,
	}
}
