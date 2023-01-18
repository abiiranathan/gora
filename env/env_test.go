package env

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary file with some test data
	file, err := os.CreateTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	_, err = file.WriteString("KEY1=value1\nKEY2=123\nKEY3=true\nKEY4=3.14\nKEY5=456\n")
	if err != nil {
		t.Fatal(err)
	}

	// Define a struct to hold the config data
	type Config struct {
		Key1 string  `name:"KEY1"`
		Key2 int     `name:"KEY2"`
		Key3 bool    `name:"KEY3"`
		Key4 float64 `name:"KEY4"`
		Key5 uint    `name:"KEY5"`
	}
	config := &Config{}

	// Test loading the config file
	err = LoadConfig(file.Name(), config)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the config data was loaded correctly
	if config.Key1 != "value1" {
		t.Errorf("Expected Key1 to be 'value1', got %s", config.Key1)
	}
	if config.Key2 != 123 {
		t.Errorf("Expected Key2 to be 123, got %d", config.Key2)
	}
	if config.Key3 != true {
		t.Errorf("Expected Key3 to be true, got %t", config.Key3)
	}
	if config.Key4 != 3.14 {
		t.Errorf("Expected Key4 to be 3.14, got %f", config.Key4)
	}
	if config.Key5 != 456 {
		t.Errorf("Expected Key5 to be 456, got %d", config.Key5)
	}
}
