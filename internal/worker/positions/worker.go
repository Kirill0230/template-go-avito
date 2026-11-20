package positions

import (
	"container/heap"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

const (
	resultDuplicate = "duplicate"
	resultQueued    = "queued"
	resultDropped   = "dropped"
	resultSuccess   = "success"
	resultError     = "error"
)

type task struct {
	trip     model.StaleTrip
	token    uint64
	scanSpan trace.SpanContext
	index    int
}

type pendingTask struct {
	token uint64
	item  *task
}

type Worker struct {
	repository     staleTripRepository
	push           pushGateway
	metrics        workerMetrics
	logger         *slog.Logger
	checkInterval  time.Duration
	staleAfter     time.Duration
	requestTimeout time.Duration
	workerCount    int
	batchSize      int
	tickers        tickerFactory
	queueCapacity  int
	queue          priorityQueue
	queueClosed    bool

	mu      sync.Mutex
	cond    *sync.Cond
	pending map[uuid.UUID]pendingTask
	next    uint64

	startOnce sync.Once
	stopOnce  sync.Once
	scanStop  context.CancelFunc
	workStop  context.CancelFunc
	scanDone  chan struct{}
	workDone  chan struct{}
	workers   sync.WaitGroup
}

func New(
	repository staleTripRepository,
	push pushGateway,
	metrics workerMetrics,
	logger *slog.Logger,
	checkInterval time.Duration,
	staleAfter time.Duration,
	requestTimeout time.Duration,
	workerCount int,
	queueSize int,
	batchSize int,
) *Worker {
	return newWithTickerFactory(
		repository, push, metrics, logger,
		checkInterval, staleAfter, requestTimeout,
		workerCount, queueSize, batchSize, realTickerFactory{},
	)
}

func newWithTickerFactory(
	repository staleTripRepository,
	push pushGateway,
	metrics workerMetrics,
	logger *slog.Logger,
	checkInterval time.Duration,
	staleAfter time.Duration,
	requestTimeout time.Duration,
	workerCount int,
	queueSize int,
	batchSize int,
	tickers tickerFactory,
) *Worker {
	worker := &Worker{
		repository: repository, push: push, metrics: metrics, logger: logger,
		checkInterval: checkInterval, staleAfter: staleAfter,
		requestTimeout: requestTimeout, workerCount: workerCount,
		batchSize: batchSize, tickers: tickers, queueCapacity: queueSize,
		pending:  make(map[uuid.UUID]pendingTask),
		scanDone: make(chan struct{}), workDone: make(chan struct{}),
	}
	worker.cond = sync.NewCond(&worker.mu)
	return worker
}

func (w *Worker) Start(ctx context.Context) {
	w.startOnce.Do(func() {
		baseContext := context.WithoutCancel(ctx)
		scanContext, scanStop := context.WithCancel(baseContext)
		workContext, workStop := context.WithCancel(baseContext)
		w.scanStop = scanStop
		w.workStop = workStop

		w.workers.Add(w.workerCount)
		for range w.workerCount {
			go w.runWorker(workContext)
		}
		go func() {
			w.workers.Wait()
			close(w.workDone)
		}()
		go w.runScanner(scanContext)
	})
}

func (w *Worker) Stop(ctx context.Context) error {
	w.stopOnce.Do(func() {
		w.scanStop()
		<-w.scanDone
		w.mu.Lock()
		w.queueClosed = true
		w.cond.Broadcast()
		w.mu.Unlock()
	})

	select {
	case <-w.workDone:
		w.workStop()
		return nil
	case <-ctx.Done():
		w.workStop()
		<-w.workDone
		return ctx.Err()
	}
}

func (w *Worker) Cancel(tripID uuid.UUID) {
	w.mu.Lock()
	pending, exists := w.pending[tripID]
	removed := exists && pending.item != nil
	if removed {
		w.queue.remove(pending.item.index)
	}
	if exists {
		delete(w.pending, tripID)
	}
	w.mu.Unlock()
	if removed {
		w.metrics.QueueChanged(context.Background(), -1)
	}
}

func (w *Worker) runScanner(ctx context.Context) {
	defer close(w.scanDone)
	ticker := w.tickers.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			w.scan(ctx)
		}
	}
}

