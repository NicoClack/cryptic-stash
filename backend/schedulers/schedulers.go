package schedulers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
)

const (
	ShutdownTimeout = 10 * time.Second
)

type Engine struct {
	App   *common.App
	tasks []Task
	// Note: outside of the schedulers package, this should only be used by LoopFuncs to listen for shutdown requests
	RequestShutdownChan chan struct{}
	shutdownCtx         context.Context
	cancelShutdownCtx   context.CancelFunc
	shutdownWg          sync.WaitGroup
	runOnce             sync.Once
	shutdownOnce        sync.Once
	mu                  sync.Mutex
}

func NewEngine(app *common.App) *Engine {
	return &Engine{
		App:                 app,
		tasks:               make([]Task, 0),
		RequestShutdownChan: make(chan struct{}),
	}
}
func (engine *Engine) Register(callback TaskCallback, delayFunc DelayFunc) {
	engine.RegisterTask(NewTask(callback, delayFunc))
}
func (engine *Engine) RegisterJob(versionedName string, delayFunc DelayFunc) {
	engine.RegisterTask(NewJobTask(versionedName, delayFunc))
}
func (engine *Engine) RegisterTask(task Task) {
	engine.tasks = append(engine.tasks, task)
}
func (engine *Engine) Run() {
	engine.runOnce.Do(func() {
		for _, task := range engine.tasks {
			if task.Init != nil {
				task.Init(engine)
			}
			engine.shutdownWg.Go(func() {
				task.Run(engine)
			})
		}
		engine.shutdownWg.Wait()
		engine.mu.Lock()
		engine.cancelShutdownCtx()
		engine.mu.Unlock()
	})
}
func (engine *Engine) Shutdown() {
	engine.shutdownOnce.Do(func() {
		engine.App.Logger.Info("scheduler shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		engine.mu.Lock()
		engine.shutdownCtx = ctx
		engine.cancelShutdownCtx = cancel
		engine.mu.Unlock()
		close(engine.RequestShutdownChan)
		engine.runOnce.Do(func() {
			<-engine.RequestShutdownChan
			engine.mu.Lock()
			engine.cancelShutdownCtx()
			engine.mu.Unlock()
		})

		<-ctx.Done()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			engine.App.Logger.Error("scheduler shutdown timed out")
		} else {
			engine.App.Logger.Info("scheduler stopped")
		}
	})
}
