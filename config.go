package kekahu

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/koding/multiconfig"
)

// FindConfigPath returns the first file in path search list that exists.
func FindConfigPath() (string, error) {
	// Prepare PATH list
	paths := make([]string, 0, 3)

	// Look in CWD directory first
	if path, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(path, "kekahu"))
	}

	// Look in user's home directory next
	if user, err := user.Current(); err == nil {
		paths = append(paths, filepath.Join(user.HomeDir, ".kekahu"))
	}

	// Finally look in etc for the global configuration
	paths = append(paths, "/etc/kekahu")

	for _, path := range paths {
		for _, ext := range []string{".toml", ".json", ".yml", ".yaml"} {
			fpath := path + ext
			if _, err := os.Stat(fpath); !os.IsNotExist(err) {
				return fpath, nil
			}
		}
	}

	return "", errors.New("no configuration file found")
}

// Config uses the multiconfig loader and validators to store configuration
// values required for the kekahu service and to parse complex types.
type Config struct {
	Interval    string `default:"2m" validate:"duration" json:"interval"`              // the delay between heartbeats
	APIKey      string `required:"true" json:"api_key"`                                // API Key to access Kahu service
	URL         string `default:"https://kahu.bengfort.com" validate:"url" json:"url"` // Base URL of the Kahu service
	Verbosity   int    `default:"2" validate:"uint" json:"verbosity"`                  // Log verbosity, lower is more verbose
	PeersPath   string `default:"peers.json" validate:"path" json:"peers_path"`        // Path to save peers JSON file
	APITimeout  string `default:"5s" validate:"duration" json:"api_timeout"`           // Timeout for API HTTP requests
	PingTimeout string `default:"10s" validate:"duration" json:"ping_timeout"`         // Timeout for ping GRPC requests
}

// Load the configuration from default values, then from a configuration file,
// and finally from the environment. Validate the configuration on complete.
func (c *Config) Load() error {
	loaders := []multiconfig.Loader{}

	// Read default values defined via tag fields "default"
	loaders = append(loaders, &multiconfig.TagLoader{})

	// Find the config path and add the appropriate file loader
	if path, err := FindConfigPath(); err == nil {
		if strings.HasSuffix(path, "toml") {
			loaders = append(loaders, &multiconfig.TOMLLoader{Path: path})
		}

		if strings.HasSuffix(path, "json") {
			loaders = append(loaders, &multiconfig.JSONLoader{Path: path})
		}

		if strings.HasSuffix(path, "yml") || strings.HasSuffix(path, "yaml") {
			loaders = append(loaders, &multiconfig.YAMLLoader{Path: path})
		}
	}

	// Load the environment variable loader
	env := &multiconfig.EnvironmentLoader{Prefix: "KEKAHU", CamelCase: true}
	loaders = append(loaders, env)

	loader := multiconfig.MultiLoader(loaders...)
	if err := loader.Load(c); err != nil {
		return err
	}

	// Validate the loaded configuration
	validators := multiconfig.MultiValidator(
		&multiconfig.RequiredValidator{},
		&ComplexValidator{},
	)

	return validators.Validate(c)
}

// Update the configuration from another configuration struct
func (c *Config) Update(o *Config) error {
	conf := structs.New(c)

	// Then update the current config with values from the other config
	for _, field := range structs.Fields(o) {
		if !field.IsZero() {
			updateField := conf.Field(field.Name())
			updateField.Set(field.Value())
		}
	}

	// Validate the newly updated config
	validators := multiconfig.MultiValidator(
		&multiconfig.RequiredValidator{},
		&ComplexValidator{},
	)

	if err := validators.Validate(c); err != nil {
		return err
	}

	return nil
}

// GetURL parses the url and returns it
func (c *Config) GetURL() (*url.URL, error) {
	return url.Parse(c.URL)
}

// GetInterval parses the interval duration and returns it
func (c *Config) GetInterval() (time.Duration, error) {
	return time.ParseDuration(c.Interval)
}

// GetAPITimeout parses the api timeout duration and returns it
func (c *Config) GetAPITimeout() (time.Duration, error) {
	return time.ParseDuration(c.APITimeout)
}

// GetPingTimeout parses the ping timeout duration and returns it
func (c *Config) GetPingTimeout() (time.Duration, error) {
	return time.ParseDuration(c.PingTimeout)
}

//===========================================================================
// Validators
//===========================================================================

// ComplexValidator validates complex types that multiconfig doesn't understand
type ComplexValidator struct {
	TagName string
}

// Validate implements the multiconfig.Validator interface.
func (v *ComplexValidator) Validate(s interface{}) error {
	if v.TagName == "" {
		v.TagName = "validate"
	}

	for _, field := range structs.Fields(s) {
		if err := v.processField("", field); err != nil {
			return err
		}
	}

	return nil
}

func (v *ComplexValidator) processField(fieldName string, field *structs.Field) error {
	fieldName += field.Name()
	switch field.Kind() {
	case reflect.Struct:
		fieldName += "."
		for _, f := range field.Fields() {
			if err := v.processField(fieldName, f); err != nil {
				return err
			}
		}
	default:
		if field.IsZero() {
			return nil
		}

		switch strings.ToLower(field.Tag(v.TagName)) {
		case "":
			return nil
		case "duration":
			return v.processDurationField(fieldName, field)
		case "url":
			return v.processURLField(fieldName, field)
		case "path":
			return v.processPathField(fieldName, field)
		case "uint":
			return v.processUintField(fieldName, field)
		default:
			return fmt.Errorf("cannot validate type '%s'", field.Tag(v.TagName))
		}

	}

	return nil
}

func (v *ComplexValidator) processDurationField(fieldName string, field *structs.Field) error {
	_, err := time.ParseDuration(field.Value().(string))
	if err != nil {
		return fmt.Errorf("could not validate %s: %s", fieldName, err.Error())
	}
	return nil
}

func (v *ComplexValidator) processURLField(fieldName string, field *structs.Field) error {
	if _, err := url.Parse(field.Value().(string)); err != nil {
		return fmt.Errorf("could not validate %s: %s", fieldName, err.Error())
	}

	return nil
}

func (v *ComplexValidator) processPathField(fieldName string, field *structs.Field) error {
	// No path validation quite yet
	return nil
}

func (v *ComplexValidator) processUintField(fieldName string, field *structs.Field) error {
	val := field.Value().(int)
	if val < 0 {
		return fmt.Errorf("%s is less than zero", fieldName)
	}
	return nil
}
