package health

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/log"
)

const (
	// DefaultInterval defines how often health check will run.
	// If DefaultTimeout > DefaultInterval, the next health check will be
	// dropped because of unfinished health check.
	DefaultInterval = 5 * time.Second

	// DefaultTimeout defines how long health check will run.
	// If the check function cannot be finished in time, a timeout error will be returned.
	DefaultTimeout = time.Second
)

// State is health state code
type State int

const (
	StateUnknown   State = 0
	StateHealthy         = 1
	StateUnhealthy       = 2
)

// Status is health status
type Status struct {
	State  State
	Error  error
	Source string // Source of error, usually is health check name
}

func (s *Status) Equal(d *Status) bool {
	return s.State == d.State && s.Error == d.Error && s.Source == d.Source
}

// CheckFunc is function which returns an error.
// It's an abstract definition of checking process. A customized checking process should be a CheckFunc.
//
// Any panic in CheckFunc will cause process exit with error.
type CheckFunc func(ctx context.Context) error

// Checker is an interface that implemented healtch check callback for health check runner.
type Checker interface {
	// HealthCheck checks service health status
	HealthCheck(ctx context.Context) error
}

type checkController interface {
	// Check checks health status
	Check(ctx context.Context) Status
}

type Reporter interface {
	// LastStatus gets last health status
	LastStatus() Status

	// StatusReportChan returns health status report channel
	// There is only one channel for each reporter instance
	// The message in channel should be consumed immediately, otherwise the message will be dropped
	StatusReportChan() chan Status
}

type Runner interface {
	Reporter

	// AddCheck adds health check callback
	AddCheck(name string, cf CheckFunc)

	// Start starts health check runner and returns first result
	Start() Status

	// Stop stops health check runner
	Stop()
}

type unaryChecker struct {
	checkController

	name string
	cf   CheckFunc
}

func newUnaryCheck(name string, cf CheckFunc) *unaryChecker {
	return &unaryChecker{
		name: name,
		cf:   cf,
	}
}

// Check method of checker will run the CheckFunc with context.
func (c *unaryChecker) Check(ctx context.Context) Status {
	var newCtx context.Context
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		newCtx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	} else {
		newCtx = ctx
	}

	ch := make(chan Status, 1)
	go func() {
		defer close(ch)

		// Handle the panic case and still write the error log
		defer func() {
			if v := recover(); v != nil {
				var panicErr error
				rawErr, ok := v.(error)
				if ok {
					panicErr = errors.Wrap(rawErr, "panic")
				} else {
					panicErr = errors.New("panic")
				}

				log.Logger.Error().
					Stack().
					Err(panicErr).
					Str("source", c.name).
					Interface("panic", v).
					Msg("health_check_panic")

				// reraise error
				panic(v)
			}
		}()

		err := c.cf(newCtx)
		if err == nil {
			ch <- Status{
				State: StateHealthy,
			}
		} else {
			ch <- Status{
				State:  StateUnhealthy,
				Error:  err,
				Source: c.name,
			}
		}
	}()

	select {
	case status := <-ch:
		return status
	case <-newCtx.Done():
		return Status{
			State:  StateUnknown,
			Source: c.name,
		}
	}
}

func newCompositeCheck() *compositeChecker {
	return &compositeChecker{
		cl: make([]checkController, 0),
	}
}

type compositeChecker struct {
	checkController

	cl []checkController
}

func (c *compositeChecker) AddChecker(checker checkController) {
	c.cl = append(c.cl, checker)
}

// Check runs underlying checkers concurrently. Wait until all checker are finished.
func (c *compositeChecker) Check(ctx context.Context) Status {
	wg := &sync.WaitGroup{}
	finalStatus := Status{
		State: StateHealthy,
	}
	for _, c := range c.cl {
		wg.Add(1)
		go func(c checkController) {
			defer wg.Done()
			status := c.Check(ctx)
			if status.State != StateHealthy {
				if status.State == StateUnhealthy || finalStatus.State == StateHealthy {
					finalStatus = status
				}
			}
		}(c)
	}
	wg.Wait()
	return finalStatus
}

type runOptions struct {
	name         string
	interval     time.Duration
	timeout      time.Duration
	panicOnError bool
}

// RunOption is used by a health runner. A runner will apply these options
// when running the health check.
type RunOption func(*runOptions)

