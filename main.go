package main

import (
	"log/slog"
	"os"

	"github.com/hesusruiz/onboardng/mainapp"
)

func main() {

	// Invoke the real application, which can return an error
	if err := mainapp.Run(); err != nil {
		slog.Error("Error running application", "error", err)
		os.Exit(1)
	}

}
