package cli

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ==================== Global Variables ====================

var _DB *sql.DB

// ==================== Database Connection ====================

// OpenDB establishes a connection to the database.
// Parameters:
//   - dBType: database driver name (e.g., "sqlite", "mysql", "postgres")
//   - dBName: data source name (connection string)
func OpenDB(dBType, dBName string) (err error) {

	// be careful, NO ':'
	_DB, err = sql.Open(dBType, dBName)
	if err != nil {
		return (err)
	}

	// Verify the connection is alive
	if err = _DB.Ping(); err != nil {
		return (err)

	}

	fmt.Printf("Database connected: %s\n", dBName)

	return nil
}

// CloseDB closes the global database connection.
// Should be called at program exit (deferred in main).
func CloseDB() {
	if _DB != nil {
		fmt.Printf("Closing database connection...\n")
		if err := _DB.Close(); err != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", err)
		}
	}

}

// ==================== CSV Import ====================

// LoadCsv imports a CSV file into a database table.
// It uses a transaction for atomicity (all rows or none).
// Security: Uses parameterized queries to prevent SQL injection.

func LoadCsv(tableName, csvName string) error {

	fmt.Printf("Importing CSV: %s -> table: %s\n", csvName, tableName)

	// Open CSV file
	file, err := os.Open(csvName)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true // Handle spaces in CSV

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build parameterized INSERT statement
	// INSERT INTO table (col1, col2, ...) VALUES (?, ?, ...)
	placeholders := strings.Repeat("?,", len(headers))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(headers, ","),
		placeholders,
	)

	// Begin transaction
	tx, err := _DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)

	}
	// Ensure proper rollback on error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Prepare statement for reuse (improves performance and security)

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	rowCount := 0
	for {

		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break // End of file, normal exit
			}
			return fmt.Errorf("CSV read error at row %d: %w", rowCount+1, err)
		}

		// Convert record to []any for parameterized query
		args := make([]any, len(record))
		for i, val := range record {
			args[i] = val
		}

		// Execute parameterized INSERT (safe from SQL injection)
		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("insert error at row %d: %w", rowCount+1, err)
		}
		rowCount++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported %d rows into table '%s'\n", rowCount, tableName)
	return nil
}

// ==================== SQL Execution ====================

// ExecSqlFile reads and executes SQL statements from a file.
// Note: For multiple statements, the file should contain valid SQL syntax.

func ExecSqlFile(sqlFileName string) error {

	data, err := os.ReadFile(sqlFileName)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	sql := string(data)
	fmt.Printf("Executing SQL from file: %s\n", sqlFileName)

	_, err = _DB.Exec(sql)
	if err != nil {
		return fmt.Errorf("SQL execution failed: %w", err)
	}

	fmt.Println("SQL executed successfully")
	return nil

}

// ExecSqlString executes a single SQL statement.
// Use with caution - accepts raw SQL (potential injection risk).
func ExecSqlString(sql string) error {

	if _DB == nil {
		return errors.New("no database connection. Use 'sqlite <db>' first")
	}

	fmt.Printf("Executing: %s\n", sql)

	_, err := _DB.Exec(sql)
	if err != nil {
		return fmt.Errorf("SQL execution failed: %w", err)
	}

	fmt.Println("SQL executed successfully")
	return nil

}

// ==================== Query and Display ====================
// SelectPrint executes a SELECT query and prints results in a formatted table.
func SelectPrint(sql string) error {

	if _DB == nil {
		return errors.New("no database connection. Use 'sqlite <db>' first")
	}

	names, results, err := queryMany(sql)

	if err != nil {
		return (err)
	}

	for _, name := range names {
		fmt.Printf("|%-20v", name)
	}
	fmt.Println("|")
	fmt.Println(strings.Repeat("-", 21*len(names)+1))

	for _, row := range results {
		for _, name := range names {
			fmt.Printf("|%-20v", row[name])
		}
		fmt.Println("|")
	}

	return nil
}

func queryMany(sql string) ([]string, []map[string]any, error) {

	fmt.Println(sql)

	rows, err := _DB.Query(sql)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns() // Remember to check err afterwards
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, len(cols))
	results := make([]map[string]any, 0)

	vals := make([]any, len(cols))
	scanArgs := make([]any, len(cols))

	for i, col := range cols {
		scanArgs[i] = &vals[i]
		names[i] = col
	}
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]any)
		for i, val := range vals {
			var value any
			if b, ok := val.([]byte); ok {
				value = string(b)
			} else {
				value = val
			}
			row[names[i]] = value
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return names, results, err
	}

	return names, results, err

}
