# GOAT - A Pure Go CLI Tool with Embedded SQLite and a web application framework

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)]()

**GOAT** (Go@) is a pure Go CLI tool that combines an interactive shell, embedded SQLite database, and a web application framework. It runs on Windows, Linux, and macOS with zero external dependencies.

## ✨ Features

### 🖥️ Interactive Shell
- **Built-in commands**: `ls`, `mkdir`, `cd`, `cat`, `pwd`, `clear`, `exit` (or `Ctrl+D`)
- **Smart command repetition**: Pressing `Enter` on an empty line repeats the last command
- **Cross-platform**: Works on Windows, Linux, and macOS

### 🗄️ Database Management
- **Embedded SQLite**: No separate database server required
- **CSV import**: Load CSV files directly into database tables
- **SQL execution**: Run SQL from files or interactive commands
- **Formatted query output**: Beautiful table formatting for `SELECT` queries

### 🌐 Web Framework
- **API gateway**: Built-in routing with middleware support
- **Request tracing**: Automatic trace ID propagation
- **Structured logging**: JSON log output with slog
- **Graceful shutdown**: Clean resource cleanup on exit

### 🛠️ Development Tools
- **Cross-compilation**: Build binaries for Windows, Linux, and macOS
- **Project scaffolding**: `init` command creates new Go projects
- **Source code packaging**: Built-in ZIP compression

## 📦 Installation

### Build from Source
```bash
git clone https://github.com/imyaoyu/goat.git
cd goat
go build -o goat
```

## 🚀 Quick Start

### Start the Interactive Shell
```bash
./goat -b
```

### Start as a Web Server
```bash
./goat -p 8080 -w ./www
```

### Database Operations
```bash
# Open a SQLite database
go@[~]> sqlite myapp.db

# Execute SQL from file
go@[~]> sqlf schema.sql

# Import CSV data
go@[~]> load users.csv

# Run queries
go@[~]> select * from users
```

### Create a New Project
```bash
go@[~]> init myapi
```

### Cross-Platform Build
```bash
go@[~]> build linux   # Build for Linux
go@[~]> build windows # Build for Windows
go@[~]> build darwin  # Build for macOS
```

## 📋 Command Reference

### File System Commands

| Command | Description | Example |
|:---|:---|:---|
| `ls` | List directory contents | `ls` |
| `cd <dir>` | Change directory | `cd /home/user` |
| `pwd` | Print working directory | `pwd` |
| `cat <file>` | Display file content | `cat README.md` |
| `mkdir <dir>` | Create directory | `mkdir data` |
| `clear` | Clear screen | `clear` |
| `exit` (or `Ctrl+D`) | Exit the shell | `exit` |

### Database Commands

| Command | Description | Example |
|:---|:---|:---|
| `sqlite [db]` | Open SQLite database | `sqlite test.db` |
| `sqlf <file>` | Execute SQL file | `sqlf init.sql` |
| `load <csv>` | Import CSV file | `load data.csv` |
| `select <sql>` | Execute SELECT query | `select * from users` |
| `update <sql>` | Execute UPDATE | `update users set name='Alice' where id=1` |
| `delete <sql>` | Execute DELETE | `delete from users where id=1` |

### Development Commands

| Command | Description | Example |
|:---|:---|:---|
| `init <name>` | Create new Go project | `init myapi` |
| `build <os>` | Cross-compile binary | `build linux` |
| `zip` | Package source code | `zip` |
| `help` | Show help | `help` |

## 🏗️ Architecture

### Project Structure
```
goat/
├── app/           # Web framework core
│   ├── app.go     # Server initialization
│   ├── ctx.go     # Request context
├── cli/           # Interactive shell
│   ├── cli.go     # Command line interface
│   ├── cmd.go     # Command handlers
│   └── dba.go     # Database operations
├── api/           # Example web handlers
│   ├── 1.go       # API 1 
│   ├── db.go      # Database table 
├── www/           # Static file (Optional) 
└── main.go        # Application entry
```

