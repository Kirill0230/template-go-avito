package positions

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/goleak"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

type manualTicker struct{ ticks chan time.Time }

func (t *manualTicker) C() <-chan time.Time { return t.ticks }
func (t *manualTicker) Stop()               {}

type manualTickerFactory struct{ ticker *manualTicker }

func (f manualTickerFactory) NewTicker(time.Duration) ticker { return f.ticker }

type staticRepository struct{ trips []model.StaleTrip }

func (r staticRepository) ListStale(
	context.Context,
	time.Duration,
	*model.StaleTripCursor,
	int,
) ([]model.StaleTrip, error) {
	return r.trips, nil
}

type metricsSpy struct {
	stale chan string
	rows  chan int64
}

func (m *metricsSpy) StaleCheck(_ context.Context, result string) {
	if m.stale != nil {
		m.stale <- result
	}
}
func (*metricsSpy) QueueChanged(context.Context, int64)                    {}
func (*metricsSpy) ActiveWorkersChanged(context.Context, int64)            {}
func (*metricsSpy) PositionRequest(context.Context, string, time.Duration) {}
func (m *metricsSpy) RowsRead(_ context.Context, rows int64) {
	if m.rows != nil {
		m.rows <- rows
	}
}

type pushFunc func(context.Context, uuid.UUID, uuid.UUID) error

func (f pushFunc) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	return f(ctx, driverID, tripID)
}

func testWorker(
	repository staleTripRepository,
	push pushGateway,
	metrics workerMetrics,
	workers, queueSize int,
	ticker *manualTicker,
) *Worker {
	return newWithTickerFactory(
		repository, push, metrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Hour, time.Minute, time.Hour,
		workers, queueSize, 100, manualTickerFactory{ticker: ticker},
	)
}

func stopWorker(t *testing.T, worker *Worker) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Stop(ctx); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestDuplicateTripIsNotQueuedTwice(t *testing.T) {
	trip := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	ticker := &manualTicker{ticks: make(chan time.Time)}
	metrics := &metricsSpy{stale: make(chan string, 2)}
	started := make(chan struct{})
	release := make(chan struct{})
	worker := testWorker(staticRepository{[]model.StaleTrip{trip}}, pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return nil
	}), metrics, 1, 1, ticker)
	worker.Start(context.Background())

	ticker.ticks <- time.Now()
	if result := <-metrics.stale; result != resultQueued {
		t.Fatalf("first result = %q", result)
	}
	<-started
	ticker.ticks <- time.Now()
	if result := <-metrics.stale; result != resultDuplicate {
		t.Fatalf("second result = %q", result)
	}

	close(release)
	stopWorker(t, worker)
}

func TestCanceledQueuedTripIsSkipped(t *testing.T) {
	first := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	second := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	started := make(chan uuid.UUID, 2)
	release := make(chan struct{})
	worker := testWorker(staticRepository{}, pushFunc(func(_ context.Context, _, tripID uuid.UUID) error {
		started <- tripID
		if tripID == first.ID {
			<-release
		}
		return nil
	}), &metricsSpy{stale: make(chan string, 2)}, 1, 2, &manualTicker{ticks: make(chan time.Time)})
	worker.Start(context.Background())

	worker.enqueue(context.Background(), first, trace.SpanContext{})
	if got := <-started; got != first.ID {
		t.Fatalf("started trip = %s", got)
	}
	worker.enqueue(context.Background(), second, trace.SpanContext{})
	worker.Cancel(second.ID)
	close(release)
	stopWorker(t, worker)

	select {
	case got := <-started:
		t.Fatalf("unexpected push for canceled trip %s", got)
	default:
	}
}

