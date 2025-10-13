package log

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the standard logging interface for the Cloupeer project.
type Logger interface {
	// Debug logs a message at DebugLevel.
	Debug(msg string, keysAndValues ...any)

	// Info logs a message at InfoLevel.
	Info(msg string, keysAndValues ...any)

	// Warn logs a message at WarnLevel.
	Warn(msg string, keysAndValues ...any)

	// Error logs a message at ErrorLevel.
	Error(err error, msg string, keysAndValues ...any)

	// WithName returns a new logger with the specified name appended.
	WithName(name string) Logger

	// WithValues returns a new logger with additional key-value pairs.
	WithValues(keysAndValues ...any) Logger

	// Logr returns a logr.Logger adapter for compatibility with controller-runtime.
	Logr() logr.Logger
}

// Static check to ensure zapLogger satisfies the Logger interface.
var _ Logger = (*zapLogger)(nil)

// zapLogger is the implementation of the Logger interface using zap.
type zapLogger struct {
	core *zap.Logger
}

// NewLogger creates a new Logger instance based on the provided options.
func NewLogger(opts *Options) Logger {
	if opts == nil {
		opts = NewOptions()
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:    "message",
		LevelKey:      "level",
		TimeKey:       "timestamp",
		NameKey:       "logger",
		CallerKey:     "caller",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendFloat64(float64(d) / float64(time.Millisecond))
		},
	}

	if opts.Format == "console" && opts.EnableColor {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	outputPaths := opts.OutputPaths
	if len(outputPaths) == 0 {
		outputPaths = []string{"stdout"}
	}

	cfg := &zap.Config{
		DisableCaller:    opts.DisableCaller,
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Encoding:         opts.Format,
		EncoderConfig:    encoderConfig,
		OutputPaths:      outputPaths,
		ErrorOutputPaths: []string{"stderr"},
	}

	core, err := cfg.Build(zap.AddCallerSkip(opts.CallerSkip), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		panic(fmt.Sprintf("failed to build zap logger: %v", err))
	}

	if opts.Name != "" {
		core = core.Named(opts.Name)
	}

	return &zapLogger{core: core}
}

func Debug(msg string, keysAndValues ...any)            { std.Debug(msg, keysAndValues...) }
func Info(msg string, keysAndValues ...any)             { std.Info(msg, keysAndValues...) }
func Warn(msg string, keysAndValues ...any)             { std.Warn(msg, keysAndValues...) }
func Error(err error, msg string, keysAndValues ...any) { std.Error(err, msg, keysAndValues...) }
func WithName(name string) Logger                       { return std.WithName(name) }
func WithValues(keysAndValues ...any) Logger            { return std.WithValues(keysAndValues...) }
func Logr() logr.Logger                                 { return std.Logr() }

func (z *zapLogger) Debug(msg string, keysAndValues ...any) {
	z.core.Debug(msg, toFields(keysAndValues...)...)
}

func (z *zapLogger) Info(msg string, keysAndValues ...any) {
	z.core.Info(msg, toFields(keysAndValues...)...)
}

func (z *zapLogger) Warn(msg string, keysAndValues ...any) {
	z.core.Warn(msg, toFields(keysAndValues...)...)
}

func (z *zapLogger) Error(err error, msg string, keysAndValues ...any) {
	fields := toFields(keysAndValues...)

	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	z.core.Error(msg, fields...)
}

func (z *zapLogger) WithName(name string) Logger {
	return &zapLogger{core: z.core.Named(name)}
}

func (z *zapLogger) WithValues(keysAndValues ...any) Logger {
	return &zapLogger{core: z.core.With(toFields(keysAndValues...)...)}
}

func (z *zapLogger) Logr() logr.Logger {
	return zapr.NewLogger(z.core)
}

var (
	once sync.Once

	std = NewNopLogger()
)

// Init initializes the global logger with the provided options.
// It is safe to call Init multiple times.
func Init(opts *Options) {
	once.Do(func() {
		std = NewLogger(opts)
	})
}

// Std returns the global logger instance.
func Std() Logger {
	return std
}

// NewNopLogger returns a logger that performs no operations.
func NewNopLogger() Logger {
	return &zapLogger{core: zap.NewNop()}
}
