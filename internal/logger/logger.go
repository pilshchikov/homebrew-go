package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel defines the logging level
type LogLevel int

const (
	// DebugLevel shows all logs
	DebugLevel LogLevel = iota
	// InfoLevel shows info, warning and error logs
	InfoLevel
	// WarnLevel shows warning and error logs
	WarnLevel
	// ErrorLevel shows only error logs
	ErrorLevel
	// QuietLevel shows no logs
	QuietLevel
)

var (
	currentLevel = InfoLevel
	debugLogger  *log.Logger
	infoLogger   *log.Logger
	warnLogger   *log.Logger
	errorLogger  *log.Logger
)

// Init initializes the logger with the given configuration
func Init(debug, verbose, quiet bool) {
	// Determine log level - debug has highest priority
	if debug {
		currentLevel = DebugLevel
	} else if quiet {
		currentLevel = QuietLevel
	} else if verbose {
		currentLevel = InfoLevel
	} else {
		// Default to InfoLevel for normal operation output
		currentLevel = InfoLevel
	}

	// Create loggers
	debugLogger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	infoLogger = log.New(os.Stdout, "", 0)
	warnLogger = log.New(os.Stderr, "Warning: ", 0)
	errorLogger = log.New(os.Stderr, "Error: ", 0)
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if currentLevel <= DebugLevel {
		debugLogger.Printf(format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if currentLevel <= InfoLevel {
		infoLogger.Printf(format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if currentLevel <= WarnLevel {
		warnLogger.Printf(format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if currentLevel <= ErrorLevel {
		errorLogger.Printf(format, args...)
	}
}

// Success logs a success message with green color
func Success(format string, args ...interface{}) {
	if currentLevel <= InfoLevel {
		message := fmt.Sprintf(format, args...)
		infoLogger.Printf("\033[32m%s\033[0m", message)
	}
}

// Failure logs a failure message with red color
func Failure(format string, args ...interface{}) {
	if currentLevel <= ErrorLevel {
		message := fmt.Sprintf(format, args...)
		errorLogger.Printf("\033[31m%s\033[0m", message)
	}
}

// Progress logs a progress message
func Progress(format string, args ...interface{}) {
	if currentLevel <= InfoLevel {
		message := fmt.Sprintf(format, args...)
		infoLogger.Printf("==> %s", message)
	}
}

// Step logs a step message
func Step(format string, args ...interface{}) {
	if currentLevel <= InfoLevel {
		message := fmt.Sprintf(format, args...)
		infoLogger.Printf("  - %s", message)
	}
}

// Cmd logs a command being executed
func Cmd(format string, args ...interface{}) {
	if currentLevel <= InfoLevel {
		message := fmt.Sprintf(format, args...)
		infoLogger.Printf("$ %s", message)
	}
}

// PrintDivider prints a visual divider
func PrintDivider() {
	if currentLevel <= InfoLevel {
		infoLogger.Println(strings.Repeat("=", 80))
	}
}

// PrintHeader prints a header with the given title
func PrintHeader(title string) {
	if currentLevel <= InfoLevel {
		divider := strings.Repeat("=", len(title)+4)
		infoLogger.Printf("%s", divider)
		infoLogger.Printf("  %s", title)
		infoLogger.Printf("%s", divider)
	}
}

// Timer represents a timing operation
type Timer struct {
	name  string
	start time.Time
}

// NewTimer creates a new timer
func NewTimer(name string) *Timer {
	return &Timer{
		name:  name,
		start: time.Now(),
	}
}

// Stop stops the timer and logs the duration
func (t *Timer) Stop() {
	duration := time.Since(t.start)
	Info("%s took %v", t.name, duration)
}

// StopWithResult stops the timer and logs the duration with result
func (t *Timer) StopWithResult(result string) {
	duration := time.Since(t.start)
	Info("%s %s (%v)", t.name, result, duration)
}

// Fatal logs an error and exits with code 1
func Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	errorLogger.Printf("\033[31mError: %s\033[0m", message)
	os.Exit(1)
}

// Question asks the user a question and returns their response
func Question(format string, args ...interface{}) string {
	if currentLevel <= InfoLevel {
		message := fmt.Sprintf(format, args...)
		fmt.Print(message)
	}
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.TrimSpace(response)
}

// Confirm asks for yes/no confirmation
func Confirm(format string, args ...interface{}) bool {
	response := Question(format+" [y/N]: ", args...)
	response = strings.ToLower(response)
	return response == "y" || response == "yes"
}

// ProgressSpinner shows a simple progress indicator
type ProgressSpinner struct {
	message string
	done    chan bool
}

// NewProgressSpinner creates a new progress spinner
func NewProgressSpinner(message string) *ProgressSpinner {
	return &ProgressSpinner{
		message: message,
		done:    make(chan bool),
	}
}

// Start begins the spinner animation
func (p *ProgressSpinner) Start() {
	if currentLevel > InfoLevel {
		return
	}

	go func() {
		spinners := []string{"|", "/", "-", "\\"}
		i := 0
		for {
			select {
			case <-p.done:
				return
			default:
				fmt.Printf("\r%s %s ", spinners[i%len(spinners)], p.message)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner
func (p *ProgressSpinner) Stop() {
	if currentLevel > InfoLevel {
		return
	}

	p.done <- true
	fmt.Print("\r")
}

// ErrorContext provides structured error information
type ErrorContext struct {
	Operation   string
	Formula     string
	Version     string
	Platform    string
	Error       error
	Suggestions []string
}

// LogDetailedError logs a detailed error with context and suggestions
func LogDetailedError(ctx ErrorContext) {
	if currentLevel > ErrorLevel {
		return
	}

	errorLogger.Printf("\033[31m==> Error: %s failed\033[0m", ctx.Operation)
	if ctx.Formula != "" {
		errorLogger.Printf("Formula: %s", ctx.Formula)
	}
	if ctx.Version != "" {
		errorLogger.Printf("Version: %s", ctx.Version)
	}
	if ctx.Platform != "" {
		errorLogger.Printf("Platform: %s", ctx.Platform)
	}
	errorLogger.Printf("Reason: %v", ctx.Error)

	if len(ctx.Suggestions) > 0 {
		errorLogger.Printf("\nSuggestions:")
		for _, suggestion := range ctx.Suggestions {
			errorLogger.Printf("  - %s", suggestion)
		}
	}
	errorLogger.Println()
}

// IsQuiet returns true if the logger is in quiet mode
func IsQuiet() bool {
	return currentLevel >= QuietLevel
}

// GetCurrentLevel returns the current log level
func GetCurrentLevel() LogLevel {
	return currentLevel
}
