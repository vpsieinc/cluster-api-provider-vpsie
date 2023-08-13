package domains

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconcile reconcile cluster domain components.
func (s *Service) Reconcile(ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Reconciling domain resources")

	return nil
}
