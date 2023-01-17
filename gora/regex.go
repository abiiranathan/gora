package gora

import (
	"errors"
	"regexp"
	"strings"
)

// convert a pathPrefix into a valid regex string.
// Supports custom types: int, str, float, bool, date, datetime
func pathPrefixToRegex(pathPrefix string) (string, error) {
	// Split the path prefix into its individual segments
	segments := strings.Split(pathPrefix, "/")

	// Initialize the regular expression
	regex := "^/"

	// Iterate through the segments
	numSigments := len(segments)
	for index, segment := range segments {
		// Check if the segment is a path parameter
		if strings.Contains(segment, "{") && strings.Contains(segment, "}") {
			// Extract the parameter name from the segment
			paramName := segment[1 : len(segment)-1]

			// Check if the parameter has a type specified
			if strings.Contains(paramName, ":") {
				// Extract the parameter name and type from the segment
				var paramType string
				params := strings.Split(paramName, ":")
				paramName = params[0]
				paramType = params[1]

				// Check the parameter type and add the corresponding regular expression to the regex
				if paramType == "int" {
					regex += "(?P<" + paramName + ">\\d+)"
				} else if paramType == "str" {
					regex += "(?P<" + paramName + ">\\w+)"
				} else if paramType == "float" {
					regex += "(?P<" + paramName + ">\\d+\\.\\d+)"
				} else if paramType == "bool" {
					regex += "(?P<" + paramName + ">true|false)"
				} else if paramType == "date" {
					regex += "(?P<" + paramName + ">\\d{4}-\\d{2}-\\d{2})"
				} else if paramType == "datetime" {
					regex += "(?P<" + paramName + ">\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2})"
				} else {
					return "", errors.New("invalid parameter type: " + paramType)
				}
			} else {
				// The parameter has no type specified, so consider it a string
				regex += "(?P<" + paramName + ">\\w+)"
			}
		} else {
			// Add the segment to the regex as is
			regex += segment
		}

		if index < numSigments-1 && segment != "" {
			regex += "/"
		}
	}

	// Add trailing slash if StrictSlash and path does not end in /
	if StrictSlash && len(regex) > 2 && regex[len(regex)-1] != '/' {
		regex += "/"
	}

	// Add the end anchor to the regex
	regex += "$"
	return regex, nil
}

// Compiles a regex pattern string into a regexp.Regexp
// panics if pattern is not valid.
func compileRegex(pat string) *regexp.Regexp {
	regex, err := pathPrefixToRegex(pat)
	if err != nil {
		panic(err)
	}
	return regexp.MustCompile(regex)
}
