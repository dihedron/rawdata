package rawdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format is the type representing the possible formats
// for the complex flag structure.
type Format uint8

const (
	// FormatUnknown indicates that the format could not be determined.
	FormatUnknown Format = iota
	// FormatJSON indicates that the flag is in JSON format.
	FormatJSON
	// FormatYAML indicates that the flag is in YAML format.
	FormatYAML
)

// Unmarshal unmarshals a complex value into an object; if the value
// starts with a '@' it is assumed to be a file on the local filesystem,
// it is read into memory and then unmarshalled into a generic map or
// array depending on the contents; if it does not start with '@', it
// can be either a YAML inline representation (in which case it MUST
// start with '---') or an inline JSON representation and is unmarshalled
// accordingly.
func Unmarshal(value string) (any, error) {
	// read data and detect its format
	format, content, err := ReadContent(value)
	if err != nil {
		return nil, err
	}
	// now depending on the format, unmarshal to JSON or YAML
	switch format {
	case FormatJSON:
		return unmarshalJSON(content)
	case FormatYAML:
		return unmarshalYAML(content)
	default:
		return nil, fmt.Errorf("unsupported encoding: %v", format)
	}
}

// UnmarshalInto is a more type-constrained version of Unmarshal: it requires
// the output object (either a struct or an array) to passed in as a pointer.
// The input value can either be an inline JSON/YAM value, or a reference to
// a file (e.g. '@myfile.json') in JSON/YAML format.
func UnmarshalInto(value string, target any) error {
	// read data and detect its format
	format, content, err := ReadContent(value)
	if err != nil {
		return err
	} // now depending on the format, unmarshal to JSON or YAML
	switch format {
	case FormatJSON:
		if err := json.Unmarshal(content, target); err != nil {
			return fmt.Errorf("error unmarshalling from JSON: %w", err)
		}
		return nil
	case FormatYAML:
		if err := yaml.Unmarshal(content, target); err != nil {
			return fmt.Errorf("error unmarshalling from YAML: %w (%T)", err, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported encoding: %v", format)
	}
}

// ReadContent reads the data from the given input value,either taken as the
// literal value to be parsed or as a path to a file (in either JSON or YAML
// format); it returns the auto-detected data format and the data itself as a
// byte slice.
func ReadContent(value string) (Format, []byte, error) {
	var format Format
	var content []byte
	if strings.HasPrefix(value, "@") {
		// it's a file on disk, check it exist
		filename := strings.TrimPrefix(value, "@")
		info, err := os.Stat(filename)
		if os.IsNotExist(err) {
			return format, nil, fmt.Errorf("file '%s' does not exist: %w", filename, err)
		}
		if info.IsDir() {
			return format, nil, fmt.Errorf("'%s' is a directory, not a file", filename)
		}
		// read into memory
		content, err = os.ReadFile(filename)
		if err != nil {
			return format, nil, fmt.Errorf("error reading file '%s': %w", filename, err)
		}
		// type detection is based on file extension
		ext := path.Ext(filename)
		switch strings.ToLower(ext) {
		case ".yaml", ".yml":
			format = FormatYAML
		case ".json":
			format = FormatJSON
		default:
			return format, nil, fmt.Errorf("unsupported data format in file: %s", path.Ext(filename))
		}
	} else {
		// not a file, type detection is based on the data
		value = strings.TrimSpace(value)
		content = []byte(value)
		if strings.HasPrefix(value, "---") {
			format = FormatYAML
		} else if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
			// TODO: we could optimise by recording whether it's a struct or an array
			format = FormatJSON
		} else {
			return format, nil, fmt.Errorf("unrecognisable input format in inline data")
		}
	}
	return format, content, nil
}

// unmarshalJSON unmarshals a JSON document; a JSON document can
// represent either an object or an array but the standard library
// methods expect the target object to be pre-allocated; thus, we
// try to unmarshal to a map, which is the most general representation
// of a struct; if it fails with a parse error because the JSON document
// represents an array, we try with an array next.
func unmarshalJSON(content []byte) (any, error) {
	// first attempt: unmarshalling to a map (like a struct would)...
	m := map[string]any{}
	if err := json.Unmarshal(content, &m); err != nil {
		if err, ok := err.(*json.UnmarshalTypeError); ok {
			if err.Value == "array" && err.Offset == 1 {
				// second attempt: it is not a struct, it's an array, let's try that...
				a := []any{}
				if err := json.Unmarshal(content, &a); err != nil {
					return nil, fmt.Errorf("error unmarshalling from JSON: %w", err)
				}
				return a, nil
			}
		}
		return nil, fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return m, nil
}

// unmarshalYAML unmarshals a YAML document; a YAML document can
// represent either an object or an array but the YAML library
// methods expect the target object to be pre-allocated; thus, we
// try to unmarshal to a map, which is the most general representation
// of a struct; if it fails with a parse error because the YAML document
// represents an array, we try with an array next.
func unmarshalYAML(content []byte) (any, error) {
	object := map[string]any{}
	if err := yaml.Unmarshal(content, object); err != nil {
		if err, ok := err.(*yaml.TypeError); ok {
			// TODO: find a way to circumvent marshalling error in case of array
			for _, e := range err.Errors {
				if strings.HasSuffix(e, "cannot unmarshal !!seq into map[string]interface {}") {
					// second attempt: it is not a struct, it's an array, let's try that...
					a := []any{}
					if err := yaml.Unmarshal(content, &a); err != nil {
						return nil, fmt.Errorf("error unmarshalling from YAML: %w", err)
					}
					return a, nil
				}
			}
			return nil, fmt.Errorf("error: %s, %+v", err.Error(), err.Errors)
		}
		return nil, fmt.Errorf("error unmarshalling from YAML: %w (%T)", err, err)
	}
	return object, nil
}
