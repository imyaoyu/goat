package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"gopkg.in/validator.v2"
)

const OK = "OK"

// apiError represents a business logic error with HTTP status code and cause
type apiError struct {
	ErrCode  int
	ErrMsg   string
	ErrCause error
}

func (e *apiError) Error() string {
	if e.ErrCause != nil {
		return fmt.Sprintf("%d:%s|%v", e.ErrCode, e.ErrMsg, e.ErrCause)
	}
	return fmt.Sprintf("%d:%s", e.ErrCode, e.ErrMsg)
}

// newError creates a new apiError with the given code, message, and optional cause
func newError(code int, msg string, err error) *apiError {
	return &apiError{
		ErrCode:  code,
		ErrMsg:   msg,
		ErrCause: wrapError(err, 4),
	}
}

// wrapError adds file, line, and function context to an error for easier debugging
// skip parameter allows callers to skip additional wrapper frames
func wrapError(err error, skip ...int) error {
	if err == nil {
		return nil
	}
	skipLevel := 1 // Default: skip this function
	if len(skip) > 0 {
		skipLevel = skip[0]
	}

	// pc: program counter
	// file:
	// line:
	// ok:
	pc, file, line, ok := runtime.Caller(skipLevel)
	if !ok {
		return fmt.Errorf("unable to get caller info: %w", err)
	}

	// Get function name
	funcName := runtime.FuncForPC(pc).Name()

	//Simplify function name (keep last part)
	if idx := strings.LastIndex(funcName, "."); idx != -1 {
		funcName = funcName[idx+1:]
	}

	// Simplify file name (keep only filename)
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		file = file[idx+1:]
	}

	return fmt.Errorf("%s [%s]:%d <%v>", funcName, file, line, err)

}

// apiResult is the standard response format for all API endpoints
type apiResult struct {
	Code   string `json:"code"`
	Msg    string `json:"msg"`
	Result any    `json:"result,omitempty"`
}

// newApiCtx creates a new ApiCtx from an HTTP request/response pair
func newApiCtx(w http.ResponseWriter, r *http.Request) *ApiCtx {
	c := new(ApiCtx)
	c.BeginTime = time.Now()
	c.Request = r
	c.Response = w
	c.UserId = r.Header.Get(X_USER_ID)

	// Generate or propagate trace ID
	if traceID := r.Header.Get(X_TRACE_ID); traceID == "" {
		c.TraceId = UUID()
	} else {
		c.TraceId = traceID
	}

	return c
}

// ApiCtx is the context object passed through all API handlers
type ApiCtx struct {
	ApiCode   string              // API code for routing
	UserId    string              // User identifier from authentication
	I, O      any                 // Input and output objects
	Request   *http.Request       // Original HTTP request
	Response  http.ResponseWriter // HTTP response writer
	TraceId   string              // Trace ID for distributed tracing
	BeginTime time.Time           // Request start time
}

// Init parses and validates the request body into the input object
func (c *ApiCtx) Init(i any, o any) {
	c.I, c.O = i, o

	// Decode JSON request body
	if err := json.NewDecoder(c.Request.Body).Decode(c.I); err != nil {
		c.Panic(http.StatusBadRequest, "DecodeJsonError", err)
	}

	c.Log("InitInput", "c.I", c.I)

	// Validate input using validator tags
	if err := validator.Validate(c.I); err != nil {
		c.Panic(http.StatusBadRequest, "ValidateJsonError", err)
	}
}

// Log outputs structured log entries with context information
func (c *ApiCtx) Log(desc string, v ...any) {
	slog.Info(desc,
		slog.Group("ApiCtx",
			"TraceId", c.TraceId,
			"ApiCode", c.ApiCode,
			"UserId", c.UserId,
		),
		slog.Group("ApiFunc", v...),
	)
}

