// Copyright 2025 The Autopeer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"github.com/spf13/pflag"
)

// Options contains configuration settings for the logger.
type Options struct {
	// Name is an optional name for the logger, which will be added as a field to each log entry.
	Name string `json:"name,omitempty" mapstructure:"name"`

	// Level is the minimum log level to output. Can be 'debug', 'info', 'warn', 'error'.
	Level string `json:"level,omitempty" mapstructure:"level"`

	// Format specifies the log output format. Can be 'json' or 'console'.
	Format string `json:"format,omitempty" mapstructure:"format"`

	// EnableColor enables colorized output for console format.
	EnableColor bool `json:"enable-color,omitempty" mapstructure:"enable-color"`

	// DisableCaller stops annotating logs with the calling function's file name and line number.
	DisableCaller bool `json:"disable-caller,omitempty" mapstructure:"disable-caller"`

	// CallerSkip increases the number of callers skipped by caller annotation.
	// This is useful for building wrappers around the logger.
	CallerSkip int `json:"caller-skip,omitempty" mapstructure:"caller-skip"`

	// OutputPaths is a list of paths to write logs to. Use "stdout" or "stderr" for console output.
	// Defaults to ["stdout"].
	OutputPaths []string `json:"output-paths,omitempty" mapstructure:"output-paths"`
}

// NewOptions creates a new Options object with default values.
func NewOptions() *Options {
	return &Options{
		Level:       "info",
		Format:      "console",
		EnableColor: true,
		CallerSkip:  2, // Default to 2, which is correct for direct usage of the log package.
		OutputPaths: []string{"stdout"},
	}
}

// Validate validates all the required options.
// Currently a no-op, but provided for future extension.
func (o *Options) Validate() []error {
	return nil
}

// AddFlags binds command-line flags to the Options fields.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "log.name", o.Name, "An optional name for the logger.")
	fs.StringVar(&o.Format, "log.format", o.Format, "The log output format ('json' or 'console').")
	fs.BoolVar(&o.EnableColor, "log.enable-color", o.EnableColor, "Enable colorized output for the console format.")
	fs.IntVar(&o.CallerSkip, "log.caller-skip", o.CallerSkip, "The number of caller frames to skip.")

	usage := "The minimum log level to output (e.g., 'debug', 'info', 'warn', 'error')."
	fs.StringVar(&o.Level, "log.level", o.Level, usage)

	usage = "Disable the caller field in logs (file and line number)."
	fs.BoolVar(&o.DisableCaller, "log.disable-caller", o.DisableCaller, usage)

	usage = "A list of log output paths (e.g., 'stdout', '/var/log/app.log')."
	fs.StringSliceVar(&o.OutputPaths, "log.output-paths", o.OutputPaths, usage)
}
