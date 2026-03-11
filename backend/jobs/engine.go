package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/dbcommon"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/job"
)

const (
	UnlimitedRetriesLimit = 10
	ShutdownTimeout       = 15 * time.Second
)

type Engine struct {
	App                 *common.App
	Registry            *Registry
	newJobChan          chan struct{}
	waitingForJobsChan  chan struct{}
	requestShutdownChan chan struct{}
	shutdownCtx         context.Context
	cancelShutdownCtx   context.CancelFunc
	listenOnce          sync.Once
	shutdownOnce        sync.Once
	mu                  sync.Mutex
}

func NewEngine(registry *Registry) *Engine {
	return &Engine{
		App:                 registry.App,
		Registry:            registry,
		newJobChan:          make(chan struct{}, 1),
		waitingForJobsChan:  make(chan struct{}),
		requestShutdownChan: make(chan struct{}),
	}
}

type completedJob struct {
	Object    *ent.Job
	StartTime time.Time
	Err       common.WrappedError
}

func (engine *Engine) Listen() {
	engine.listenOnce.Do(func() {
		completedJobChan := make(chan completedJob, min(engine.App.Env.MAX_TOTAL_JOB_WEIGHT, 100))
		currentWeight := 0

	listenLoop:
		for {
			runJobsResult := engine.runJobs(currentWeight, completedJobChan)
			currentWeight = runJobsResult.newWeight
			if runJobsResult.shouldShutdown {
				break listenLoop
			}
			// TODO: check for stalled jobs, do a similar thing to scheduled jobs. Wait until they should have finished
			// Maybe schedule a job to check for stalled jobs every 5 minutes or so
			// because checking after all the jobs have finished running might be too late if there are a lot

			maxWaitTime := engine.App.Env.JOB_POLL_INTERVAL
			nextJob, stdErr := dbcommon.WithReadTx(
				context.TODO(), engine.App.Database,
				func(tx *ent.Tx, ctx context.Context) (*ent.Job, error) {
					return tx.Job.Query().
						Where(job.StatusEQ("pending")).
						Order(ent.Asc(job.FieldDueAt)).
						First(ctx)
				},
			)
			if stdErr == nil {
				timeUntil := time.Until(nextJob.DueAt)
				if timeUntil < maxWaitTime {
					maxWaitTime = timeUntil
				}
			} else if !ent.IsNotFound(stdErr) {
				engine.App.Logger.Error("failed to query next due job", "error", stdErr)
				time.Sleep(250 * time.Millisecond)
				continue
			}

			for currentWeight > 0 {
				select {
				case completedJob := <-completedJobChan:
					engine.handleCompletedJob(completedJob, &currentWeight)
				case <-engine.requestShutdownChan:
					break listenLoop
				}
			}

			engine.mu.Lock()
			oldChan := engine.waitingForJobsChan
			engine.waitingForJobsChan = make(chan struct{})
			close(oldChan)
			engine.mu.Unlock()

			select {
			case <-time.After(maxWaitTime):
			case <-engine.newJobChan:
			case <-engine.requestShutdownChan:
				break listenLoop
			}
		}
		close(engine.requestShutdownChan)
		for currentWeight > 0 {
			completedJob := <-completedJobChan
			engine.handleCompletedJob(completedJob, &currentWeight)
		}

		engine.mu.Lock()
		engine.cancelShutdownCtx()
		engine.mu.Unlock()
	})
}

type runJobsResult struct {
	shouldShutdown bool
	newWeight      int
}

