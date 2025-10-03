# Flexible unmarshalling of values into Golang objects

[![Go Report Card](https://goreportcard.com/badge/github.com/dihedron/rawdata)](https://goreportcard.com/report/github.com/dihedron/rawdata)
[![Godoc](https://godoc.org/github.com/dihedron/rawdata?status.svg)](https://godoc.org/github.com/dihedron/rawdata)

This library provides a facility to unmarshal unknown values from both inline and on-disk values, in either JSON or YAML formats.

It can be used wherever an input must be unmarshalled into a Golang object.

## Importing the library

In order to use the library, import it like this:

```golang
import (
    "github.com/dihedron/rawdata"
)
```

Then open a command prompt in your project's root directory and run:

```bash
$> go mod tidy
```

## Using the library for command line flags

One use case for this library is alongside Jesse van den Keiboom's [Flags library](https://github.com/jessevdk/go-flags), to simplify the unmarshalling of complex command line values into Golang structs and arrays. This provides a simple and elegant way to support complex configurations on the command line.

Let's see the trivial case first, where the library provides the boilerplate code needed to unmarshal a well-known JSON/YAML data structure into a defined Golang struct/array.  
In this case you would use the `UnmarshalInto` function, which expects a pointer to the destination struct/array to be passed in, so the object and the input value must both be known in advance and match one another. 

```golang
type MyCommand struct {
    Param1     CustomFlagType1  `short:"p" long:"param1" description:"An input parameter, either as an inline value or as a @file (in JSON or YAML format)."`
    Param2     CustomFlagType2  `short:"q" long:"param2" description:"A partially deserialised input parameter, either as an inline value or as a @file (in JSON or YAML format)."`
}

type CustomFlagType1 struct {
    Name    string `json:"name,omitempty" yaml:"name,omitempty"`
    Surname string `json:"surname,omitempty" yaml:"surname,omitempty"`
    Age     int    `json:"age,omitempty" yaml:"age,omitempty"`
}

func (c *CustomFlagType1) UnmarshalFlag(value string) error {
    return rawdata.UnmarshalInto(value, c)
}

```

The less trivial use case is when the exact type of the input data is not perfectly known in advance or it varies depending on e.g. a `type` field.

In this case you can use the `Unmarshal` function, which is more lax with respect to `UnmarshalInto`: it detects the type of entity (object/array) in the input and *returns* either a `map[string]any` (if the input value is an object) or a `[]any` if the input is an array of objects. It is up to the caller to handle the two cases properly, and this leaves the possibility of using e.g. such tools as Mitchell Hashimoto's [Map Structure](https://github.com/mitchellh/mapstructure) library to perform the final unmarshalling into the destination data structure, possibly with some switching logic. Overall, this provides a way to perform a smarter, adaptive staged unmarshalling where you partially unmarshall into an intermediate data structure, analyse it and decide what to do next.

```golang
type CustomFlagType2 struct {
    // ...
}

func (c *CustomFlagType2) UnmarshalFlag(value string) error {
    var err error
    data, err = rawdata.Unmarshal(value)
    // after this call, data may contain a map[string]any 
    // or a []any, depending on whether the input is a 
    // JSON/YAML object or an array; you can hook your custom 
    // unmarshalling logic here
    switch data := data.(type) {
    case map[string]any:
        // it's a map, switch on the "type" field
        // retrieve the "type" value and cast it to string, 
        if v, ok := data["type"] ; ok {
            // the value is there, attempt casting it to string
            if t, ok := v.(string); ok {
                switch(t) {
                case "foo":
                    // do whatever you need to do with a type "foo";
                    // the CustomFlagType2 pointer allows manipulation 
                    // of the struct
                    c.SomeField = data["key_dependent_on_type_foo"]
                    // and so on...
                case "bar":
                    // ...same for "bar"
                }
            }
        }
    case []any:
        // logic to handle arrays here
	default:
		return errors.New("unexpected type of returned data")
	}    
    return err
}

```

