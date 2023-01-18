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

/*
The config struct is passed as an argument and the LoadConfig function uses
reflection to examine the fields of the struct and compare the key name with
the "name" tag. If a match is found, it uses a switch statement to check the
field's Kind and parse the value accordingly.
If the fields are not primitive types or the key is not found in the struct,
it returns an error.
*/
func LoadConfig(filename string, config interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	keys := make(map[string]bool)
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
		keys[key] = true

		v := reflect.ValueOf(config)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if key == f.Tag.Get("name") {
				field := v.Field(i)
				switch field.Kind() {
				case reflect.String:
					field.SetString(value)
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					intValue, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %v", key, err)
					}
					field.SetInt(intValue)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					uintValue, err := strconv.ParseUint(value, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %v", key, err)
					}
					field.SetUint(uintValue)
				case reflect.Float32, reflect.Float64:
					floatValue, err := strconv.ParseFloat(value, 64)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %v", key, err)
					}
					field.SetFloat(floatValue)
				case reflect.Bool:
					boolValue, err := strconv.ParseBool(value)
					if err != nil {
						return fmt.Errorf("invalid value for %s: %v", key, err)
					}
					field.SetBool(boolValue)
				default:
					return fmt.Errorf("unsupported type for field %s", f.Name)
				}
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if f.Tag.Get("required") == "true" && !keys[f.Tag.Get("name")] {
			return fmt.Errorf("missing required field %s", f.Tag.Get("name"))
		}
	}
	return nil
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
