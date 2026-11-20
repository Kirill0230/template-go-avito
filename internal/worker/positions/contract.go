package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type staleTripRepository interface {
	ListStale(
		ctx context.Context,
		staleAfter time.Duration,
		cursor *model.StaleTripCursor,
		limit int,
	) ([]model.StaleTrip, error)
}

type pushGateway interface {
	RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
}

type workerMetrics interface {
	StaleCheck(ctx context.Context, result string)
	QueueChanged(ctx context.Context, delta int64)
	ActiveWorkersChanged(ctx context.Context, delta int64)
	PositionRequest(ctx context.Context, result string, duration time.Duration)
	RowsRead(ctx context.Context, rows int64)
}

type ticker interface {
	C() <-chan time.Time
	Stop()
}

type tickerFactory interface {
	NewTicker(duration time.Duration) ticker
}