func (engine *Engine) runJobs(currentWeight int, completedJobChan chan completedJob) runJobsResult {
	for {
		currentJob, stdErr := dbcommon.WithReadTx(
			context.TODO(), engine.App.Database,
			func(tx *ent.Tx, ctx context.Context) (*ent.Job, error) {
				return tx.Job.Query().
					Where(job.StatusEQ("pending"), job.DueAtLTE(time.Now())).
					Order(ent.Asc(job.FieldStatus), ent.Desc(job.FieldPriority), ent.Asc(job.FieldDueAt)).
					First(ctx)
			},
		)
		if stdErr != nil {
			if ent.IsNotFound(stdErr) {
				return runJobsResult{
					shouldShutdown: false,
					newWeight:      currentWeight,
				}
			}
			// It might be good to shut down if this happens enough times in a row
			// But if jobs aren't running, then downloads should be blocked because the endpoint checks that
			// enough messages have been sent recently
			// And those won't send if the job engine is failing
			// TODO: handle shutdown signals during this potential loop
			engine.App.Logger.Error("failed to get current job to run", "error", stdErr)
			engine.App.Clock.Sleep(250 * time.Millisecond)
			continue
		}

		maxTotalWeightToStart := engine.App.Env.MAX_TOTAL_JOB_WEIGHT - currentWeight
		for currentWeight > maxTotalWeightToStart && currentWeight > 0 {
			select {
			case completedJob := <-completedJobChan:
				engine.handleCompletedJob(completedJob, &currentWeight)
			case <-engine.requestShutdownChan:
				return runJobsResult{
					shouldShutdown: true,
					newWeight:      currentWeight,
				}
			}
		}
		select {
		case <-engine.requestShutdownChan:
			return runJobsResult{
				shouldShutdown: true,
				newWeight:      currentWeight,
			}
		default:
		}

		stdErr = dbcommon.WithWriteTx(
			context.TODO(), engine.App.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				now := engine.App.Clock.Now()
				return tx.Job.UpdateOneID(currentJob.ID).
					SetStatus("running").SetStartedAt(now).
					Exec(ctx)
			},
		)
		if stdErr != nil {
			engine.App.Logger.Error(
				"failed to update job status to running",
				"jobID", currentJob.ID,
				"error", stdErr,
			)
			engine.App.Clock.Sleep(250 * time.Millisecond)
			continue
		}

		currentWeight += currentJob.Weight
		jobDefinition, ok := engine.Registry.jobs[common.GetVersionedType(currentJob.Type, currentJob.Version)]
		if ok {
			if jobDefinition.NoParallelize {
				engine.runJob(jobDefinition, currentJob, completedJobChan)
			} else {
				go engine.runJob(jobDefinition, currentJob, completedJobChan)
			}
		} else { // Note: this shouldn't happen
			completedJobChan <- completedJob{
				Object:    currentJob,
				StartTime: engine.App.Clock.Now(),
				Err:       ErrWrapperRunJob.Wrap(ErrUnknownJobType),
			}
		}
		// Otherwise continue processing jobs
	}
}
func (engine *Engine) handleCompletedJob(completedJob completedJob, currentWeightPtr *int) {
	logger := engine.App.Logger.With(
		"jobID",
		completedJob.Object.ID,
		"jobType",
		common.GetVersionedType(completedJob.Object.Type, completedJob.Object.Version),
	)

	*currentWeightPtr -= completedJob.Object.Weight
	if completedJob.Err == nil {
		stdErr := dbcommon.WithWriteTx(context.TODO(), engine.App.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				return tx.Job.DeleteOneID(completedJob.Object.ID).Exec(ctx)
			},
		)
		if stdErr != nil {
			logger.Error("failed to delete job", "error", stdErr)
		}
		logger.Info(
			"job completed",
			"totalRetries", completedJob.Object.Retries,
			"runDuration", engine.App.Clock.Since(completedJob.StartTime),
		)
	} else {
		if completedJob.Err.MaxRetries() < 0 {
			if completedJob.Err.MaxRetries() == -1 {
				logger.Warn("error returned by job has unlimited retries (-1). Setting to UnlimitedRetriesLimit")
				completedJob.Err.SetMaxRetriesMut(UnlimitedRetriesLimit)
			}
			completedJob.Err.SetMaxRetriesMut(0)
		}
		if completedJob.Err.MaxRetries() > 0 {
			if completedJob.Err.RetryBackoffBase() < time.Second {
				logger.Warn(
					"error returned by job has a low base retry backoff. Did you forget to wrap it in WithRetries?",
					"retryBackoffBase", completedJob.Err.RetryBackoffBase(),
				)
			}
		}
		retriedFraction := completedJob.Object.RetriedFraction
		if completedJob.Err.MaxRetries() > 0 {
			retriedFraction += 1 / float64(completedJob.Err.MaxRetries()+1)
		}
		shouldRetry := retriedFraction >= 1-common.BackoffMaxRetriesEpsilon || completedJob.Err.MaxRetries() < 1
		backoff := common.CalculateBackoff(
			completedJob.Object.Retries,
			completedJob.Err.RetryBackoffBase(),
			completedJob.Err.RetryBackoffMultiplier(),
		)
		sendJobSignal := false
		stdErr := dbcommon.WithWriteTx(context.TODO(), engine.App.Database,
			func(tx *ent.Tx, ctx context.Context) error {
				if shouldRetry {
					logger.Error(
						"job failed",
						"error", completedJob.Err,
						"totalRetries", completedJob.Object.Retries,
						"runDuration", engine.App.Clock.Since(completedJob.StartTime),
					)
					return tx.Job.UpdateOneID(completedJob.Object.ID).
						SetStatus("failed").
						Exec(ctx)
				} else {
					logger.Info(
						"queueing job for retry",
						"backoff", backoff,
						"error", completedJob.Err,
						"totalRetries", completedJob.Object.Retries,
					)
					sendJobSignal = true
					return tx.Job.UpdateOneID(completedJob.Object.ID).
						SetStatus("pending").
						SetDueAt(engine.App.Clock.Now().Add(backoff)).
						AddRetries(1).
						SetRetriedFraction(retriedFraction).
						SetLoggedStallWarning(false).
						Exec(ctx)
				}
			},
		)
		if stdErr != nil {
			logger.Error("failed to mark job as failed / reset to pending", "error", stdErr)
		} else if sendJobSignal {
			select {
			case engine.newJobChan <- struct{}{}:
			default:
			}
		}
	}
}
func (engine *Engine) runJob(
	jobDefinition *Definition, job *ent.Job,
	completedJobChan chan completedJob,
) {
	logger := engine.App.Logger.With(
		"jobID",
		job.ID,
		"jobType",
		common.GetVersionedType(job.Type, job.Version),
	)
	logger.Info("running job")
	stdErr := jobDefinition.Handler(&Context{
		Job:        job,
		Definition: jobDefinition,
		Context:    context.TODO(), // TODO: put the logger in the context
		Logger:     logger,
		Body:       job.Body,
	})
	completedJobChan <- completedJob{
		Object:    job,
		StartTime: engine.App.Clock.Now(),
		Err:       ErrWrapperRunJob.Wrap(stdErr),
	}
}

