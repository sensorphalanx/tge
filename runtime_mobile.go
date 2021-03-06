// Copyright (c) 2019 Thomas MILLET. All rights reserved.
// Copyright 2014 The Go Authors.  All rights reserved.

// +build android ios

package tge

import (
	fmt "fmt"
	ioutil "io/ioutil"
	time "time"

	mobile "github.com/thommil/tge-mobile/app"
	asset "github.com/thommil/tge-mobile/asset"
	lifecycle "github.com/thommil/tge-mobile/event/lifecycle"
	paint "github.com/thommil/tge-mobile/event/paint"
	size "github.com/thommil/tge-mobile/event/size"
	touch "github.com/thommil/tge-mobile/event/touch"
	gl "github.com/thommil/tge-mobile/gl"
)

func init() {
	_runtimeInstance = &mobileRuntime{}
}

// -------------------------------------------------------------------- //
// Runtime implementation
// -------------------------------------------------------------------- //
type mobileRuntime struct {
	app       App
	host      mobile.App
	context   gl.Context
	settings  Settings
	isPaused  bool
	isStopped bool
}

func (runtime *mobileRuntime) GetAsset(p string) ([]byte, error) {
	if file, err := asset.Open(p); err != nil {
		return nil, err
	} else {
		return ioutil.ReadAll(file)
	}
}

func (runtime *mobileRuntime) GetHost() interface{} {
	return runtime.host
}

func (runtime *mobileRuntime) GetRenderer() interface{} {
	return runtime.context
}

func (runtime *mobileRuntime) GetSettings() Settings {
	return runtime.settings
}

func (runtime *mobileRuntime) Subscribe(channel string, listener Listener) {
	subscribe(channel, listener)
}

func (runtime *mobileRuntime) Unsubscribe(channel string, listener Listener) {
	unsubscribe(channel, listener)
}

func (runtime *mobileRuntime) Publish(event Event) {
	publish(event)
}

func (runtime *mobileRuntime) Stop() {
	// Not implemented
}

// Run main entry point of runtime
func Run(app App) error {
	// -------------------------------------------------------------------- //
	// Create
	// -------------------------------------------------------------------- //
	settings := defaultSettings
	err := app.OnCreate(&settings)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer app.OnDispose()

	// Instanciate Runtime
	mobileRuntime := _runtimeInstance.(*mobileRuntime)
	mobileRuntime.app = app
	mobileRuntime.settings = settings
	mobileRuntime.isPaused = true
	mobileRuntime.isStopped = true
	defer dispose()

	// -------------------------------------------------------------------- //
	// Ticker Loop
	// -------------------------------------------------------------------- //
	syncChan := make(chan interface{})
	startTicker := func() {
		elapsedTpsTime := time.Duration(0)
		for !mobileRuntime.isStopped {
			if !mobileRuntime.isPaused {
				now := time.Now()
				app.OnTick(elapsedTpsTime, syncChan)
				elapsedTpsTime = time.Since(now)
			}
		}
	}

	// -------------------------------------------------------------------- //
	// Init
	// -------------------------------------------------------------------- //
	var moveEvtChan chan MouseEvent
	elapsedFpsTime := time.Duration(0)
	mobile.Main(func(a mobile.App) {
		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.To {
				case lifecycle.StageFocused:
					mobileRuntime.context, _ = e.DrawContext.(gl.Context)
					mobileRuntime.host = a

					// Init plugins
					initPlugins()

					err := app.OnStart(mobileRuntime)
					if err != nil {
						fmt.Println(err)
						panic(err)
					}
					mobileRuntime.isStopped = false
					go startTicker()
					app.OnResume()
					mobileRuntime.isPaused = false

					// Mouse motion hack to queue move events
					moveEvtChan = make(chan MouseEvent, 100)
					go func() {
						for !mobileRuntime.isStopped {
							publish(<-moveEvtChan)
						}
					}()

				case lifecycle.StageAlive:
					mobileRuntime.isPaused = true
					app.OnPause()
					mobileRuntime.isStopped = true
					close(moveEvtChan)
					app.OnStop()
					mobileRuntime.context = nil

					// Release plugins
					dispose()
				}

			case paint.Event:
				if !mobileRuntime.isPaused {
					if mobileRuntime.context != nil && !e.External {
						now := time.Now()
						app.OnRender(elapsedFpsTime, syncChan)
						a.Publish()
						elapsedFpsTime = time.Since(now)
					}
					a.Send(paint.Event{})
				}

			case size.Event:
				publish(ResizeEvent{int32(e.WidthPx), int32(e.HeightPx)})

			case touch.Event:
				button := ButtonNone
				switch e.Sequence {
				case 0:
					button = TouchFirst
				case 1:
					button = TouchSecond
				case 2:
					button = TouchThird
				}
				switch e.Type {
				case touch.TypeBegin:
					// mouse down
					if (settings.EventMask & MouseButtonEventEnabled) != 0 {
						moveEvtChan <- MouseEvent{
							X:      int32(e.X),
							Y:      int32(e.Y),
							Type:   TypeDown,
							Button: button,
						}
					}
				case touch.TypeMove:
					// mouse move
					if (settings.EventMask & MouseMotionEventEnabled) != 0 {
						moveEvtChan <- MouseEvent{
							X:      int32(e.X),
							Y:      int32(e.Y),
							Type:   TypeMove,
							Button: button,
						}
					}
				case touch.TypeEnd:
					// Touch down
					if (settings.EventMask & MouseButtonEventEnabled) != 0 {
						moveEvtChan <- MouseEvent{
							X:      int32(e.X),
							Y:      int32(e.Y),
							Type:   TypeUp,
							Button: button,
						}
					}
				}
			}

		}
	})

	return nil
}