### Key Components

#### ApiCtx - Request Context
```go
type ApiCtx struct {
    ApiCode  string          		// API routing code
    UserId   string          		// Authenticated user ID
    TraceId  string          		// Distributed tracing ID
    Request  *http.Request   		// HTTP request
    Response http.ResponseWriter //HTTP response
    I,O      any             		// Input & Ouput struct
    ...
}
```

#### Middleware Support
```go
app.AddFunc(LogIp)// Log request
)
```

## 🔧 Configuration

### Command Line Flags
```bash
./goat -p 8080 -c config.json -w ./www -d

Flags:
  -p int        HTTP listen port (default: 80)
  -c string     JSON configuration file
  -db string    Database driver name (default: "sqlite")
  -dsn string   Database source name (default: "goat.db")
  -w string     Web static files directory
  -b            Start as interactive shell
  -d            Start as daemon process
```

### Configuration File (JSON)
```json
{
    "_DB_NAME": "mysql",
    "_DB_DSN": "user:pass@tcp(localhost:3306)/dbname",
    "SQL": "init sql string"
}
```

# ApiCtx Public Methods Interface Documentation

## Overview

`ApiCtx` is the core context object passed through all API handlers. It provides methods for request parsing, validation, logging, error handling, and database operations.

---

## Initialization & Request Handling

### `Init`

Parses and validates the JSON request body into the input object.

```go
func (c *ApiCtx) Init(i any, o any)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `i` | `any` | Yes | Pointer to input struct for request parameters |
| `o` | `any` | Yes | Pointer to output struct for response data |

**Behavior**:
- Reads JSON from `c.Request.Body` and unmarshals into `i`
- Validates `i` using struct tags (requires `validator` package)
- Stores `i` and `o` in `c.I` and `c.O` for later use
- On error: calls `c.Panic()` with appropriate HTTP status code

**Example**:
```go
type LoginRequest struct {
    Username string `json:"username" validate:"required"`
    Password string `json:"password" validate:"required"`
}
type LoginResponse struct {
    Token string `json:"token"`
}

i :=new(LoginRequest)
o :=new(LoginResponse)
c.Init(i,o)
```

**Error Cases**:
- Invalid JSON → `400 BadRequest` with `DecodeJsonError`
- Validation failed → `400 BadRequest` with `ValidateJsonError`

---

### `Log`

Outputs structured log information with automatic context enrichment.

```go
func (c *ApiCtx) Log(desc string, v ...any)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `desc` | `string` | Yes | Log message description |
| `v` | `...any` | No | Alternating key-value pairs for structured data |

**Behavior**:
- Automatically includes `TraceId`, `ApiCode`, `UserId` in log context
- Outputs in JSON format (via `slog`)
- Log level: INFO

**Example**:
```go
c.Log("User logged in", "user_id", 123, "ip", "192.168.1.1")

// Output:
// {"time":"...","level":"INFO","msg":"User logged in","ApiCtx":{"TraceId":"abc","ApiCode":"login","UserId":"123"},"ApiFunc":{"user_id":123,"ip":"192.168.1.1"}}
```

---

### `Panic`

Throws an API error that will be caught by the recovery middleware.

```go
func (c *ApiCtx) Panic(code int, msg string, err error)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `code` | `int` | Yes | HTTP status code (e.g., 400, 401, 403, 404, 500) |
| `msg` | `string` | Yes | Error message identifier |
| `err` | `error` | No | Original error for debugging (can be nil) |

**Behavior**:
- Wraps error with stack trace information
- Panics immediately - execution stops
- Recovered by middleware which returns formatted JSON error response

**Example**:
```go
if user == nil {
    c.Panic(404, "UserNotFound", nil)
}