// Log outputs structured log entries with context information
func (c *ApiCtx) errLog(v ...any) {
	slog.Error("ErrorOrPanic",
		slog.Group("ApiCtx",
			"TraceId", c.TraceId,
			"ApiCode", c.ApiCode,
			"UserId", c.UserId,
		),
		slog.Group("ApiFunc", v...),
	)
}

// Panic throws an API error that will be caught by the recovery middleware
func (c *ApiCtx) Panic(code int, msg string, err error) {
	panic(newError(code, msg, err))
}

// returnResult sends a successful JSON response to the client
func (c *ApiCtx) returnResult() {
	c.writeJSON(&apiResult{OK, Now(), c.O})
	c.Log("returnResult", "c.O", c.O, "DurationMs", time.Since(c.BeginTime).Milliseconds())
}

// returnErrorResult sends an error JSON response to the client
func (c *ApiCtx) returnErrorResult(err *apiError) {

	code := fmt.Sprintf("%s.%d", c.ApiCode, err.ErrCode)
	msg := _CodeMsg[err.ErrCode]
	if msg == "" {
		msg = err.ErrMsg
	}

	result := &apiResult{Code: code, Msg: msg}
	c.writeJSON(result)
	c.errLog("err", err, "result", result, "DurationMs", time.Since(c.BeginTime).Milliseconds())

}

// writeJSON writes json string to web client
func (c *ApiCtx) writeJSON(res *apiResult) {
	c.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(c.Response).Encode(res); err != nil {
		c.Panic(http.StatusInternalServerError, "InternalServerError", err)
	}
}

// returnNotKnown sends a 500 error response for unexpected panics
func (c *ApiCtx) returnNotKnown(rc any) {
	c.returnErrorResult(newError(500, "NotKnown", fmt.Errorf("%v", rc)))
}

// ==================== Database Accessors ====================

// Insert creates a new record in the database
func (c *ApiCtx) Insert(tabel any) {
	ctx := c.Request.Context()
	_, err := DB().Context(ctx).Insert(tabel)
	if err != nil {
		c.Panic(500, "DBError", err)
	}
}

// Select retrieves a single record matching the where condition
func (c *ApiCtx) Select(table any, where string, values ...any) bool {
	ctx := c.Request.Context()
	has, err := DB().Context(ctx).Where(where, values...).Get(table)
	if err != nil {
		c.Panic(500, "DBError", err)
	}
	return has
}

// SelectM executes a raw SQL query and returns results as a slice of maps
func (c *ApiCtx) SelectM(sql string) []map[string][]byte {
	ctx := c.Request.Context()
	results, err := DB().Context(ctx).Query(sql)
	if err != nil {
		c.Panic(500, "DBError", err)
	}
	return results

}

// SelectS performs a paginated query and populates the results slice
func (c *ApiCtx) SelectS(results *[]any, limit, start int, orderby string, where string, values ...any) {
	ctx := c.Request.Context()
	err := DB().Context(ctx).
		Where(where, values...).
		OrderBy(orderby).
		Limit(limit, start).
		Find(results)
	if err != nil {
		c.Panic(500, "DBError", err)
	}
	return
}

// Update updates a record by primary key with the given map of changes
func (c *ApiCtx) Update(table any, id int64, set map[string]any) {
	ctx := c.Request.Context()
	affected, err := DB().Context(ctx).Table(table).ID(id).Update(set)
	if err != nil {
		c.Panic(500, "DBError", err)
	}
	if affected != 1 {
		c.Panic(500, "DBError", fmt.Errorf("update %v where id=%d set[%v] affected != 1", table, id, set))
	}

}

// ExecSQL executes a raw SQL statement (UPDATE/DELETE)
func (c *ApiCtx) ExecSQL(sqlOrVals ...any) {
	result, err := DB().Exec(sqlOrVals...)
	if err != nil {
		c.Panic(500, "DBError", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		c.Panic(500, "DBError", err)
	}
	if affected != 1 {
		c.Panic(500, "DBError", fmt.Errorf("Update %v affected != 1", sqlOrVals))

	}

}
