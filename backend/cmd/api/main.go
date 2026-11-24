package main

import (
	"fmt"

	"github.com/chaitin/panda-wiki/setup"
	"github.com/chaitin/panda-wiki/telemetry"
)

// @title panda-wiki API
// @version 1.0
// @description panda-wiki API documentation
// @BasePath /
// @securityDefinitions.apikey	bearerAuth
// @in	header
// @name	Authorization
// @description	Type "Bearer" + a space + your token to authorize
func main() {
	app, err := createApp()
	if err != nil {
		panic(err)
	}
	if err := setup.CheckInitCert(); err != nil {
		panic(err)
	}
	port := app.Config.HTTP.Port
	app.Logger.Info(fmt.Sprintf("Starting server version %s on port %d", telemetry.Version, port))
	app.HTTPServer.Echo.Logger.Fatal(app.HTTPServer.Echo.Start(fmt.Sprintf(":%d", port)))
}
