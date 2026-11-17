package main

//go:generate msgp -marshal=false -tests=false
//msgp:newtime
//msgp:limit arrays:1048576 maps:4096
//msgp:ignore App

import _ "embed"

// State is what the app keeps across a reload.
type State struct {
	Count int `msg:"count"`
}

// App is the state and its handlers.
type App struct {
	State
}

func (a *App) Save() *State     { return &a.State }
func (a *App) Restore(s *State) { a.State = *s }

func (a *App) Add() { a.Count++ }

// Key crashes the app on p, for the test of a crash.
func (a *App) Key(k string) bool {
	if k == "p" {
		panic("the test asked")
	}
	return false
}

// CSS is the app's stylesheet.
//
//go:embed style/app.css
var CSS string
