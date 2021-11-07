package main

import (
	"github.com/euiko/tooyoul/mineman/lib/app"
	_ "github.com/euiko/tooyoul/mineman/modules/hello"

	"github.com/euiko/tooyoul/mineman/lib/event"
	_ "github.com/euiko/tooyoul/mineman/lib/event/channel"
)

func main() {
	app := app.New("mineman", event.NewHook(), app.NewWebHook())
	if err := app.Run(); err != nil {
		println("error running app :", err)
		return
	}
}
