package main

import "github.com/euiko/tooyoul/mineman/lib/app"

func main() {
	app := app.New("mineman", app.NewWebHook())
	if err := app.Run(); err != nil {
		println("error running app :", err)
		return
	}
}
