package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ==================== Constants ====================

// HelpInfo displays the CLI usage information
const HelpInfo = `
	GOAT (go@) - A pure Go CLI with embedded SQLite.

	Runs on Windows, Linux, and macOS.

	Built-in commands: ls, mkdir, cd, cat, pwd, clear, exit (or Ctrl+D).

	Pressing Enter on an empty line repeats the last command.
    
  	--- YU.YAO.IM @ 2026.6.1 ---
  	eg.
	> ./goat
	go@[~]> help
	go@[~]> sqlite [xx.db]
	go@[~]> sqlf db.sql
	go@[~]> load data.csv
	go@[~]> select * from test
	go@[~]> update test set where
	go@[~]> delete from test where
	go@[~]> clear
	go@[~]> build [linux]		
	go@[~]> zip
	go@[~]> init gocb1	
`

// Built-in filesystem commands
const (
	CD    = "cd"    // Change directory
	CAT   = "cat"   // Display file content
	PWD   = "pwd"   // Print working directory
	EXIT  = "exit"  // Exit the CLI
	LS    = "ls"    // List directory contents
	CLEAR = "clear" // Clear screen
	MKDIR = "mkdir" // Create directory
)

// Extended commands for database and project management
const (
	HELP   = "help"   // Show help
	TEST   = "test"   // Run tests
	SQLITE = "sqlite" // Open SQLite database
	INITDB = "initdb" // Initialize database schema
	SQLF   = "sqlf"   // Execute SQL from file
	LOAD   = "load"   // Load CSV into table
	DUMP   = "dump"   // Export table to CSV
	SELECT = "select" // Execute SELECT query
	BUILD  = "build"  // Cross-compile Go binary
	ZIP    = "zip"    // Zip source code
	INIT   = "init"   // Initialize new project
	UPDATE = "update" // Execute UPDATE SQL
	DELETE = "delete" // Execute DELETE SQL
)

// ==================== StartCmdLine ====================

// StartCmdLine initializes the interactive command-line interface.
// It displays the prompt, reads user input, and executes commands in a loop.
func StartCmdLine() {
	// Recover from any panic (e.g., EXIT command uses panic for termination)
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("main: %+v \n", err)
		}
	}()
	defer CloseDB() // Ensure database connection is closed on exit

	fmt.Println(HelpInfo)

	lastCmd := HELP // Track last successful command for repetition

	in := bufio.NewReader(os.Stdin)

	for {
		// Build prompt with current working directory
		dir, _ := os.Getwd()
		pwd := filepath.Clean(dir)

		// Replace home directory with tilde for cleaner display
		if home := os.Getenv("HOME"); home != "" && strings.HasPrefix(pwd, home) {
			pwd = "~" + strings.TrimPrefix(pwd, home)
		}

		// Read command line input
		fmt.Printf("go@[%s]> ", pwd)

		// read string input until u enter return
		cmdLine, err := in.ReadString('\n')
		if err != nil {
			// Handle Ctrl+D (EOF) as exit
			if err == io.EOF {
				fmt.Println("\nbye!")
				break
			}
			fmt.Fprintf(os.Stderr, "read input error: %v\n", err)
			continue
		}

		// Clean and trim input
		cmdLine = strings.TrimSpace(cmdLine)
		// Repeat last command if input is empty
		if cmdLine == "" {
			cmdLine = lastCmd
			fmt.Println("Repeat last command ->", cmdLine)
		}

		// Execute the command
		if err := RunCmd(cmdLine); err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		} else {
			// Only update lastCmd on successful execution
			lastCmd = cmdLine
		}
	}

}

// ==================== RunCmd ====================

// RunCmd parses and executes a single command line.
// Returns an error if the command fails or is unknown.
func RunCmd(cmdLine string) error {

	args := strings.Fields(cmdLine)
	if len(args) == 0 {
		return nil // Empty command
	}

	cmd, params := args[0], args[1:]

	switch cmd {

	// Built-in shell commands

	case HELP:
		fmt.Println(HelpInfo)

	case CLEAR:
		// ANSI escape code to clear screen and move cursor to top-left
		fmt.Print("\033[2J\033[H")

	case PWD:
		return PrintPWD()

	case EXIT:
		panic("bye!")

	case CD:
		return ChangeDir(params[0])

	case LS:
		return ListFiles()

	case CAT:
		if len(params) == 0 {
			return errors.New("cat: missing file argument")
		}
		return CatFile(params)

	case MKDIR:
		if len(params) == 0 {
			return errors.New("mkdir: missing directory name")
		}
		return MakeDir(params[0])

	case SQLITE:
		dbPath := os.Args[0] + ".db" // Default: executable_name.db

		if len(params) > 0 {
			dbPath = params[0]
		}
		return OpenDB("sqlite", dbPath)

	case SQLF:
		if len(params) == 0 {
			return errors.New("sqlf: missing SQL file path")
		}
		return ExecSqlFile(params[0])

	case SELECT:
		return SelectPrint(cmdLine)

	case UPDATE, DELETE:
		return ExecSqlString(cmdLine)

	// Data import
	case LOAD:
		if len(params) == 0 {
			return errors.New("load: missing CSV file path")
		}
		filePath := params[0]
		tableName := strings.TrimSuffix(filepath.Base(filePath), ".csv")
		return LoadCsv(tableName, filePath)

	case DUMP: //dump db tables to csv
		// TODO: Implement table export to CSV
		return errors.New("dump: not yet implemented")

	case BUILD:
		if len(params) == 0 {
			return errors.New("build: missing target OS (windows/linux/darwin)")
		}
		return GoBuild(params[0])

	case ZIP:
		// Consider rewriting using Go's archive/zip for cross-platform compatibility
		cmd := `zip -r ./code/goat.zip ./go* -i "*.go" "*.mod" "*.sum" "*.html" "*.tpl" "*.js" "*.css"`
		return ExecShellCmd(cmd)

	case INIT:
		if len(params) == 0 {
			return errors.New("init: missing server/project name")
		}
		return InitProject(params[0])

	default:
		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", cmd)
	}

	return nil
}