func (w *Worker) scan(ctx context.Context) {
	ctx, span := otel.Tracer("tripgo/worker/positions").Start(ctx, "scan stale trips")
	defer span.End()

	var rowsRead int64
	defer func() { w.metrics.RowsRead(ctx, rowsRead) }()

	var cursor *model.StaleTripCursor
	for {
		trips, err := w.repository.ListStale(ctx, w.staleAfter, cursor, w.batchSize)
		if err != nil {
			w.logger.ErrorContext(ctx, "scan stale trips", slog.Any("error", err))
			return
		}
		rowsRead += int64(len(trips))

		for _, trip := range trips {
			result, displaced := w.enqueue(ctx, trip, span.SpanContext())
			w.metrics.StaleCheck(ctx, result)
			if displaced != nil {
				w.metrics.StaleCheck(ctx, resultDropped)
				w.logDropped(ctx, displaced.ID)
				return
			}
			if result == resultDropped {
				w.logDropped(ctx, trip.ID)
				return
			}
		}

		if len(trips) < w.batchSize {
			return
		}
		last := trips[len(trips)-1]
		cursor = &model.StaleTripCursor{SilentSince: last.SilentSince, ID: last.ID}
	}
}

func (w *Worker) logDropped(ctx context.Context, tripID uuid.UUID) {
	w.logger.WarnContext(ctx, "position request queue is full", slog.String("trip_id", tripID.String()))
}

func (w *Worker) enqueue(
	ctx context.Context,
	staleTrip model.StaleTrip,
	scanSpan trace.SpanContext,
) (string, *model.StaleTrip) {
	w.mu.Lock()

	if _, exists := w.pending[staleTrip.ID]; exists {
		w.mu.Unlock()
		return resultDuplicate, nil
	}

	w.next++
	queuedTask := &task{trip: staleTrip, token: w.next, scanSpan: scanSpan, index: -1}
	if len(w.queue) < w.queueCapacity {
		heap.Push(&w.queue, queuedTask)
		w.pending[staleTrip.ID] = pendingTask{token: queuedTask.token, item: queuedTask}
		w.cond.Signal()
		w.mu.Unlock()
		w.metrics.QueueChanged(ctx, 1)
		return resultQueued, nil
	}

	freshestIndex := w.queue.freshestIndex()
	freshest := w.queue[freshestIndex]
	if !taskOlder(queuedTask, freshest) {
		w.mu.Unlock()
		return resultDropped, nil
	}

	displaced := w.queue.remove(freshestIndex)
	delete(w.pending, displaced.trip.ID)
	heap.Push(&w.queue, queuedTask)
	w.pending[staleTrip.ID] = pendingTask{token: queuedTask.token, item: queuedTask}
	w.cond.Signal()
	w.mu.Unlock()
	return resultQueued, &displaced.trip
}

func (w *Worker) runWorker(ctx context.Context) {
	defer w.workers.Done()
	for {
		queuedTask, ok := w.dequeue(ctx)
		if !ok {
			return
		}
		w.process(ctx, queuedTask)
		w.finish(queuedTask)
	}
}

func (w *Worker) dequeue(ctx context.Context) (task, bool) {
	w.mu.Lock()
	for len(w.queue) == 0 && !w.queueClosed {
		w.cond.Wait()
	}
	if len(w.queue) == 0 {
		w.mu.Unlock()
		return task{}, false
	}
	item := heap.Pop(&w.queue).(*task)
	pending := w.pending[item.trip.ID]
	if pending.token == item.token {
		pending.item = nil
		w.pending[item.trip.ID] = pending
	}
	w.mu.Unlock()
	w.metrics.QueueChanged(ctx, -1)
	return *item, true
}

func (w *Worker) process(ctx context.Context, queuedTask task) {
	link := trace.Link{SpanContext: queuedTask.scanSpan}
	ctx, span := otel.Tracer("tripgo/worker/positions").Start(
		ctx,
		"request trip position",
		trace.WithLinks(link),
	)
	defer span.End()

	w.metrics.ActiveWorkersChanged(ctx, 1)
	defer w.metrics.ActiveWorkersChanged(context.WithoutCancel(ctx), -1)

	startedAt := time.Now()
	requestContext, cancel := context.WithTimeout(ctx, w.requestTimeout)
	err := w.push.RequestPosition(requestContext, queuedTask.trip.DriverID, queuedTask.trip.ID)
	cancel()
	result := resultSuccess
	if err != nil {
		result = resultError
		span.RecordError(err)
		w.logger.ErrorContext(ctx, "request trip position", slog.String("trip_id", queuedTask.trip.ID.String()), slog.Any("error", err))
	}
	w.metrics.PositionRequest(ctx, result, time.Since(startedAt))
}

func (w *Worker) finish(queuedTask task) {
	w.mu.Lock()
	if w.pending[queuedTask.trip.ID].token == queuedTask.token {
		delete(w.pending, queuedTask.trip.ID)
	}
	w.mu.Unlock()
}
