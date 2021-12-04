package main

import (
	"github.com/euiko/tooyoul/mineman/pkg/app"

	"github.com/euiko/tooyoul/mineman/pkg/event"
	_ "github.com/euiko/tooyoul/mineman/pkg/event/channel"

	_ "github.com/euiko/tooyoul/mineman/modules/hello"
	_ "github.com/euiko/tooyoul/mineman/modules/network"
)

func main() {
	app := app.New("mineman", event.NewHook(), app.NewWebHook())
	if err := app.Run(); err != nil {
		println("error running app :", err)
		return
	}
}
