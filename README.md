# Simple API Gateway

This is a simple API Gateway that routes requests to the appropriate microservice(s).

## Configuration

Configuration is done via a TOML file. Below is an example configuration file:

```toml
# Application Configuration
name = "App"            # The name of the application
host = ""               # The host of the application (if empty, it will default to the host of the machine)
port = 8080             # The port on which the application is running
cache = false           # Determines whether the application should cache responses (true or false)
log = true              # Specifies the log level of the application (true or false)

# Service Configuration
# Each service should start with "service"
# host:port/api/v1/simple/1 -> http://x.com/1
[service.api.v1.simple]
servers = ["http://x.com"]   # Defines the servers for the specified service

# If you want to use multiple servers, you can specify the method (default method is random)
[service.api.v1.random]
method = "random"            # Specifies the method for load balancing ("random" or "first")
servers = [
    "https://example.workers.dev",
    "https://example.run.app",
    "https://example.us-east-1.on.aws/"
]                            # Lists the servers to be used for the specified service
```

Note: The configuration file is divided into two parts: application configuration and service configuration. The application configuration contains settings that, if changed, require a restart of the application. The service configuration, on the other hand, supports hot-reloading, meaning you can modify it without restarting the application.