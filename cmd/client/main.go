// Package main является точкой входа CLI-клиента GophKeeper.
// Клиент позволяет пользователям безопасно хранить и получать доступ к приватным данным.
package main

import (
	"fmt"
	"os"

	"gophkeeper/internal/client/app"
)

// Переменные для версионирования, устанавливаются при сборке через ldflags.
var (
	// Version содержит версию приложения.
	Version = "dev"
	// BuildDate содержит дату сборки приложения.
	BuildDate = "unknown"
)

func main() {
	if err := app.Run(Version, BuildDate); err != nil {
		fmt.Fprintf(os.Stderr, "client error: %v\n", err)
		os.Exit(1)
	}
}
