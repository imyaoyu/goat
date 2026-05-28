package cli

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// ==================== Directory & File Operations ====================

// ChangeDir changes the current working directory.
// If dir is empty, it changes to the user's home directory.
// Supports both Unix ($HOME) and Windows (%USERPROFILE%).

func ChangeDir(dir string) error {
	if len(dir) == 0 {
		// Try HOME (Unix) first, fallback to USERPROFILE (Windows)
		dir = os.Getenv("HOME")
		if dir == "" {
			dir = os.Getenv("USERPROFILE")
		}
		if dir == "" {
			return fmt.Errorf("cannot determine home directory")
		}
	}
	return os.Chdir(dir)
}

// ListFiles prints all files and directories in the current working directory.
func ListFiles() error {
	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}
	for _, file := range files {
		fmt.Printf("%-v\n", file)
	}
	return nil
}

// CatFile prints the content of one or more files.
// Each file's content is separated by a visual divider.
func CatFile(fileNames []string) error {

	for i, fileName := range fileNames {
		data, err := os.ReadFile(fileName)
		if err != nil {
			return err
		}
		// Add separator between multiple files
		if i > 0 {
			fmt.Printf("\n--- %s ---\n", fileName)
		}

		fmt.Println(string(data))

	}

	return nil
}

// PrintPWD prints the absolute path of the current working directory.
func PrintPWD() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	pwd := filepath.Clean(dir)
	fmt.Println(pwd)
	return nil
}

// MakeDir creates a directory (and any necessary parent directories).
// Similar to `mkdir -p` in Unix. Does nothing if the directory already exists.
func MakeDir(dirName string) error {

	dir := filepath.Clean(dirName)

	// Check if directory already exists

	_, err := os.Stat(dir)
	if err == nil {
		fmt.Printf("Directory already exists: %s\n", dir)
		return nil
	}

	// Directory doesn't exist, create it with parents
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		fmt.Printf("Directory created: %s\n", dir)
		return nil
	}

	return nil

}

// ==================== Go Build ====================

// GoBuild cross-compiles the current Go module for the specified OS.
// Supported OS: windows, linux, darwin (macOS).
// The output binary is named as `{directory_name}-{os_type}`.
func GoBuild(osType string) error {
	// Locate the Go compiler
	goPath, err := exec.LookPath("go")

	if err != nil {
		fmt.Printf("Go command not found: %v\n", err)
		fmt.Println("Please ensure Go is installed and added to PATH")
		return err
	}
	fmt.Printf("Go path: %s\n", goPath)

	// Get current directory name as binary name
	dir, err := os.Getwd()
	if err != nil {
		return (err)
	}
	pwd := filepath.Clean(dir)

	cmdName := filepath.Base(pwd)

	outputFileName := fmt.Sprintf("%s-%s", cmdName, osType)

	outputPath := filepath.Join(pwd, outputFileName)

	cmd := exec.Command(goPath, "build", "-o", outputPath)

	// 设置交叉编译的环境变量
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		fmt.Sprintf("GOOS=%s", osType),
		"GOARCH=amd64",
	)

	fmt.Printf("Executing: %s\n", cmd.Path)
	fmt.Printf("Args: %v\n", cmd.Args)
	fmt.Printf("GOOS=%s, GOARCH=amd64\n", osType)

	// Run command and capture output

	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", output)
		}
		return err
	}

	fmt.Printf("Build successful: %s\n", outputPath)
	return nil
}

// ==================== ZIP Compression ====================

// ZipDirectory compresses the entire source directory into a ZIP file.
// It walks through all files and folders recursively.
// To add file filtering, modify the walk function.
func ZipDirectory(sourceDir, zipFile string) error {
	// Create ZIP file
	outFile, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 遍历目录
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Handle directories: add entry with trailing slash
		if info.IsDir() {
			_, err := zipWriter.Create(relPath + "/")
			return err
		}

		// Create file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Open source file and copy content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

// ==================== Shell Commands ====================

// ExecShellCmd executes a shell command using `sh -c`.
// Note: This is Unix-specific.
func ExecShellCmd(cmdStr string) error {

	fmt.Printf("Executing: sh -c '%s'\n", cmdStr)

	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Command failed: %v\n", err)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", output)
		}
		return err
	}

	if len(output) > 0 {
		fmt.Printf("Output: %s\n", output)
	}
	return nil

}

// ==================== Project Initialization ====================

// InitProject creates a new Go project directory with basic structure.
// It generates:
//   - A new directory with the given server name
//   - Initializes go.mod
//   - Creates a main.go file with a basic app template
func InitProject(serverName string) error {

	// Create project directory
	if err := MakeDir(serverName); err != nil {
		return (err)
	}

	// Change into project directory
	if err := ChangeDir(serverName); err != nil {
		return (err)
	}

	// Locate Go compiler
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Printf("Go command not found: %v\n", err)
		fmt.Println("Please ensure Go is installed and added to PATH")
		return err
	}
	fmt.Printf("go path: %s\n", goPath)

	// Initialize go.mod
	cmd := exec.Command(goPath, "mod", "init", serverName)
	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("go mod init failed: %s\n", serverName)
		return err
	}
	fmt.Println("go.mod initialized successfully")

	mainFile := `// GOAT generated: API server skeleton
package main

import (
	"goat/app"
)

func main() {

	// Example API endpoint: code "hi" returns success
	app.Add("hi", func(c *app.ApiCtx) {
		c.Log("Hello from generated API")
	})

	// Start the server
	app.Run()
}`

	if err := os.WriteFile(serverName+".go", []byte(mainFile), 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}
	fmt.Printf("Project %s created successfully!\n", serverName)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", serverName)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  go run %s.go\n", serverName)

	return nil
}
