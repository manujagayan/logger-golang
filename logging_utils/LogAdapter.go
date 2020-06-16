package logging_utils

import (
	"fmt"
	"github.com/nu7hatch/gouuid"
	"github.com/robfig/cron/v3"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"clean-base-template/app/config"
	"clean-base-template/domain/boundary/adapters"
)

// LogAdapter is used to provide structured log messages.
type LogAdapter struct {
	cfg config.LogConfig
	logger *lumberjack.Logger
	appName string
	msName string
}
var cr *cron.Cron

// NewLogAdapter creates a new Log adapter instance.
func NewLogAdapter(cfg config.LogConfig, appCfg config.AppConfig) (adapters.LogAdapterInterface, error) {

	a := &LogAdapter{
		cfg: cfg,
		appName: appCfg.AppName,
		msName: appCfg.MsName,
	}

	err := a.initLogFile()
	if err != nil {
		return nil, err
	}

	//register scheduler to rotate log files
	cr = cron.New()
	_, _ = cr.AddFunc("@daily", func() {
		err := a.logger.Rotate()
		if err != nil {
			fmt.Println(err)
			panic("Error occurred in log rotation")
		}
	})
	cr.Start()

	return a, nil
}

// Error logs a message as of error type.
func (a *LogAdapter) Error(message string, options ...interface{}) {
	a.log("ERROR", message, options)
}

// Debug logs a message as of debug type.
func (a *LogAdapter) Debug(message string, options ...interface{}) {
	a.log("DEBUG", message)
}

// Info logs a message as of information type.
func (a *LogAdapter) Info(message string, options ...interface{}) {
	a.log("INFO", message, options)
}

// Warn logs a message as of warning type.
func (a *LogAdapter) Warn(message string, options ...interface{}) {
	a.log("WARN", message, options)
}

// Destruct will close the logging_utils gracefully releasing all resources.
func (a *LogAdapter) Destruct() {

	if a.cfg.File {
		_ = a.logger.Close()
		cr.Stop()
	}
}

// Initialize the log file.
func (a *LogAdapter) initLogFile() error {

	if !a.cfg.File {
		return nil
	}

	ld := a.cfg.Directory
	a.logger = &lumberjack.Logger{
		 Filename:   ld + "/go-base-template.log",
		 LocalTime: true,
		 MaxSize:    a.cfg.MaxSize, // megabytes
		 MaxBackups: a.cfg.MaxBackup,
		 MaxAge:     a.cfg.MaxAge, //days
		 Compress:   a.cfg.Compress, // disabled by default

	}

	return nil
}

// Logs a message using the following format.
// <date> <time_in_24h_foramt_plus_milliseconds>|goRouteId|hostName|logLevel|loggerName|AppName|MicroserviceName|uuid|Message
// ex:
//2020-06-16 00:39:15.7164|[7]|105393-001L|INFO|application-log|clean-base-template|clean-base-template-ms|2ea75038-bc06-45c1-523a-0edd7978eab1|Controller started...
func (a *LogAdapter) log(logLevel string, message string, options ...interface{}) {

	// check whether the message should be logged
	if !a.isLoggable(logLevel) {
		return
	}

	m := a.formatMessage(logLevel, message, options)

	a.logToConsole(m)
	a.logToFile(m)
}

// formatMessage create log message according to log pattern.
func (a *LogAdapter) formatMessage(logLevel string, message string, options ...interface{}) string {

	now := time.Now().Format("2006-01-02 15:04:05.0000")
	uuidV, _ := uuid.NewV4()
	goId := goid()
	hostname,err := os.Hostname()
	if err != nil {
		fmt.Println("Error occurred at requesting host name")
	}
	loggerName := "application-log"
	appName := a.appName
	msName := a.msName

	return fmt.Sprintf("%s|[%v]|%s|%s|%s|%s|%s|%v|%s", now, goId, hostname, logLevel, loggerName,appName, msName, uuidV, message)
}

// Check whether the message should be logged depending on the log level setting.
func (a *LogAdapter) isLoggable(logLevel string) bool {

	l := map[string]int{
		"ERROR": 1,
		"DEBUG": 2,
		"WARN":  3,
		"INFO":  4,
	}

	return l[logLevel] >= l[a.cfg.Level]
}

func goid() int {

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

// Logs a message to the console.
func (a *LogAdapter) logToConsole(message string) {

	if a.cfg.Console {
		fmt.Println(message)
	}
}

// Logs a message to a file.
func (a *LogAdapter) logToFile(message string) {

	if !a.cfg.File {
		return
	}

	_, err := a.logger.Write([]byte(message + "\n"))
	if err != nil {
		fmt.Println(err)
	}
}