// WithName sets name of health check runner.
func WithName(name string) RunOption {
	return func(options *runOptions) {
		options.name = name
	}
}

// WithCheckInterval sets the period of health check interval.
func WithCheckInterval(interval time.Duration) RunOption {
	return func(options *runOptions) {
		options.interval = interval
	}
}

// WithCheckTimeout sets the timeout of health check.
func WithCheckTimeout(d time.Duration) RunOption {
	return func(options *runOptions) {
		options.timeout = d
	}
}

// WithPanicOnError indicates if we need to panic and crash the process when there is
// any error during health check.
//
// If there is no error during health check, this option doesn't have any affect.
func WithPanicOnError(b bool) RunOption {
	return func(options *runOptions) {
		options.panicOnError = b
	}
}

func applyOptions(options ...RunOption) *runOptions {
	opts := &runOptions{
		interval: DefaultInterval,
		timeout:  DefaultTimeout,
	}
	for _, o := range options {
		o(opts)
	}
	return opts
}

// uint32 boolean values
const (
	uFalse = iota
	uTrue
)

type runnerImpl struct {
	Runner

	opts              *runOptions
	stop              chan struct{}
	c                 *compositeChecker
	once              sync.Once
	lastStatus        Status
	runningCheckCycle uint32
	reportChan        chan Status
}

// NewRunner creates a health runner.
func NewRunner(options ...RunOption) Runner {
	opts := applyOptions(options...)
	return &runnerImpl{
		opts:              opts,
		stop:              make(chan struct{}),
		c:                 newCompositeCheck(),
		lastStatus:        Status{State: StateUnknown},
		runningCheckCycle: uFalse,
		reportChan:        make(chan Status, 1),
	}
}

func (r *runnerImpl) LastStatus() Status {
	return r.lastStatus
}

func (r *runnerImpl) StatusReportChan() chan Status {
	return r.reportChan
}

func (r *runnerImpl) AddCheck(name string, cf CheckFunc) {
	checker := newUnaryCheck(name, cf)
	r.c.AddChecker(checker)
}

func (r *runnerImpl) Start() Status {
	// Run health check immediately, wait for result
	r.runHealthCheck()

	// Run health check loop in new go routine
	go r.run()

	// Return health check result in first round
	return r.lastStatus
}

func (r *runnerImpl) Stop() {
	close(r.stop)
}

// run starts an infinite loop to check the health status periodically.
func (r *runnerImpl) run() {
	r.once.Do(func() {
		tick := time.Tick(r.opts.interval)
		for {
			select {
			case <-tick:
				go r.runHealthCheck()
			case <-r.stop:
				return
			}
		}
	})
}

// runHealthCheck runs single health check cycle
func (r *runnerImpl) runHealthCheck() {
	if !atomic.CompareAndSwapUint32(&r.runningCheckCycle, uFalse, uTrue) {
		log.Logger.Warn().Str("name", r.opts.name).Msg("health_check_busy")
		return
	}
	defer atomic.CompareAndSwapUint32(&r.runningCheckCycle, uTrue, uFalse)

	var status Status
	ctx, cancel := context.WithTimeout(context.Background(), r.opts.timeout)
	defer cancel()
	status = r.c.Check(ctx)
	if status.Equal(&r.lastStatus) {
		return
	}

	ctxLogger := log.Logger.With().Str("name", r.opts.name).Int("state", int(status.State))
	if status.State != StateHealthy {
		ctxLogger = ctxLogger.Str("source", status.Source)
		if status.Error != nil {
			ctxLogger = ctxLogger.Err(status.Error)
		}
	}
	l := ctxLogger.Logger()
	if status.State == StateHealthy {
		l.Info().Msg("health_check_healthy")
	} else if status.State == StateUnknown {
		l.Error().Msg("health_check_unknown_state")
	} else {
		if r.opts.panicOnError {
			l.Error().Msg("health_check_panic_on_error")
			panicErr := errors.Wrap(status.Error, fmt.Sprintf("health_check_error(%s)", status.Source))
			panic(panicErr)
		} else {
			l.Error().Msg("health_check_unhealthy")
		}
	}

	r.lastStatus = status
	select {
	case r.reportChan <- status:
		// success
	default:
		l.Error().Msg("health_check_status_report_dropped_for_blocking")
	}
}
