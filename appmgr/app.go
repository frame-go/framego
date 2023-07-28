package appmgr

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/linxGnu/mssqlx"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/frame-go/framego/client/cache"
	"github.com/frame-go/framego/client/database"
	pulsarclient "github.com/frame-go/framego/client/pulsar"
	"github.com/frame-go/framego/config"
	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/log"
	"github.com/frame-go/framego/utils"
)

type appImpl struct {
	App

	ctx         context.Context
	cancel      context.CancelFunc
	cmd         *cobra.Command
	initOK      *atomic.Bool
	config      *AppConfig
	middlewares *middlewareManager
	jobs        map[string]func(context.Context) error
	services    map[string]Service
	observable  ObservableService
	clients     ClientManager
	databases   database.ClientManager
	caches      cache.ClientManager
	pulsars     pulsarclient.ClientManager
}

// Init initializes application by config
func (a *appImpl) Init() {
	utils.InitRand()

	_ = a.cmd.Execute()
	if !a.initOK.Load() {
		os.Exit(0)
	}

	err := config.InitConfig()
	if err != nil {
		exitWithError("Init Config Error", err)
		return
	}

	log.Init(viper.GetString("log_level"), viper.GetBool("debug"), viper.GetBool("beautify_log"))
	errors.SetGRPCDebugMode(viper.GetBool("debug"))

	err = config.GetStructWithValidation("app", a.config)
	if err != nil {
		log.Logger.Error().Err(err).Msg("parse_app_config_error")
		exitWithError("Parse App Config Error", err)
		return
	}

	runJob := viper.GetString("job")
	if runJob != "" {
		// if specified job in command line, only run this job
		a.config.Jobs = []string{runJob}
		a.config.Services = []ServiceConfig{}
	}

	a.clients, err = newClientManager(a.ctx, a.middlewares, &a.config.Clients)
	if err != nil {
		log.Logger.Error().Err(err).Msg("init_clients_error")
		exitWithError("Init Clients Error", err)
		return
	}

	a.databases, err = database.NewClientManager(a.config.Databases, database.WithLogger(log.Logger))
	if err != nil {
		log.Logger.Error().Err(err).Msg("init_databases_error")
		exitWithError("Init Databases Error", err)
		return
	}

	a.caches, err = cache.NewClientManager(a.config.Caches, cache.WithLogger(log.Logger))
	if err != nil {
		log.Logger.Error().Err(err).Msg("init_caches_error")
		exitWithError("Init Caches Error", err)
		return
	}

	a.pulsars, err = pulsarclient.NewClientManager(a.config.Pulsars, pulsarclient.WithLogger(log.Logger))
	if err != nil {
		log.Logger.Error().Err(err).Msg("init_pulsars_error")
		exitWithError("Init Pulsars Error", err)
		return
	}

	initGin(a.config.Name)
	initGrpc()
	for _, serviceConfig := range a.config.Services {
		a.services[serviceConfig.Name], err = newService(a.ctx, a, a.middlewares, &serviceConfig)
		if err != nil {
			log.Logger.Error().Err(err).Str("service", serviceConfig.Name).Msg("init_service_error")
			exitWithError("Init Service Error", err)
			return
		}
	}
	a.observable = newObservable(a.ctx, a.middlewares, &a.config.Observable, a.services)
}

func (a *appImpl) RegisterMiddleware(name string, middleware Middleware) {
	a.middlewares.RegisterMiddleware(name, middleware)
}

func (a *appImpl) AddJob(name string, job func(context.Context) error) {
	_, ok := a.jobs[name]
	if ok {
		log.Logger.Warn().Str("name", name).Msg("add_job_with_duplicated_name")
	}
	a.jobs[name] = job
}

func (a *appImpl) GetContext() context.Context {
	return a.ctx
}

func (a *appImpl) GetService(name string) Service {
	service, ok := a.services[name]
	if !ok {
		return nil
	}
	return service
}