func TestFullQueueDropsWithoutBlocking(t *testing.T) {
	worker := testWorker(staticRepository{}, pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error { return nil }), &metricsSpy{stale: make(chan string, 2)}, 0, 1, &manualTicker{ticks: make(chan time.Time)})
	old := time.Now().Add(-time.Hour)
	if got, _ := worker.enqueue(context.Background(), model.StaleTrip{ID: uuid.New(), SilentSince: old}, trace.SpanContext{}); got != resultQueued {
		t.Fatalf("first enqueue = %q", got)
	}
	if got, _ := worker.enqueue(context.Background(), model.StaleTrip{ID: uuid.New(), SilentSince: old.Add(time.Minute)}, trace.SpanContext{}); got != resultDropped {
		t.Fatalf("second enqueue = %q, want dropped", got)
	}
}

func TestWorkerPoolProcessesConcurrently(t *testing.T) {
	const workerCount = 3
	started := make(chan struct{}, workerCount)
	release := make(chan struct{})
	worker := testWorker(staticRepository{}, pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error {
		started <- struct{}{}
		<-release
		return nil
	}), &metricsSpy{stale: make(chan string, workerCount)}, workerCount, workerCount, &manualTicker{ticks: make(chan time.Time)})
	worker.Start(context.Background())
	for range workerCount {
		worker.enqueue(context.Background(), model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}, trace.SpanContext{})
	}
	for range workerCount {
		<-started
	}
	close(release)
	stopWorker(t, worker)
}

func TestPushErrorDoesNotStopWorker(t *testing.T) {
	called := make(chan int, 2)
	var mu sync.Mutex
	calls := 0
	worker := testWorker(staticRepository{}, pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error {
		mu.Lock()
		calls++
		current := calls
		mu.Unlock()
		called <- current
		if current == 1 {
			return errors.New("push failed")
		}
		return nil
	}), &metricsSpy{stale: make(chan string, 2)}, 1, 2, &manualTicker{ticks: make(chan time.Time)})
	worker.Start(context.Background())
	worker.enqueue(context.Background(), model.StaleTrip{ID: uuid.New()}, trace.SpanContext{})
	worker.enqueue(context.Background(), model.StaleTrip{ID: uuid.New()}, trace.SpanContext{})
	if got := <-called; got != 1 {
		t.Fatalf("first call = %d", got)
	}
	if got := <-called; got != 2 {
		t.Fatalf("second call = %d", got)
	}
	stopWorker(t, worker)
}

type pagedRepository struct {
	trips []model.StaleTrip
	calls int
}

func (r *pagedRepository) ListStale(
	_ context.Context,
	_ time.Duration,
	cursor *model.StaleTripCursor,
	limit int,
) ([]model.StaleTrip, error) {
	r.calls++
	start := 0
	if cursor != nil {
		for start < len(r.trips) {
			trip := r.trips[start]
			if trip.SilentSince.After(cursor.SilentSince) ||
				(trip.SilentSince.Equal(cursor.SilentSince) && bytes.Compare(trip.ID[:], cursor.ID[:]) > 0) {
				break
			}
			start++
		}
	}
	end := min(start+limit, len(r.trips))
	return r.trips[start:end], nil
}

func staleTrips(count int) []model.StaleTrip {
	trips := make([]model.StaleTrip, 0, count)
	silentSince := time.Date(2026, time.November, 20, 10, 0, 0, 0, time.UTC)
	for i := range count {
		id := uuid.Nil
		id[14] = byte(i >> 8)
		id[15] = byte(i)
		trips = append(trips, model.StaleTrip{
			ID: id, DriverID: uuid.New(), SilentSince: silentSince,
		})
	}
	return trips
}

func TestScannerReadsAllKeysetBatches(t *testing.T) {
	repository := &pagedRepository{trips: staleTrips(250)}
	metrics := &metricsSpy{stale: make(chan string, 250), rows: make(chan int64, 1)}
	worker := newWithTickerFactory(
		repository,
		pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error { return nil }),
		metrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Hour, time.Minute, time.Second,
		0, 300, 100, manualTickerFactory{ticker: &manualTicker{ticks: make(chan time.Time)}},
	)

	worker.scan(context.Background())
	if repository.calls != 3 {
		t.Fatalf("repository calls = %d, want 3", repository.calls)
	}
	if len(worker.queue) != 250 {
		t.Fatalf("queued trips = %d, want 250", len(worker.queue))
	}
	if rows := <-metrics.rows; rows != 250 {
		t.Fatalf("rows read = %d, want 250", rows)
	}
}

