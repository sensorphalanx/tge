package main

import (
	"sync"
	"time"

	tge "github.com/thommil/tge"
)

type App struct {
}

func (app *App) OnCreate(settings *tge.Settings) error {
	return nil
}

func (app *App) OnStart(runtime tge.Runtime) error {
	return nil
}

func (app *App) OnResume() {
}

func (app *App) OnRender(elapsedTime time.Duration, mutex *sync.Mutex) {
}

func (app *App) OnTick(elapsedTime time.Duration, mutex *sync.Mutex) {
}

func (app *App) OnPause() {
}

func (app *App) OnStop() {
}

func (app *App) OnDispose() error {
	return nil
}

func main() {
	tge.Run(&App{})
}