func (engine *Engine) WaitForJobs() {
	engine.mu.Lock()
	waitingForJobsChan := engine.waitingForJobsChan
	engine.mu.Unlock()
	<-waitingForJobsChan
}

func (engine *Engine) Shutdown() {
	go engine.Listen()
	engine.shutdownOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		engine.mu.Lock()
		engine.shutdownCtx = ctx
		engine.cancelShutdownCtx = cancel
		engine.mu.Unlock()

		engine.App.Logger.Info("requesting job engine shutdown")
		select {
		case engine.requestShutdownChan <- struct{}{}:
			engine.App.Logger.Info("waiting for job engine to finish jobs...")
			<-ctx.Done()
		case <-ctx.Done():
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			engine.App.Logger.Error("job engine shutdown timed out")
		} else {
			engine.App.Logger.Info("job engine stopped")
		}
	})
}

func (engine *Engine) Enqueue(
	versionedType string,
	body any,
	ctx context.Context,
) (*ent.Job, common.WrappedError) {
	return engine.EnqueueWithModifier(versionedType, body, nil, ctx)
}
func (engine *Engine) EnqueueEncoded(
	versionedType string,
	encodedBody json.RawMessage,
	ctx context.Context,
) (*ent.Job, common.WrappedError) {
	return engine.EnqueueEncodedWithModifier(versionedType, encodedBody, nil, ctx)
}

func (engine *Engine) EnqueueWithModifier(
	versionedType string,
	body any,
	modifier func(jobCreate *ent.JobCreate),
	ctx context.Context,
) (*ent.Job, common.WrappedError) {
	_, ok := engine.Registry.jobs[versionedType]
	if !ok {
		return nil, ErrWrapperEnqueue.Wrap(ErrUnknownJobType)
	}
	encoded, wrappedErr := engine.Registry.Encode(
		versionedType,
		body,
	)
	if wrappedErr != nil {
		return nil, ErrWrapperEnqueue.Wrap(wrappedErr)
	}

	return engine.EnqueueEncodedWithModifier(versionedType, encoded, modifier, ctx)
}
func (engine *Engine) EnqueueEncodedWithModifier(
	versionedType string,
	encodedBody json.RawMessage,
	modifier func(jobCreate *ent.JobCreate),
	ctx context.Context,
) (*ent.Job, common.WrappedError) {
	jobDefinition, ok := engine.Registry.jobs[versionedType]
	if !ok {
		return nil, ErrWrapperEnqueue.Wrap(ErrUnknownJobType)
	}

	jobType, version, wrappedErr := common.ParseVersionedType(versionedType)
	if wrappedErr != nil { // This shouldn't happen because of the Encode call but just in case
		return nil, ErrWrapperEnqueue.Wrap(wrappedErr)
	}

	tx := ent.TxFromContext(ctx)
	if tx == nil {
		return nil, ErrWrapperEnqueue.Wrap(ErrNoTxInContext)
	}
	now := engine.App.Clock.Now()
	jobCreate := tx.Job.Create().
		SetType(jobType).
		SetCreatedAt(now).
		SetDueAt(now).
		SetOriginallyDueAt(now).
		SetVersion(version).
		SetPriority(jobDefinition.Priority).
		SetWeight(jobDefinition.Weight).
		SetBody(encodedBody)
	if modifier != nil {
		modifier(jobCreate)
	}
	job, stdErr := jobCreate.Save(ctx)
	if stdErr != nil {
		return nil, ErrWrapperEnqueue.Wrap(ErrWrapperDatabase.Wrap(stdErr))
	}

	// Otherwise the engine will look for the job before it's committed
	tx.OnCommit(func(committer ent.Committer) ent.Committer {
		return ent.CommitFunc(
			func(ctx context.Context, tx *ent.Tx) error {
				stdErr := committer.Commit(ctx, tx)
				if stdErr != nil {
					return stdErr
				}

				select {
				case engine.newJobChan <- struct{}{}:
				default:
				}
				return nil
			},
		)
	})
	return job, nil
}
