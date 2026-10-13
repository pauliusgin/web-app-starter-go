// Command server runs the web_app_starter HTTP API.
package main

import (
	"log/slog"
	"os"
)

func main() {
	if err := runServer(serverStartupSteps()); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