func TestScannerStopsAfterFirstDroppedBatch(t *testing.T) {
	repository := &pagedRepository{trips: staleTrips(250)}
	metrics := &metricsSpy{stale: make(chan string, 101), rows: make(chan int64, 1)}
	worker := newWithTickerFactory(
		repository,
		pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error { return nil }),
		metrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Hour, time.Minute, time.Second,
		0, 50, 100, manualTickerFactory{ticker: &manualTicker{ticks: make(chan time.Time)}},
	)

	worker.scan(context.Background())
	if repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.calls)
	}
	if len(worker.queue) != 50 {
		t.Fatalf("queued trips = %d, want 50", len(worker.queue))
	}
	if rows := <-metrics.rows; rows != 100 {
		t.Fatalf("rows read = %d, want 100", rows)
	}
}

func TestPriorityQueueProcessesOldestFirst(t *testing.T) {
	now := time.Now()
	old := model.StaleTrip{ID: uuid.New(), SilentSince: now.Add(-3 * time.Hour)}
	middle := model.StaleTrip{ID: uuid.New(), SilentSince: now.Add(-2 * time.Hour)}
	fresh := model.StaleTrip{ID: uuid.New(), SilentSince: now.Add(-time.Hour)}
	processed := make(chan uuid.UUID, 3)
	worker := testWorker(staticRepository{}, pushFunc(func(_ context.Context, _, tripID uuid.UUID) error {
		processed <- tripID
		return nil
	}), &metricsSpy{}, 1, 3, &manualTicker{ticks: make(chan time.Time)})
	worker.enqueue(context.Background(), middle, trace.SpanContext{})
	worker.enqueue(context.Background(), fresh, trace.SpanContext{})
	worker.enqueue(context.Background(), old, trace.SpanContext{})
	worker.Start(context.Background())

	for index, want := range []uuid.UUID{old.ID, middle.ID, fresh.ID} {
		if got := <-processed; got != want {
			t.Fatalf("processed[%d] = %s, want %s", index, got, want)
		}
	}
	stopWorker(t, worker)
}

func TestPriorityQueueEvictsFreshest(t *testing.T) {
	now := time.Now()
	old := model.StaleTrip{ID: uuid.New(), SilentSince: now.Add(-3 * time.Hour)}
	middle := model.StaleTrip{ID: uuid.New(), SilentSince: now.Add(-2 * time.Hour)}
	fresh := model.StaleTrip{ID: uuid.New(), SilentSince: now.Add(-time.Hour)}
	newest := model.StaleTrip{ID: uuid.New(), SilentSince: now}
	worker := testWorker(staticRepository{}, pushFunc(func(context.Context, uuid.UUID, uuid.UUID) error { return nil }), &metricsSpy{}, 0, 2, &manualTicker{ticks: make(chan time.Time)})

	worker.enqueue(context.Background(), middle, trace.SpanContext{})
	worker.enqueue(context.Background(), fresh, trace.SpanContext{})
	result, displaced := worker.enqueue(context.Background(), old, trace.SpanContext{})
	if result != resultQueued || displaced == nil || displaced.ID != fresh.ID {
		t.Fatalf("old enqueue = (%q, %#v), want queued and fresh displaced", result, displaced)
	}
	if _, exists := worker.pending[fresh.ID]; exists {
		t.Fatal("fresh trip is still pending after displacement")
	}
	if result, displaced = worker.enqueue(context.Background(), newest, trace.SpanContext{}); result != resultDropped || displaced != nil {
		t.Fatalf("newest enqueue = (%q, %#v), want dropped", result, displaced)
	}
}
