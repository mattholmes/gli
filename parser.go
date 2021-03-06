package gli

import (
    "github.com/indeedhat/gli/util"
    "errors"
    "fmt"
    "reflect"
    "strconv"
    "strings"
)

type Parser struct {
    app      *App
    Raw      []string
    Expected []*ExpectedArg
    Valid    map[string]*ValidArg
}


// Parser constructor
func NewParser(app *App, args []string) (parser Parser) {
    parser = Parser{}
    parser.app = app

    // split args on first occurrence of =
    parser.Raw = func (args []string) (final []string) {
        for _, val := range args {
            i := strings.Index(val, "=")
            if i > 0 {
                final = append(final, val[:i])
                final = append(final, val[i + 1:])
            } else {
                final = append(final, val)
            }
        }
        return
    }(args)

    parser.extractExpectedArgs()
    parser.Valid = make(map[string]*ValidArg)

    return
}


// Main method of the parser runs the thing
func (parser *Parser) Parse() (err error) {
    override := false
    for i, c := 0, 0; i < len(parser.Raw); i++ {
        arg         := parser.Raw[i]
        value       := ""
        errorString := ""
        position    := -1
        key         := ""
        var expected *ExpectedArg

        if isntDashed(arg) {
            expected = parser.findExpected(arg, true)

            if nil != expected && parser.app.SelectCommand(expected.FieldName) {
                parser.extractExpectedArgs()
                continue
            }
        }

        if isDoubleDash(arg) || isSingleDash(arg) {
            errorString = fmt.Sprintf("Found unexpected arg %s", arg)
            offset, _   := util.IfElse(isDoubleDash(arg), 2, 1).(int)
            expected    = parser.findExpected(arg[offset:], false)

            // if it is not a flag then the next arg becomes the val
            if nil != expected {
                key   = arg[offset:]
                value = parser.nextValue(arg, i, -1)
                if util.CheckKind(expected.ArgType, reflect.Bool) {
                    if _, err = strconv.ParseBool(value); nil != err {
                        value = "true"
                    }
                } else if "" == value  {
                    errorString = fmt.Sprintf("No value given for arg %s", arg)
                    expected    = nil
                }

                if "" != value {
                    if nil != err {
                        err = nil
                    } else {
                        i++
                    }
                }
            }
        } else if isDashGroup(arg) {
            arg = arg[1:]

            // split into single dash flags
            for x := 0; x < len(arg); x++ {
                key         = string(arg[x])
                errorString = fmt.Sprintf("Found unexpected arg %s", arg[x])
                if expected = parser.findExpected(key, false); nil == expected {
                    break
                }

                value = parser.nextValue(arg, i, x)
                if util.CheckKind(expected.ArgType, reflect.Bool) {
                    if _, err := strconv.ParseBool(value); nil != err {
                        value = "true"
                    }
                }

                if arg[x + 1:] == value {
                    break
                } else if "" != value {
                    i++
                }
            }
        } else {
            errorString = fmt.Sprintf("Found unexpected positional arg '%s' at position %d", arg, c)
            expected    = parser.findPositionalExpected(c)
            value       = arg
            position    = c
            c++
        }

        if err = parser.checkAndAssign(value, key, errorString, expected, position); nil != err {
            return
        }

        if nil != expected && expected.Override {
            parser.Valid    =  map[string]*ValidArg {
                expected.FieldName: parser.Valid[expected.FieldName],
            }
            override = true
            break
        }
    }

    if false == override {
        parser.assignMissingDefaults()
        err = parser.validateRequiredArgs()
    }

    return
}


func (parser *Parser) nextValue(arg string, i, x int) string {
    if -1 != x && len(arg) -1 > x {
        return arg[x + 1:]
    } else if len(parser.Raw) -1 > i  {
        return parser.Raw[i + 1]
    }

    return ""
}

// extract all the expected arguments
func (parser *Parser) extractExpectedArgs() {
    parser.Expected = []*ExpectedArg{}

    t := reflect.TypeOf(parser.app.Subject.Cmd)
    v := reflect.ValueOf(parser.app.Subject.Cmd)
    for i := 0; i < v.Elem().NumField(); i++ {
        arg := newExpectedArg(
            t.Elem().Field(i),
            v.Elem().Field(i),
        )
        if nil == arg { continue }

        parser.Expected = append(
            parser.Expected,
            arg,
        )
    }
}


// post parse: find any missing args with a default value and add them to the valid list
func (parser *Parser) assignMissingDefaults() {
    for _, expected := range parser.Expected {
        if _, ok := parser.Valid[expected.FieldName]; !ok && "" != expected.DefaultVal {
            parser.addToValid(expected.DefaultVal, expected.Keys[0], expected, -1)
        }
    }
}


// post parse: check that all of the required arguments are found
func (parser *Parser) validateRequiredArgs() (err error) {
    for _, expected := range parser.Expected {
        if _, ok := parser.Valid[expected.FieldName]; !ok && expected.Required {
            if 0 == len(expected.Keys) {
                return errors.New("Missing required positional argument")
            } else {
                tpe := util.IfElse(util.CheckKind(expected.ArgType, reflect.Bool), "flag", "named argument")
                return errors.New(fmt.Sprintf("Missing required %s %s", tpe, expected.Keys[0]))
            }
        }
    }

    return
}


// check for an error
// assign valid arg if none is found
func (parser *Parser) checkAndAssign(value, key, errorString string, expected *ExpectedArg, offset int) (err error) {
    if nil == expected && !parser.unexpectedAllowed() {
        return errors.New(errorString)
    }

    parser.addToValid(value, key, expected, offset)
    return nil
}


// return an expected argument for the given raw argument
func (parser *Parser) findExpected(arg string, commandSearch bool) (expected *ExpectedArg) {
    for _, expected = range parser.Expected {
        if expected.hasKey(arg) && util.Xor(commandSearch, reflect.Struct != expected.ArgType.Kind()) {
            return
        }
    }

    return nil
}


// find an expected argument for the given positional raw argument
func (parser *Parser) findPositionalExpected(i int) (expected *ExpectedArg) {
    k := 0
    for _, expected = range parser.Expected {
        // skip none positional
        if 0 != len(expected.Keys) { continue }

        if k == i || util.IsSlice(expected.ArgType) {
            return
        }

        k++
    }

    return nil
}


// add argument to the valid args map
func (parser *Parser) addToValid(value, key string, expected *ExpectedArg, offset int) {
    if val, ok := parser.Valid[expected.FieldName]; ok {
        val.Value = append(val.Value, value)
    } else {
        parser.Valid[expected.FieldName] = newValidArg(expected, value, key, offset)
    }
}


// check if unexpected arguments are allowed by the command
func (parser *Parser) unexpectedAllowed() bool {
    unex, ok := parser.app.Subject.Cmd.(IgnoreUnexpected)
    return ok && unex.IgnoreUnexpected()
}


// check for double dash argument
func isDoubleDash(arg string) bool {
    return strings.HasPrefix(arg, "--")
}


// check for a single dash argument
func isSingleDash(arg string) bool {
    return 2 == len(arg) &&
        strings.HasPrefix(arg, "-")
}


// check if the argument is a group of single dash arguments
func isDashGroup(arg string) bool {
    return 2 < len(arg) &&
        strings.HasPrefix(arg, "-") &&
        !isDoubleDash(arg)
}


// check that the arg doesnt start with a -
func isntDashed(arg string) bool {
    return !strings.HasPrefix(arg, "-")
}
