/*
Package env provides functions for loading and parsing configuration files in the '.env' format.

The '.env' format is a simple key-value format, with one key-value pair per line.
Lines that begin with a '#' character are treated as comments and are ignored.
Keys and values may be surrounded by quotes, but this is not required.

The LoadConfig function can be used to load a configuration file and parse its
key-value pairs into a struct or map. Converter functions can be provided to
specify how each key's value should be parsed.

The LoadEnv function can be used to load a configuration file and set
the corresponding environment variables for the current process.
*/
package env

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type ConfigConverter func(string) (any, error)

// LoadConfig loads the key-value pairs from a configuration file in the '.env'
// format and parses them into the provided config value. The config value must
// be a pointer to a struct or map. Converter functions can be provided to
// specify how each key's value should be parsed.
//
// Lines that begin with a '#' character are treated as comments and are
// ignored. Keys and values may be surrounded by quotes, but this is not
// required.
//
// If the file does not exist or cannot be read, or if a key does not have a
// corresponding field or key in the config value, an error is returned.
func LoadConfig(filename string, config interface{}, converters map[string]ConfigConverter) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		converter, ok := converters[key]
		if !ok {
			continue
		}
		result, err := converter(value)
		if err != nil {
			return fmt.Errorf("invalid data type conversion for %s: %v", key, err)
		}

		// Set the value in the config value using reflection
		v := reflect.ValueOf(config)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Map {
			mapKey := reflect.ValueOf(key)
			v.SetMapIndex(mapKey, reflect.ValueOf(result))
		} else if v.Kind() == reflect.Struct {
			field := v.FieldByName(key)
			if !field.IsValid() {
				return fmt.Errorf("no field named %s in config struct", key)
			}
			field.Set(reflect.ValueOf(result))
		} else {
			return fmt.Errorf("config value must be a pointer to a struct or map")
		}
	}

	return scanner.Err()
}

// LoadEnv loads the key-value pairs from a configuration file in the '.env'
// format and sets the corresponding environment variables for the current
// process. If a key is already set in the environment, it is overwritten.
// Lines that begin with a '#' character are treated as comments and are
// ignored. Keys and values may be surrounded by quotes, but this is not
// required.
//
// If the file does not exist or cannot be read, an error is returned.
func LoadEnv(filename string) error {
	// Open the configuration file
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Parse the key-value pairs
	pairs, err := parseEnv(f)
	if err != nil {
		return err
	}

	// Set the environment variables
	for _, pair := range pairs {
		if err := os.Setenv(pair.Key, pair.Value); err != nil {
			return err
		}
	}

	return nil
}

// parseEnv parses the key-value pairs from an '.env' file. Lines that begin
// with a '#' character are treated as comments and are ignored. Keys and
// values may be surrounded by quotes, but this is not required.
//
// If the file cannot be read, an error is returned.
func parseEnv(r io.Reader) ([]KeyValuePair, error) {
	var pairs []KeyValuePair

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Split the line into key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Unquote the value if necessary
		if len(value) > 1 && value[0] == '"' && value[len(value)-1] == '"' {
			v, err := strconv.Unquote(value)
			if err != nil {
				return nil, err
			}

			value = v
		}

		pairs = append(pairs, KeyValuePair{Key: key, Value: value})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pairs, nil
}

// KeyValuePair represents a key-value pair in an '.env' file.
type KeyValuePair struct {
	Key   string
	Value string
}
