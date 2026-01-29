// Package main является точкой входа серверного приложения GophKeeper.
// Сервер обеспечивает безопасное хранение и синхронизацию приватных данных пользователей.
package main

import (
	"fmt"
	"os"

	"gophkeeper/internal/server/app"
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
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
