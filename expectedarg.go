package gli

import (
    "reflect"
    "strings"
)


type ExpectedArg struct {
    FieldName   string
    Keys        []string
    ArgType     reflect.Value
    DefaultVal  string
    Required    bool
    Description string
    Override    bool
}


// constructor for an expected argument
func newExpectedArg(field reflect.StructField, val reflect.Value) *ExpectedArg {
    // skip fields without a gli tag
    gliTag := field.Tag.Get("gli")
    if "" == gliTag { return nil }

    // check arg options
    required  := strings.Contains(gliTag, "!")
    override  := strings.Contains(gliTag, "^")
    if required || override {
        rep := strings.NewReplacer("!", "", "^", "")
        gliTag = rep.Replace(gliTag)
    }

    // parse description tag
    description := field.Tag.Get("description")

    // parse default tag for all but bools, arrays and slices (they cannot have default values)
    defaultVal := field.Tag.Get("default")
    if reflect.Bool == val.Kind() && "true" != defaultVal && "false" != defaultVal {
        defaultVal = "false"
    }

    // create arg for return
    return &ExpectedArg{
        field.Name,
        strings.Split(gliTag, ","),
        val,
        defaultVal,
        required,
        description,
        override,
    }
}


// basically this is in_array for the keys slice
func (arg *ExpectedArg) hasKey(key string) bool {
    for _, v := range arg.Keys {
        if key == v {
            return true
        }
    }

    return false
}