func (a *appImpl) GetGrpcClientConn(name string) grpc.ClientConnInterface {
	return a.clients.GetGrpcClientConn(name)
}

func (a *appImpl) GetGrpcClientConns(name string) []grpc.ClientConnInterface {
	return a.clients.GetGrpcClientConns(name)
}

func (a *appImpl) GetDatabaseClient(name string) *gorm.DB {
	return a.databases.GetClient(name)
}

func (a *appImpl) GetCacheClient(name string) cache.Client {
	return a.caches.GetClient(name)
}

func (a *appImpl) GetDatabaseSqlxClient(name string) *mssqlx.DBs {
	return a.databases.GetSqlxClient(name)
}

func (a *appImpl) GetPulsarClient(name string) pulsar.Client {
	return a.pulsars.GetClient(name)
}

func (a *appImpl) Run() (err error) {
	// run all jobs
	chJobs := make(chan error, len(a.config.Jobs))
	var wgJobs sync.WaitGroup
	for _, name := range a.config.Jobs {
		job, ok := a.jobs[name]
		if !ok {
			log.Logger.Fatal().Str("name", name).Msg("job_not_found")
		}
		wgJobs.Add(1)
		go func(name string, job func(context.Context) error) {
			defer wgJobs.Done()
			err := job(a.ctx)
			if err != nil {
				log.Logger.Error().Err(err).Str("name", name).Msg("job_exit_with_error")
				chJobs <- errors.Wrap(err, fmt.Sprintf("Job <%s> Exit With Error", name))
			}
		}(name, job)
		log.Logger.Info().Str("job", name).Msg("job_started")
	}

	// convert wait group signal to channel event for select
	if len(a.config.Jobs) > 0 {
		go func() {
			wgJobs.Wait()
			close(chJobs)
		}()
	}

	// run all services
	for name, service := range a.services {
		err = service.Run()
		if err != nil {
			return
		}
		log.Logger.Info().Str("service", name).Msg("service_started")
	}
	err = a.observable.Run()
	if err != nil {
		return
	}
	log.Logger.Info().Msg("observable_started")

	// Wait for interrupt signal to gracefully shut down the server with a timeout
	quit := make(chan os.Signal, 1)

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-a.ctx.Done():
		log.Logger.Warn().Msg("received_exit_command")
		fmt.Println("[Stopping] Received Exit Command.")
	case err = <-chJobs:
		if err != nil {
			fmt.Println("[Exceptional Stopping] Job Exited With Error.")
			return
		}
		log.Logger.Info().Msg("all_jobs_completed")
		fmt.Println("[Stopping] All Jobs Completed.")
		a.cancel()
	case <-quit:
		log.Logger.Warn().Msg("received_stop_signal")
		fmt.Println("[Stopping] Received Stop Signal.")
		a.cancel()
	}

	// wait for stop of all jobs and services
	wgJobs.Wait()
	for _, service := range a.services {
		service.Wait()
	}
	a.observable.Wait()
	log.Logger.Warn().Msg("exit_with_all_services_stopped")
	fmt.Println("[Exit] All Services Stopped.")
	return
}

func (a *appImpl) RunOrExit() {
	err := a.Run()
	if err != nil {
		exitWithError("Run Application Error", err)
	}
}

func NewApp(info *AppInfo) App {
	app := &appImpl{
		initOK:      atomic.NewBool(false),
		config:      &AppConfig{},
		middlewares: newDefaultMiddlewareManager(),
		jobs:        make(map[string]func(context.Context) error),
		services:    make(map[string]Service),
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	cmd := &cobra.Command{
		Use:     info.Name,
		Version: info.Version,
		Long:    info.Description,
		Run: func(cmd *cobra.Command, args []string) {
			app.initOK.Store(true)
		},
	}
	config.BindArgs(cmd)
	app.cmd = cmd
	return app
}

func exitWithError(msg string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "[Exceptional Exit] %s: %v\n", msg, err)
	os.Exit(1)
}
