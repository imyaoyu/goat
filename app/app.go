package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"xorm.io/xorm"
	"xorm.io/xorm/core"

	"github.com/imyaoyu/goat/cli"
)

// ==================== Constants ====================

const GOAT = "GreatOfAllTime"

// Header keys for internal use
const (
	X_USER_ID  = "X-User-Id"
	X_TRACE_ID = "X-Trace-Id"
)

// ==================== Global Variables =============
var (
	// Server configuration
	_PORT int // HTTP server port

	// Database configuration
	_DB_NAME              string       // Database driver name (e.g., "sqlite", "mysql")
	_DB_DSN               string       // Data source name (connection string)
	_DB                   *sql.DB      // Standard database handle
	_DB_XORM              *xorm.Engine // XORM ORM engine
	_DB_MAX_OPEN_CONNS    = 20         // Maximum open connections
	_DB_MAX_IDLE_CONNS    = 10         // Maximum idle connections
	_DB_CONN_MAX_LIFETIME = 3600       // Connection lifetime

	// Web server configuration
	_WebDir    string                              // Static files directory
	_ServerMux *http.ServeMux = http.NewServeMux() // HTTP router

	// Application configuration
	_Router    = map[string]ApiFunc{} // API routing table: code -> handler
	_FuncChain = []ApiFunc{}          // Middleware list
	_Env       = map[string]string{}  // Environment variables loaded from config file
	_CodeMsg   = map[int]string{}
)

// ApiFunc defines the signature for API handlers
type ApiFunc func(c *ApiCtx)

// ==================== Initialization ====================

// init parses command-line flags and initializes the application.
func init() {

	// Print application info
	fmt.Printf("go@app %s %s %s-%s \n", os.Args[0], runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// Command-line flags
	var (
		serverPort   int    // HTTP port
		asBash       bool   // Run as interactive shell
		configFile   string // JSON config file path
		dbName       string // Database driver name
		dbSourceName string // Database connection string
		webDir       string // Static files directory
		asServer     bool   // Run as daemon process
	)

	flag.IntVar(&serverPort, "p", 80, "listen Port")
	flag.StringVar(&configFile, "c", "", "read file xxx.json as Config")

	flag.StringVar(&dbName, "db", "sqlite", "DataBase dialect name")
	flag.StringVar(&dbSourceName, "dsn", os.Args[0]+".db", "Database Source Name ")

	flag.BoolVar(&asBash, "b", false, "start as a command line like Bash")
	flag.BoolVar(&asServer, "d", false, "start as a Daemon process")

	flag.StringVar(&webDir, "w", "", "add a dir as the WWW server")

	flag.Parse()

	// Show help if no arguments provided
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	// Configure structured logging (JSON format)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		//AddSource: true,
	})))

	// Start interactive shell mode
	if asBash {
		cli.StartCmdLine()
		os.Exit(0)
	}

	// Start as daemon process (background)
	if asServer {
		serverName := os.Args[0]
		cmd := fmt.Sprintf("nohup %s -p=%d -c=%s -w=%s >> %s.log 2>&1 & echo $! > %s.pid",
			serverName, serverPort, configFile, webDir, serverName, serverName)
		cli.ExecShellCmd(cmd)
		os.Exit(0)
	}

	// Load configuration from JSON file if provided
	if configFile != "" {
		// update config
		loadConfig(configFile)

	}

	// Apply configuration
	_WebDir = webDir
	_PORT = serverPort
	_DB_NAME = dbName
	_DB_DSN = fmt.Sprintf("%s?cache=shared&_txlock=immediate", dbSourceName)

	// Override with values from config file
	if v, ok := _Env["_DB_NAME"]; ok {
		_DB_NAME = v
	}
	if v, ok := _Env["_DB_DSN"]; ok {
		_DB_DSN = v
	}

	// Initialize database connection
	openDB()

}

// ==================== Database Functions ====================

// openDB initializes the database connection and XORM engine.
func openDB() {

	var err error

	// Open standard database connection
	_DB, err = sql.Open(_DB_NAME, _DB_DSN)
	if err != nil {
		log.Fatalf("[%d] sql.Open err:%+v", os.Getpid(), err)
	}
	// Configure connection pool
	_DB.SetMaxOpenConns(20)           // Maximum open connections
	_DB.SetMaxIdleConns(10)           // Maximum idle connections
	_DB.SetConnMaxLifetime(time.Hour) // Connection lifetime

	// Create XORM engine from existing DB connection
	_DB_XORM, err = xorm.NewEngineWithDB(_DB_NAME, _DB_DSN, core.FromDB(_DB))

	if err != nil {
		log.Fatalf("[%d] ORM Init err:%+v", os.Getpid(), err)
	}
	// Enable WAL mode for SQLite (better concurrency)
	_, err = _DB_XORM.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatalf("[%d] ORM journal_mode=WAL set err:%+v", os.Getpid(), err)
	}

	// Enable SQL logging for debugging
	_DB_XORM.ShowSQL(true)

	log.Printf("[%d] > open DB OK %s:%s", os.Getpid(), _DB_NAME, _DB_DSN)

}

