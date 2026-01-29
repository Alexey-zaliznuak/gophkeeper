// Package app содержит логику инициализации и запуска CLI-клиента.
package app

import (
	"fmt"

	"gophkeeper/internal/client/cmd"
)

// Run запускает CLI-клиент GophKeeper.
// Инициализирует конфигурацию и запускает корневую команду Cobra.
func Run(version, buildDate string) error {
	rootCmd := cmd.NewRootCmd(version, buildDate)

	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}
