// Package networking have all services and interface to work with the Vpsie network API.
package networking

import (
	"context"

	"github.com/vpsie/cluster-api-provider-vpsie/pkg/cloud/scope"
)

// Service holds a collection of interfaces.
type Service struct {
	scope *scope.ClusterScope
	ctx   context.Context
}

// NewService returns a new service given the Vpsie api client.
func NewService(ctx context.Context, scope *scope.ClusterScope) *Service {
	return &Service{
		scope: scope,
		ctx:   ctx,
	}
}
