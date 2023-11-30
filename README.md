# Framego Manual

`framego` is a versatile and easy-to-use Go library designed for modern backend development. It simplifies the development process by providing a framework and essential libraries for building robust and scalable backend systems, from gRPC APIs, HTTP APIs to job processing. 

## Key Features

- **Dual Protocol Support**: Seamlessly implement gRPC and HTTP APIs. You can easily implement gRPC services and expose corresponding HTTP APIs automatically. Provide a variety of middlewares for gRPC and HTTP services.
- **Job Processing Capabilities**: Support background tasks and job processing.
- **Rich Libraries**: Come with essential libraries, including configuration management, database/cache/queue clients, in-memory caching solutions, advanced error handling and logging utilities.
- **Ease of Use**: Intuitive setup and use, allowing for quick starting your projects.

## Repositories

- Framego library: [https://github.com/frame-go/framego](https://github.com/frame-go/framego)
- Cookiecutter project template: [https://github.com/frame-go/cookiecutter-framego](https://github.com/frame-go/cookiecutter-framego)
- Protoc plugin: [https://github.com/frame-go/protoc-gen-framego](https://github.com/frame-go/protoc-gen-framego)

## Getting Started

### Prerequisites

Ensure you have installed and properly configured these tools in Shell before initializing project:
- curl
- git
- go
- cookiecutter
- buf
- air (optional)

#### Git

Ensure you have installed and properly setup Git configuration.

Check whether you can directly clone repository from Git remote without input authentication information.

#### Go

Ensure you have installed and properly setup Golang development environment.

Check whether you can go get modules from private repository in shell.

#### Cookiecutter

Cookiecutter is a command-line utility that creates projects from project templates.

Cookiecutter is built by Python. You can use pip to install.

```bash
pip install cookiecutter
```

If you haven't installed pip, follow [this guide](https://pip.pypa.io/en/stable/installation/) to install.

#### Buf

Buf is a building tool to make Protobuf reliable and easy to use.

You can follow [this guide](https://buf.build/docs/installation) to install.

#### Air

Air is a live-reloading command line utility for Go applications in development. It is optional for this framework. 

You can follow [this guide](https://github.com/cosmtrek/air#installation) to install.

```bash

curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh
```

### Create Project

Run this command to create a project folder in the current working directory:

```bash
cookiecutter https://github.com/frame-go/cookiecutter-framego
```

You will need to provide the following template variables:

| Variable Name        | Description                                                                                                                                                                            | Example                                 |
|----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------|
| project_title        | Project name, in `CapitalizedCase` format. <br>This name will be used in project description and documents. <br>Suggest to be full name of project.                                    | `Identity and Access Management System` |
| project_name         | Project name, in `kebab-case` format. <br>This name will be used as project root folder name.                                                                                          | `iam-server`                            |
| app_name             | App name, in `snake_case` format. <br>This name will be used as an app name.                                                                                                           | `security_core`                         |
| app_package_name     | App package name, in `flatcase`. <br>This name will be used as project root cmd package name and binary name. <br>Suggest aligning with app name, using a single word or abbreviation. | `securitycore`                          |
| service_name         | Service name, in `snake_case` format. <br>This name will be used as service name, DB name.                                                                                             | `security_core`                         |
| service_package_name | Sedervice package name, in `flatcase`. <br>This name will be used as the api package name. <br>Suggest aligning with service name, using a single word or abbreviation.                | `securitycore`                          |
| go_module            | Full go module path. <br>Suggest to end with project name.                                                                                                                             | `github.com/example/demo-server`        |
| go_version           | Golang version, will be used in makefile and CI.                                                                                                                                       | `1.16`                                  |

### Initialize Project

Run this command inside the project folder to initialize and install Golang tools:

```bash
make init
```

### Build Project

Generate codes and compile the project:

```bash
make
```

### Setup DB

Ensure you have MySQL/MariaDB installed and running locally.

Run this command inside the project folder to create a sample DB and tables and grant permission to the test user:

```bash
mysql < db/*.sql
mysql -v -e "CREATE USER 'test'@'127.0.0.1' IDENTIFIED BY 'password'; GRANT ALL ON *.* TO 'test'@'127.0.0.1';"
```

You can change the DB connection config inside `configs/debug/config.yaml`.

### Run Project

You can run the air command inside the project folder to run your project in live reload mode:

```bash
air
```

Or you can simply run the compiled executable inside the bin folder.

### Make Commands

The template provides many useful commands by makefile. You can run make commands in the project folder.

| Command               | Description                                                                                                        |
|-----------------------|--------------------------------------------------------------------------------------------------------------------|
| make init             | Initialize project and install tools.                                                                              |
| make / make all       | Generate code and compile project. <br>Equal to `make fmt && make generate && make build`.                         |
| make build            | Compile project.                                                                                                   |
| make clean            | Clear compiled files.                                                                                              |
| make go-generate      | Run go generate to generate go code.                                                                               |
| make buf-generate     | Run buf generate to generate code from protobuf files and export dependencies protobuf files to buf_vendor folder. |
| make generate         | Equal to make buf-generate && make go-generate .                                                                   |
| make fmt              | Run go fmt and go vet format and check go code.                                                                    |
| make go-lint          | Run golangci-lint analysis go code.                                                                                |
| make buf-lint         | Run buf lint to analysis protobuf code.                                                                            |
| make lint             | Run all static code analysis. Equal to make fmt && make go-lint && make buf-lint .                                 |
| make check-go-version | Check whether golang version matches requirement.                                                                  |

## Project Structure

Root folder name: `{{project_name}}`

| Path                                                        | Description                                                                                                                                                                                                                                        |
|-------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| cmd                                                         | Main applications. <br>Each executable should be in a separated package directory. <br>It's common to have a small main function that imports and invokes the code from other packages and nothing else.                                           |
| cmd/{{app_package_name}}/main.go                            | Application default executable.                                                                                                                                                                                                                    |
| api                                                         | API and protocol definitions and generated codes. <br>The data definition can be used by domain and handlers package.                                                                                                                              |
| api/{{service_package_name}}/{{service_package_name}}.proto | Service protobuf definition.                                                                                                                                                                                                                       |
| buf_vendor                                                  | Protobuf dependencies.                                                                                                                                                                                                                             |
| configs                                                     | Local configuration files                                                                                                                                                                                                                          |
| configs/debug/config.yaml                                   | Default configuration files for debug                                                                                                                                                                                                              |
| db                                                          | Database creation and migrations sql files.                                                                                                                                                                                                        |
| docs                                                        | Documents for the project.                                                                                                                                                                                                                         |
| internal                                                    | Private application and library code. <br>This is the code you don't want others importing in their applications or libraries.                                                                                                                     |
| internal/models                                             | Data models definitions for Data Access layer. <br>The models can be used by db, cache and domain package.                                                                                                                                         |
| internal/db                                                 | Database data access layer. CRUD for database.                                                                                                                                                                                                     |
| internal/cache                                              | Cache data access layer. CRUD for in memory cache and distributed cache.                                                                                                                                                                           |
| internal/domain                                             | Business logic layer. <br>This package contains major business logic, invokes code from db and cache packages, and provides functions for handlers.                                                                                                |
| internal/handlers                                           | Presentation layer. API handlers for HTTP and GRPC interfaces. <br>This package should only contains data validation and conversion, and invokes code from domain package. This package should not invokes code from models, db or cache packages. |
| tools                                                       | External tools used by project.                                                                                                                                                                                                                    |
| tools.go                                                    | Dummy go file to include external tool dependencies.                                                                                                                                                                                               |
| go.mod                                                      | Go module dependencies configuration.                                                                                                                                                                                                              |
| Makefile                                                    | Make command configuration.                                                                                                                                                                                                                        |
| .gitignore                                                  | Git ignore file configurations.                                                                                                                                                                                                                    |
| .golangci.yaml                                              | Golangci-lint configurations.                                                                                                                                                                                                                      |
| .dockerignore                                               | Docker ignore file configurations for docker build                                                                                                                                                                                                 |
| Dockerfile                                                  | Docker build configurations.                                                                                                                                                                                                                       |
| buf.yaml                                                    | Buf configurations, includes lint rules and dependencies.                                                                                                                                                                                          |
| buf.gen.yaml                                                | Buf generate configurations.                                                                                                                                                                                                                       |
| .air.toml                                                   | Air configuration                                                                                                                                                                                                                                  |

### Configuration

Applications built from the framework accept configurations from command line arguments, environment variables, configuration file, and Apollo. 

The priority of them is: command line arguments > environment variables > configuration file > Apollo.

#### Basic Configurations

Below are basic configuration items supported:

| Configuration Item                                 | Command Line Argument  | Environment Variable                    | Configuration Key | Default Value                                      |
|----------------------------------------------------|------------------------|-----------------------------------------|-------------------|----------------------------------------------------|
| Configuration file path                            | `-c`, `--config-path`  | `CONFIG_PATH`                           | N/A               | None                                               |
| Whether to enable debug mode                       | `-d`, `--debug`        | `DEBUG`                                 | `debug`           | `false`                                            |
| Whether to enable human-friendly, colorized log    | `-b`, `--beautify-log` | `BEAUTIFY_LOG`                          | `beautify_log`    | `FALSE`                                            |
| Minimal log level: trace, debug, info, warn, error | `-l`, `--log-level`    | `LOG_LEVEL`                             | `log_level`       | `debug` (if debug==true), `info` (if debug==false) |
| Run job                                            | `-j`, `--job`          | `JOB`                                   | `job`             | `task_executor`                                    |
| Apollo server endpoint                             | `--apollo-server`      | `APOLLO_SERVER _APOLLO_SERVER_`         | N/A               | None                                               |
| Apollo app ID                                      | `--apollo-app-id`      | `APOLLO_APP_ID _APOLLO_APP_ID_`         | N/A               | None                                               |
| Apollo app access key secret                       | `--apollo-access-key`  | `APOLLO_ACCESS_KEY _APOLLO_ACCESS_KEY_` | N/A               | None                                               |
| Apollo cluster                                     | `--apollo-cluster`     | `APOLLO_CLUSTER _APOLLO_CLUSTER_`       | N/A               | `default`                                          |
| Apollo namespace                                   | `--apollo-namespace`   | `APOLLO_NAMESPACE _APOLLO_NAMESPACE_`   | N/A               | `config.yaml`                                      |

### App Configuration

To run the application, app configuration must be provided. It can come from either a local config file or remote config (Apollo).
- For local config file, the file path must be provided by command line argument (`--config-path`) or environment variable (`CONFIG_PATH`). The framework will try to load the local config file if the config path exists.
- For remote config, the Apollo loading arguments must be provided by command line arguments (`--apollo-*`) or environment variable (`_APOLLO_*_`). The framework will try to load Apollo config if Apollo server endpoint and app ID exist. 

For the same config key, values in the local config file have higher precedence over remote config. 

Besides basic configuration items, you can configure items under `app` for services and clients configurations.

Below is a sample app configuration `config.yaml`:

```yaml
debug: true
beautify_log: true
app:
  name: sample
  observable:
    endpoints:
      http: ":8080"
    modules:
      - pprof
      - metrics
      - swagger
      - channelz
      - grpcui
  services:
    - name: sample
      endpoints:
        grpc: ":9000"
        http: ":8000"
      security:
        grpc:
          key: ./keys/service.pem
          cert: ./keys/service.crt
          ca: ./keys/ca.crt
      middlewares:
        - recovery
        - open_tracing
        - metrics
        - context_logger
        - log_request
        - name: access_control
          policy: ./acl.csv
        - request_validation
  jobs:
    - sample_job
  clients:
    grpc:
      middlewares:
        - open_tracing
        - metrics
      servers:
        - name: auth
          endpoint: "127.0.0.1:9000"
          security:
            key: ./keys/service.pem
            cert: ./keys/service.crt
            ca: ./keys/ca.crt
    databases:
      - name: sample
        database: "sample_db"
        user: "root"
        password: ""
        masters: ["127.0.0.1:3306"]
        slaves: []
    caches:
      - name: sample
        type: redis
        address: "127.0.0.1:6379"
    pulsars:
      - name: sample
        url: "pulsar://127.0.0.1:6650"
        token: "zFbeuKF3jqjfxkQFfOoMeQ"
```

A project is an app, it can has many GRPC/HTTP services under an app, and an optionalobservable service for debug, monitoring, etc. Services are listen on different ports.

Below are configuration items under `app`.

| Path                                 | Description                                                                                                                 | Example                               |
|--------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| name                                 | Name of app                                                                                                                 | `iam`                                 |
| observable                           | Built-in observable services for debug, monitoring, etc.                                                                    |                                       |
| observable.endpoint.https            | HTTP endpoint for observable service.                                                                                       | `:8080`                               |
| observable.modules                   | Enable built-in observable modules. <br>Details of available modules refer to below.                                        | `- pprof`                             |
| services                             | List of application services.                                                                                               |                                       |
| services[].name                      | Name of service                                                                                                             | `iam`                                 |
| services[].endpoint.grpc             | gRPC endpoint for this service.                                                                                             | `:9000`                               |
| services[].endpoint.http             | HTTP endpoint for this service.                                                                                             | `:8000`                               |
| services[].security.grpc             | gRPC server TLS configuration. <br>Optional, accept insecure connection if not configured.                                  |                                       |
| services[].security.grpc.key         | TLS server key.                                                                                                             | `./keys/service.pem`                  |
| services[].security.grpc.cert        | TLS server certificate chain.                                                                                               | `./keys/service.crt`                  |
| services[].security.grpc.ca          | TLS CA for verifying clients certificates.                                                                                  | `./key/ca.crt`                        |
| services[].middlewares               | Enable built-in middlewares/interceptors for HTTP/gRPC service. <br>Details of available middlewares refer to below.        | `- recovery`                          |
| jobs[]                               | Jobs enabled in the app. <br>Jobs should be registered by App.AddJob(). <br>Only enabled jobs will be run.                  | `- txn_executor`                      |
| clients                              | Clients of dependent service.                                                                                               |                                       |
| clients.gprc                         | gRPC clients of dependent service.                                                                                          |                                       |
| clients.gprc.middlewares             | Enable built-in middlewares/interceptors for all gRPC clients. <br>Details of available middlewares refer to below.         | `- metrics`                           |
| clients.grpc.servers                 | Server of gRPC clients.                                                                                                     |                                       |
| clients.grpc.servers[].name          | Name of gRPC server to fetch the client interface. <br>This name will also be verified for connection TLS CA is configured. | `iam`                                 |
| clients.grpc.servers[].endpoint      | Endpoint of gRPC server.                                                                                                    | `127.0.0.1:9000`                      |
| clients.grpc.servers[].security      | gRPC client TLS configuration. <br>Optional, use insecure connection if not configured.                                     |                                       |
| clients.grpc.servers[].security.key  | TLS client key.                                                                                                             | `./keys/service.pem`                  |
| clients.grpc.servers[].security.cert | TLS client certificate chain.                                                                                               | `./keys/service.crt`                  |
| clients.grpc.servers[].security.ca   | TLS CA for verifying server certificates.                                                                                   | `./key/ca.crt`                        |
| databases                            | Databases used by app.                                                                                                      |                                       |
| databases[].name                     | Name of database to fetch the client interface.                                                                             | `iam`                                 |
| databases[].database                 | Database schema name.                                                                                                       | `iam_db`                              |
| databases[].user                     | Database user name.                                                                                                         | `test_user`                           |
| databases[].password                 | Database passwrod.                                                                                                          | `testpass`                            |
| databases[].masters                  | Database master endpoints for read-write query.                                                                             | `["127.0.0.1"]`                       |
| databases[].slaves                   | Database slave endpoints for readonly query.                                                                                | `["10.0.0.1:6606",  "10.0.0.2:6606"]` |
| caches                               | Caches used by app.                                                                                                         |                                       |
| caches[].name                        | Name of cache to fetch the client interface.                                                                                | `default`                             |
| caches[].type                        | Cache client type.  <br>Choices: `redis`                                                                                    | `redis`                               |
| caches[].address                     | Cache server address in `<host>:<port>` format.                                                                             | `127.0.0.1:6379`                      |
| caches[].username                    | Optional. Username for authentication.                                                                                      | `test_user`                           |
| caches[].password                    | Optional. Username for authentication.                                                                                      | `testpass`                            |
| caches[].db                          | Database to be selected after connecting to the server.                                                                     | `0`                                   |
| pulsars                              | Pulsar clients                                                                                                              |                                       |
| pulsars[].name                       | Name of pulsar server to fetch the client interface                                                                         | `iam`                                 |
| pulsars[].url                        | URL of pulsar server                                                                                                        | `pulsar://10.0.0.1:6650`              |
| pulsars[].token                      | Token for pulsar auth, optional                                                                                             | `eyJhbGciOiJU...`                     |

### Observable Service Modules

Below are built-in observable service modules:

| Name     | Endpoint    | Description                                             |
|----------|-------------|---------------------------------------------------------|
| pprof    | `/pprof`    | Golang standard profiling tool.                         |
| channelz | `/channelz` | gRPC Channelz UI.                                       |
| swagger  | `/swagger`  | Swagger docs UI.                                        |
| metrics  | `/metrics`  | Prometheus metrics endpoint for gRPC and HTTP services. |
| grpcui   | `/grpcui`   | gRPC service interactive UI.                            |

### Service Middlewares

Below are built-in middlewares for gRPC/HTTP client/service:

| Name               | Description                                                                                                                     | gRPC Service | HTTP Service | gRPC Client |
|--------------------|---------------------------------------------------------------------------------------------------------------------------------|--------------|--------------|-------------|
| recovery           | Auto recover from panic in handler and record details in log.                                                                   | Y            | Y            | N           |
| open_tracing       | (Not implemented yet) Fetch or generate trace id, put in context; generate and pass request ID for sub requests.                | Y            | Y            | Y           |
| metrics            | Record request metrics.                                                                                                         | Y            | Y            | Y           |
| context_logger     | Add request metadata into logger and put logger in context. <br>Context logger can be fetched by `log.FromContext` in handlers. | Y            | Y            | N           |
| log_request        | Record log for each request, includes metadata, error code, latency, etc.                                                       | Y            | Y            | N           |
| request_validation | Validate request data in gRPC client/service by protoc-gen-validate.                                                            | Y            | N            | Y           |
| cors               | HTTP CORS handling                                                                                                              | N            | Y            | N           |
| compress           | HTTP response compression                                                                                                       | N            | Y            | N           |
| access_control     | Request access control based on gRPC TLS certificate and Casbin configuration                                                   | Y            | N            | N           |

## Libraries

### errors

`fromego/errors` module provides useful utilities for error handling, and is compatible with errors module in standard library. Refer to [errors document](./docs/errors.md) for details.