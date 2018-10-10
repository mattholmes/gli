package main

import "fmt"

type Application struct {
    SubCommand SubCommand `gli:"sub,s,subcommand" description:"does some sub command things"`
    Flag       bool       `gli:"flag,f"`
    BoolTrue   bool       `gli:"b,bt" default:"true"`
    BoolArray  []bool     `gli:"ba"`
    String     string     `gli:"string,s,S"`
    Help       bool       `gli:"help,h,H"`
}


func (app *Application) Run() int {
    fmt.Println("Main application command")
    fmt.Printf("Flag: %v\n", app.Flag)
    fmt.Printf("String: %v\n", app.String)
    fmt.Printf("BoolTrue: %v\n", app.BoolTrue)
    fmt.Printf("BoolArray: %v\n", app.BoolArray)

    return 0
}