if err != nil {
    c.Panic(500, "DatabaseError", err)
}
```

---

## Database Operations

### `Insert`

Creates a new record in the database.

```go
func (c *ApiCtx) Insert(table any)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | `any` | Yes | Pointer to struct representing the database record |

**Behavior**:
- Uses XORM's `Insert` method
- Context is automatically propagated from `c.Request.Context()`
- On success: record is inserted, ID fields are auto-populated if applicable
- On failure: calls `c.Panic(500, "DBError", err)`

**Example**:
```go
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
    Age:   30,
}
c.Insert(user)
fmt.Println("Inserted ID:", user.Id)
```

---

### `Select`

Retrieves a single record matching the WHERE condition.

```go
func (c *ApiCtx) Select(table any, where string, values ...any) bool
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | `any` | Yes | Pointer to struct to receive query result |
| `where` | `string` | Yes | WHERE clause with placeholders (e.g., `"id = ?"`) |
| `values` | `...any` | No | Values to substitute into WHERE clause |

**Returns**:
- `bool` - `true` if record found and populated, `false` otherwise

**Behavior**:
- Uses XORM's `Where().Get()` method
- Context is automatically propagated
- On database error: calls `c.Panic(500, "DBError", err)`

**Example**:
```go
var user User
if c.Select(&user, "id = ?", 123) {
    fmt.Println("User found:", user.Name)
} else {
    fmt.Println("User not found")
}
```

---

### `SelectM`

Executes a raw SQL query and returns results as a slice of maps.

```go
func (c *ApiCtx) SelectM(sql string) []map[string][]byte
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sql` | `string` | Yes | Raw SQL SELECT statement |

**Returns**:
- `[]map[string][]byte` - Array of rows, each row is a map with column names as keys and byte arrays as values

**Behavior**:
- Useful for complex queries or dynamic column selection
- Context is automatically propagated
- On error: calls `c.Panic(500, "DBError", err)`

**Example**:
```go
sql := `SELECT id, name, created_at FROM users WHERE age > ?`
results := c.SelectM(sql)

for _, row := range results {
    id := string(row["id"])
    name := string(row["name"])
    createdAt := string(row["created_at"])
    fmt.Printf("ID: %s, Name: %s, Created: %s\n", id, name, createdAt)
}
```

---

### `SelectS`

Performs a paginated query and populates the results slice.

```go
func (c *ApiCtx) SelectS(results *[]any, limit, start int, orderby string, where string, values ...any)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `results` | `*[]any` | Yes | Pointer to slice to receive results |
| `limit` | `int` | Yes | Maximum number of records to return |
| `start` | `int` | Yes | Offset for pagination (0 = first record) |
| `orderby` | `string` | Yes | ORDER BY clause (e.g., `"id DESC"`) |
| `where` | `string` | Yes | WHERE clause with placeholders |
| `values` | `...any` | No | Values to substitute into WHERE clause |

**Behavior**:
- Uses XORM's `Where().OrderBy().Limit().Find()` chain
- Context is automatically propagated
- On error: calls `c.Panic(500, "DBError", err)`

**Example**:
```go
var users []User
c.SelectS(&users, 10, 0, "id DESC", "status = ?", "active")

for _, user := range users {
    fmt.Println(user.Name)
}
```

---

### `Update`

Updates a record by primary key ID with the given map of changes.

```go
func (c *ApiCtx) Update(table any, id int64, set map[string]any)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | `any` | Yes | Struct pointer (used to determine table name) |
| `id` | `int64` | Yes | Primary key value |
| `set` | `map[string]any` | Yes | Column-value pairs to update |

**Behavior**:
- Uses XORM's `Table().ID().Update()` method
- Context is automatically propagated
- Verifies that exactly 1 row was affected
- On error or affected != 1: calls `c.Panic(500, "DBError", err)`

**Example**:
```go
c.Update(&User{}, 123, map[string]any{
    "name": "Updated Name",
    "age":  31,
})
```

---

### `ExecSQL`

Executes a raw SQL statement (UPDATE, DELETE, or INSERT).

```go
func (c *ApiCtx) ExecSQL(sqlOrVals ...any)
```

**Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sqlOrVals` | `...any` | Yes | SQL string followed by optional parameters |

