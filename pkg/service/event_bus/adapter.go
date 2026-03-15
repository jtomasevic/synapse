package event_bus

import (
	"context"
)

// IngesterFunc is an adapter that lets a plain function satisfy the
// Ingester interface. This is used by the service layer to bridge
// SynapseService.IngestEvent without importing it directly (which
// would create an import cycle).
type IngesterFunc func(ctx context.Context, synapseID string, event IngestDomainEvent) (string, error)

func (f IngesterFunc) IngestEvent(ctx context.Context, synapseID string, event IngestDomainEvent) (string, error) {
	return f(ctx, synapseID, event)
}
