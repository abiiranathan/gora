package env

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type Config struct {
	Key1 int64
	Key2 bool
	Key3 string
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	// Set up a temporary directory for the test files
	dir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a test configuration file
	filename := filepath.Join(dir, "test.env")
	err = os.WriteFile(filename, []byte(`
		Key1=123
		Key2=true
		Key3=hello
	`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set up the converter functions
	converters := map[string]ConfigConverter{
		"Key1": func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 10, 64)
		},
		"Key2": func(s string) (interface{}, error) {
			return strconv.ParseBool(s)
		},
		"Key3": func(s string) (interface{}, error) {
			return s, nil
		},
	}

	// Load the configuration
	config := &Config{}
	err = LoadConfig(filename, config, converters)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the configuration was loaded correctly
	if config.Key1 != 123 {
		t.Errorf("Expected Key1 to be 123, got %d", config.Key1)
	}
	if config.Key2 != true {
		t.Errorf("Expected Key2 to be true, got %t", config.Key2)
	}

	if config.Key3 != "hello" {
		t.Errorf("Expected Key3 to be 'hello', got '%s'", config.Key3)
	}

	// Test that everything is a string by default is converter not passed
	type Config struct {
		Key1 string
		Key2 string
		Key3 string
	}

	conf := &Config{}
	err = LoadConfig(filename, conf, map[string]ConfigConverter{})
	if err != nil {
		t.Fatal(err)
	}

	if conf.Key1 != "123" {
		t.Errorf("Expected Key1 to be 123, got %s", conf.Key1)
	}
	if conf.Key2 != "true" {
		t.Errorf("Expected Key2 to be true, got %s", conf.Key2)
	}

	if conf.Key3 != "hello" {
		t.Errorf("Expected Key3 to be 'hello', got '%s'", conf.Key3)
	}

}

func TestLoadConfigError(t *testing.T) {
	t.Parallel()

	// Set up a temporary directory for the test files
	dir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a test configuration file with an invalid value
	filename := filepath.Join(dir, "test.env")
	err = os.WriteFile(filename, []byte(`
		Key1=invalid
		Key2=true
		Key3=hello
	`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set up the converter functions
	converters := map[string]ConfigConverter{
		"Key1": func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 10, 64)
		},
		"Key2": func(s string) (interface{}, error) {
			return strconv.ParseBool(s)
		},
		"Key3": func(s string) (interface{}, error) {
			return s, nil
		},
	}

	// Load the configuration
	config := &Config{}
	err = LoadConfig(filename, config, converters)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestLoadConfigEmpty(t *testing.T) {
	t.Parallel()

	// Set up a temporary directory for the test files
	dir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a test configuration file with no key-value pairs
	filename := filepath.Join(dir, "test.env")
	err = os.WriteFile(filename, []byte(``), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set up the converter functions
	converters := map[string]ConfigConverter{
		"Key1": func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 10, 64)
		},
		"Key2": func(s string) (interface{}, error) {
			return strconv.ParseBool(s)
		},
		"Key3": func(s string) (interface{}, error) {
			return s, nil
		},
	}

	// Load the configuration
	config := &Config{}
	err = LoadConfig(filename, config, converters)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the configuration was loaded with the default values
	if config.Key1 != 0 {
		t.Errorf("Expected Key1 to be 0, got %d", config.Key1)
	}
	if config.Key2 != false {
		t.Errorf("Expected Key2 to be false, got %t", config.Key2)
	}
	if config.Key3 != "" {
		t.Errorf("Expected Key3 to be '', got '%s'", config.Key3)
	}
}

func TestLoadConfigComments(t *testing.T) {
	t.Parallel()

	// Set up a temporary directory for the test files
	dir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a test configuration file with comments
	filename := filepath.Join(dir, "test.env")
	err = os.WriteFile(filename, []byte(`
		# This is a comment
		Key1=123
		Key2=true
		Key3=hello
		# Another comment
	`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set up the converter functions
	converters := map[string]ConfigConverter{
		"Key1": func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 10, 64)
		},
		"Key2": func(s string) (interface{}, error) {
			return strconv.ParseBool(s)
		},
		"Key3": func(s string) (interface{}, error) {
			return s, nil
		},
	}

	// Load the configuration
	config := &Config{}
	err = LoadConfig(filename, config, converters)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the configuration was loaded correctly
	if config.Key1 != 123 {
		t.Errorf("Expected Key1 to be 123, got %d", config.Key1)
	}
	if config.Key2 != true {
		t.Errorf("Expected Key2 to be true, got %t", config.Key2)
	}
	if config.Key3 != "hello" {
		t.Errorf("Expected Key3 to be 'hello', got '%s'", config.Key3)
	}
}

func TestLoadConfigFull(t *testing.T) {
	// Set up a temporary directory for the test files
	dir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a test configuration file
	filename := filepath.Join(dir, "test.env")
	err = os.WriteFile(filename, []byte(`
		Key1=123
		Key2=true
		Key3=hello
		Key4=1.23
	`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set up the converter functions
	converters := map[string]ConfigConverter{
		"Key1": func(s string) (interface{}, error) {
			return strconv.ParseInt(s, 10, 64)
		},
		"Key2": func(s string) (interface{}, error) {
			return strconv.ParseBool(s)
		},
		"Key3": func(s string) (interface{}, error) {
			return s, nil
		},
		"Key4": func(s string) (interface{}, error) {
			return strconv.ParseFloat(s, 64)
		},
	}

	// Test loading into a struct
	type ConfigStruct struct {
		Key1 int64
		Key2 bool
		Key3 string
		Key4 float64
	}
	configStruct := &ConfigStruct{}
	err = LoadConfig(filename, configStruct, converters)
	if err != nil {
		t.Errorf("LoadConfig returned unexpected error for struct: %v", err)
	}
	if configStruct.Key1 != 123 {
		t.Errorf("Expected Key1 to be 123, got %d", configStruct.Key1)
	}
	if configStruct.Key2 != true {
		t.Errorf("Expected Key2 to be true, got %t", configStruct.Key2)
	}
	if configStruct.Key3 != "hello" {
		t.Errorf("Expected Key3 to be 'hello', got '%s'", configStruct.Key3)
	}
	if configStruct.Key4 != 1.23 {
		t.Errorf("Expected Key4 to be 1.23, got %f", configStruct.Key4)
	}

	// Test loading into a map
	configMap := make(map[string]interface{})
	err = LoadConfig(filename, &configMap, converters)
	if err != nil {
		t.Errorf("LoadConfig returned unexpected error for map: %v", err)
	}
	if configMap["Key1"] != int64(123) {
		t.Errorf("Expected Key1 to be 123, got %d", configMap["Key1"])
	}
	if configMap["Key2"] != true {
		t.Errorf("Expected Key2 to be true, got %t", configMap["Key2"])
	}
	if configMap["Key3"] != "hello" {
		t.Errorf("Expected Key3 to be 'hello', got '%s'", configMap["Key3"])
	}
	if configMap["Key4"] != 1.23 {
		t.Errorf("Expected Key4 to be 1.23, got %f", configMap["Key4"])
	}

	// Test missing file
	err = LoadConfig("nonexistent.env", &configStruct, converters)
	if err == nil {
		t.Errorf("LoadConfig returned nil error for missing file")
	}

	// Test invalid config value
	err = LoadConfig(filename, 123, converters)
	if err == nil {
		t.Errorf("LoadConfig returned nil error for invalid config value")
	}

}