**Behavior**:
- Uses XORM's `Exec()` method
- Verifies that exactly 1 row was affected
- On error or affected != 1: calls `c.Panic(500, "DBError", err)`

**Example**:
```go
// Without parameters
c.ExecSQL("DELETE FROM sessions WHERE expired = 1")

// With parameters
c.ExecSQL("UPDATE users SET status = ? WHERE id = ?", "inactive", 123)
```

---

## Error Handling Flow

```
User Request → Handler → c.Init() → Business Logic → c.Panic() on error
                    ↓
            Recovery Middleware
                    ↓
        ┌───────────┴───────────┐
        ↓                       ↓
   Success Path            Error Path
   c.returnResult()        c.returnErrorResult()
        ↓                       ↓
   JSON Response           JSON Error Response
   (200 OK)                (4xx/5xx)
```

---

## Usage Pattern

```go
func MyAPIHandler(c *app.ApiCtx) {
    // 1. Parse and validate request
    i,o := new(MyRequest),new(MyResponse)
    c.Init(i, o)

    // 2. Validate business logic
    if i.Amount <= 0 {
        c.Panic(400, "InvalidAmount", nil)
    }

    // 3. Database operations
    var user User
    if !c.Select(&user, "id = ?", i.UserID) {
        c.Panic(404, "UserNotFound", nil)
    }

    // 4. Update data
    c.Update(&User{}, user.Id, map[string]any{
        "balance": user.Balance - req.Amount,
    })

    // 5. Set response (automatically wrapped in apiResult)
    o.TransactionID = generateID()

    // 6. Log success
    c.Log("Transaction completed", "amount", req.Amount, "user_id", user.Id)
}
```
## 📝 Example: Building a Simple API

### 1. Create a new project
```bash
goat -b
go@[~]> init myapi
go@[~]> cd myapi
```

### 2. Define API handler (main.go)
```go
package main

import (
    "github.com/imyaoyu/goat/app"
)

func main() {
    // Define API endpoint with code "demo"
    //Define api
	app.Api("demo", func(c *app.ApiCtx){
		
		type Ping struct{
			Msg string
		}
		type Pong struct{
			Msg string
		}
	    
		i,o := new(Ping),new(Pong)
		c.Init(i,o)
		
		o.Msg = i.Msg
		//End
		
		
	})
    // Start server
    app.Run()
}
```

### 3. Run the server
```bash
go run main.go -p 8080
```

### 4. Test the API
```bash
curl -X POST http://localhost:8080/demo \
  -H "Content-Type: application/json" \
  -d '{"Msg":"Hello, World"}'

# Response: {"code":"OK","msg":"2026-01-15 10:30:45","result":{"Msg":"Hello, World"}}
```

### See more [main.go] as a demo

## 🔒 Security Features

- **SQL Injection Protection**: Parameterized queries in CSV import
- **XSS Prevention**: HttpOnly cookies for session management
- **CSRF Protection**: SameSite cookie attribute
- **Request Tracing**: Automatic trace ID propagation
- **Structured Logging**: JSON format for log aggregation

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

### Development Setup
```bash
git clone https://github.com/imyaoyu/goat.git
cd goat
go mod download
go build
./goat -b
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [SQLite](https://www.sqlite.org/) - Embedded database engine
- [XORM](https://gitea.com/xorm/xorm) - ORM library
- [Google UUID](https://github.com/google/uuid) - UUID generation

## 📧 Contact

- Author: YU.YAO.IM
- Email: yu@yao.im
- Project Home: [https://github.com/imyaou/goat](https://github.com/imyaoyu/goat)

---

**GOAT** - Your Go Application Toolkit. Build CLI tools, web servers, and data applications with pure Go.