// closeDB gracefully closes the database connection.
func closeDB() {
	if _DB_XORM != nil {
		log.Printf("[%d] > closing DB", os.Getpid())
		if err := _DB_XORM.Close(); err != nil {
			log.Printf("[%d] close DB Faild: %v", os.Getpid(), err)
		} else {
			log.Printf("[%d] > close DB OK", os.Getpid())
		}
	}
}

// ==================== Configuration Functions ====================

// loadConfig reads and parses a JSON configuration file.
func loadConfig(configFile string) {
	// 一次性读取：不存在会返回错误
	content, err := os.ReadFile(configFile)
	if err != nil {
		// 判断文件是否真的“不存在”
		if errors.Is(err, os.ErrNotExist) {
			log.Fatalf("config file not exist: %+v", err)
		} else {
			log.Fatalf("reading config file failed: %+v", err)
		}
	}
	if err := json.Unmarshal(content, &_Env); err != nil {
		log.Fatalf("parsing config file failed: %+v", err)
	}
}

// Env returns an environment variable value from the config file.
func Env(key string) string {
	return _Env[key]
}

// DB returns the XORM engine instance.
func DB() *xorm.Engine {
	return _DB_XORM
}

// ==================== Routing ====================

func CodeMsg(code int, msg string) {
	_CodeMsg[code] = msg

}

// Add registers an API handler for the given code.
func Api(code string, api ApiFunc) {
	_Router[code] = api
}

// Func registers an middlware
func Func(f ApiFunc) {
	_FuncChain = append(_FuncChain, f)
}

// Handle register a http handle for the given func(w,r)
func Handle(path string, f func(w http.ResponseWriter, r *http.Request)) {
	_ServerMux.Handle(path, http.HandlerFunc(f))
}

// ==================== Server Lifecycle ====================

// Run starts the HTTP server and waits for shutdown signal.
func Run() {

	server := initServer()

	log.Printf("[%d] > http://*%s", os.Getpid(), server.Addr)

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[%d] > Web Start Failed : %+v \n", os.Getpid(), err)
		}
	}()

	// Wait for shutdown signal
	waitShutDown(server)

	// Close database connection
	closeDB()
}

// initServer creates and configures the HTTP server.
func initServer() *http.Server {

	// Serve static files from web directory if specified
	// Important! There must be a index.html in the web dierectory
	if _WebDir != "" {
		_ServerMux.Handle("/", http.FileServer(http.Dir(_WebDir)))
	}

	// Add the last ApiFunc
	_FuncChain = append(_FuncChain, func(c *ApiCtx) {

		if apiFunc, ok := _Router[c.ApiCode]; ok {
			apiFunc(c)
		} else {
			c.Panic(404, "NotFound", fmt.Errorf("ApiCode:%s", c.ApiCode))
		}
	})

	// Register the main API gateway route
	path := fmt.Sprintf("/{%s}", GOAT)
	_ServerMux.Handle(path, wrapRecover(_FuncChain...))

	addr := fmt.Sprintf(":%d", _PORT)

	server := &http.Server{
		Addr:    addr,
		Handler: _ServerMux,
		// Timeout configurations for security and reliability
		ReadHeaderTimeout: 5 * time.Second,  // Time to read request headers
		ReadTimeout:       31 * time.Second, // Maximum duration for reading entire request
		WriteTimeout:      31 * time.Second, // Maximum duration for writing response
		IdleTimeout:       31 * time.Second, // Maximum idle time for keep-alive connections
		MaxHeaderBytes:    1 << 20,          // 1MB limit for request headers (DDoS protection)
	}
	return server

}

// waitShutDown gracefully shuts down the server on SIGINT or SIGTERM.
func waitShutDown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until signal received

	// Create timeout context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[%d] > shutting down", os.Getpid())
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[%d] > shutting down err [%v]", os.Getpid(), err)
	} else {
		log.Printf("[%d] > http server stopped", os.Getpid())
	}
}

// ==================== Recovery Middleware ====================

// wrapRecover creates a middleware that recovers from panics and returns appropriate responses.
func wrapRecover(nextFuncs ...ApiFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// new ctx
		c := newApiCtx(w, r)
		// recovery
		defer func() {
			if rc := recover(); rc != nil {

				switch err := rc.(type) {
				case *apiError:
					c.returnErrorResult(err)
				default:
					c.returnNotKnown(rc)
				}
			} else {
				c.returnResult()
			}
		}()

		// Execute all middleware/handlers in order
		for _, next := range nextFuncs {
			next(c)
		}

	})
}
