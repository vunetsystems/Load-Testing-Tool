````md
# PostgreSQL Client Package Documentation

## Overview

The `postgres` package provides a **simple, reusable abstraction** for connecting to a PostgreSQL database using Go’s standard `database/sql` package and the `lib/pq` PostgreSQL driver.

It is designed to:
- Centralize PostgreSQL connection logic
- Provide a clean client wrapper
- Support configurable and default connection parameters
- Ensure connection validation at startup

This package is suitable for **backend services, data pipelines, monitoring tools, and microservices** that require reliable PostgreSQL access.

---

## Package Declaration

```go
package postgres
````

This package encapsulates all PostgreSQL-related connection logic and should be imported wherever database access is required.

---

## Dependencies

```go
import (
    "database/sql"
    "fmt"

    _ "github.com/lib/pq"
)
```

### Dependency Breakdown

| Dependency          | Purpose                              |
| ------------------- | ------------------------------------ |
| `database/sql`      | Go standard database abstraction     |
| `github.com/lib/pq` | PostgreSQL driver implementation     |
| `fmt`               | String formatting and error wrapping |

> **Note:**
> The PostgreSQL driver is imported using a **blank identifier (`_`)** to register it with `database/sql` without directly referencing it.

---

## Configuration Structure

### `Config`

```go
type Config struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
}
```

#### Purpose

The `Config` struct holds **all parameters required to establish a PostgreSQL connection**.

#### Field Description

| Field      | Type     | Description                              |
| ---------- | -------- | ---------------------------------------- |
| `Host`     | `string` | PostgreSQL server hostname or IP address |
| `Port`     | `int`    | PostgreSQL server port (default: 5432)   |
| `User`     | `string` | Database username                        |
| `Password` | `string` | Database password                        |
| `DBName`   | `string` | Target database name                     |

---

## Client Structure

### `Client`

```go
type Client struct {
    DB *sql.DB
}
```

#### Purpose

`Client` acts as a **thin wrapper** around the `*sql.DB` connection pool, allowing:

* Centralized lifecycle management
* Future extensibility (e.g., helper methods, transactions, metrics)

---

## Client Initialization

### `NewClient`

```go
func NewClient(config Config) (*Client, error)
```

#### Description

Creates and initializes a new PostgreSQL client using the provided configuration.

#### Workflow

1. Builds a PostgreSQL connection string
2. Opens a database connection
3. Pings the database to validate connectivity
4. Returns a fully initialized client

#### Implementation

```go
connStr := fmt.Sprintf(
    "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
    config.Host,
    config.Port,
    config.User,
    config.Password,
    config.DBName,
)
```

> **Important:**
> `sslmode=disable` is explicitly set.
> This is suitable for:
>
> * Internal networks
> * Trusted environments
> * Kubernetes / service-mesh communication
>
> For production internet-facing databases, SSL **should be enabled**.

#### Error Handling

* Fails if the connection cannot be opened
* Fails if the database cannot be pinged
* Errors are wrapped using `%w` for better traceability

#### Return Values

| Value     | Description                         |
| --------- | ----------------------------------- |
| `*Client` | Initialized PostgreSQL client       |
| `error`   | Non-nil if connection or ping fails |

---

## Connection Validation

```go
if err := db.Ping(); err != nil {
    return nil, fmt.Errorf("failed to ping database: %w", err)
}
```

### Why `Ping()`?

* Ensures credentials are valid
* Ensures network connectivity
* Detects configuration issues early
* Prevents runtime failures later in the application

---

## Closing the Connection

### `Close`

```go
func (c *Client) Close() error
```

#### Description

Safely closes the database connection pool.

#### Behavior

* Checks if the DB instance is non-nil
* Closes all open connections
* Returns any close error

#### Usage Example

```go
client, err := postgres.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

---

## Default Configuration

### `GetDefaultConfig`

```go
func GetDefaultConfig() Config
```

#### Description

Returns a **predefined default PostgreSQL configuration** intended for pipeline or internal service usage.

#### Implementation

```go
return Config{
    Host:     "10.96.1.65",
    Port:     5432,
    User:     "Load_Testing_Tool_read_user",
    Password: "StrongPassword123",
    DBName:   "multicore",
}
```

#### Use Cases

* Internal tools
* Development or staging environments
* Standardized pipeline connections

> ⚠️ **Security Note**
> Hardcoded credentials should be avoided in production.
> Prefer:
>
> * Environment variables
> * Secrets managers
> * Kubernetes secrets
> * Vault / AWS Secrets Manager

---

## Example Usage

```go
package main

import (
    "log"
    "yourmodule/postgres"
)

func main() {
    cfg := postgres.GetDefaultConfig()

    client, err := postgres.NewClient(cfg)
    if err != nil {
        log.Fatalf("Failed to connect to DB: %v", err)
    }
    defer client.Close()

    // Use client.DB for queries
}
```

---

## Design Considerations

### Why use `database/sql`?

* Built-in connection pooling
* Driver-agnostic API
* Mature and battle-tested
* Integrates with observability tools

### Why a wrapper client?

* Clean separation of concerns
* Easier testing and mocking
* Future extensibility (metrics, retries, transactions)

---

## Potential Enhancements

* SSL/TLS configuration support
* Context-aware connection handling
* Query helper methods
* Connection pool tuning
* Health-check endpoint
* Environment-based config loading

---

## Summary

The `postgres` package provides:

* ✅ Clean PostgreSQL connection abstraction
* ✅ Centralized configuration management
* ✅ Early connection validation
* ✅ Safe lifecycle handling
* ✅ Extensible client design

It is well-suited for **production-grade Go services** with minimal overhead and clear structure.

---
