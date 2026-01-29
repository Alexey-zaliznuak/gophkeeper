// Package cmd содержит CLI команды клиента GophKeeper.
package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gophkeeper/internal/client/api"
	"gophkeeper/internal/client/config"
	"gophkeeper/internal/client/service"
	"gophkeeper/internal/client/storage/sqlite"
	"gophkeeper/internal/model"
)

// App содержит общие зависимости для всех команд.
type App struct {
	Config      *config.Config
	Storage     *sqlite.DB
	Client      *api.Client
	AuthService *service.AuthService
}

// NewRootCmd создаёт корневую команду CLI-клиента.
func NewRootCmd(version, buildDate string) *cobra.Command {
	var app *App

	rootCmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper - безопасное хранение приватных данных",
		Long: `GophKeeper - клиент-серверная система для безопасного хранения
логинов, паролей, бинарных данных и другой приватной информации.

Используйте подкоманды для работы с данными.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Пропускаем инициализацию для version
			if cmd.Name() == "version" {
				return nil
			}
			var err error
			app, err = initApp()
			return err
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return closeApp(app)
		},
	}

	// Добавление подкоманд
	rootCmd.AddCommand(newVersionCmd(version, buildDate))
	rootCmd.AddCommand(newRegisterCmd(&app))
	rootCmd.AddCommand(newLoginCmd(&app))
	rootCmd.AddCommand(newLogoutCmd(&app))
	rootCmd.AddCommand(newSyncCmd(&app))
	rootCmd.AddCommand(newCredentialsCmd(&app))
	rootCmd.AddCommand(newTextCmd(&app))
	rootCmd.AddCommand(newBinaryCmd(&app))
	rootCmd.AddCommand(newCardCmd(&app))

	return rootCmd
}

// initApp инициализирует приложение.
func initApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	storage, err := sqlite.New(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	client, err := api.NewClient(cfg.ServerAddress, cfg.TLSCertFile)
	if err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	app := &App{
		Config:      cfg,
		Storage:     storage,
		Client:      client,
		AuthService: service.NewAuthService(client, storage),
	}

	// Восстанавливаем токен из сохранённой сессии
	if session, err := storage.GetSession(); err == nil {
		client.SetAccessToken(session.AccessToken)
	}

	return app, nil
}

// closeApp закрывает ресурсы.
func closeApp(app *App) error {
	if app == nil {
		return nil
	}

	if app.Client != nil {
		app.Client.Close()
	}

	if app.Storage != nil {
		app.Storage.Close()
	}

	return nil
}

// readPassword читает пароль из терминала без отображения символов.
func readPassword() (string, error) {
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // перевод строки после ввода пароля
	return string(password), nil
}

// newVersionCmd создаёт команду для вывода версии.
func newVersionCmd(version, buildDate string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию и дату сборки",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("GophKeeper Client\n")
			cmd.Printf("Version:    %s\n", version)
			cmd.Printf("Build Date: %s\n", buildDate)
		},
	}
}

// newRegisterCmd создаёт команду регистрации.
func newRegisterCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Регистрация нового пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			var login string
			fmt.Print("Логин: ")
			fmt.Scanln(&login)

			fmt.Print("Пароль: ")
			password, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения пароля: %w", err)
			}

			fmt.Print("Мастер-пароль (для шифрования): ")
			masterPassword, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
			}

			if err := (*app).AuthService.Register(cmd.Context(), login, password, masterPassword); err != nil {
				return err
			}

			cmd.Println("Регистрация успешна!")
			return nil
		},
	}

	return cmd
}

// newLoginCmd создаёт команду входа.
func newLoginCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Вход в систему",
		RunE: func(cmd *cobra.Command, args []string) error {
			var login string
			fmt.Print("Логин: ")
			fmt.Scanln(&login)

			fmt.Print("Пароль: ")
			password, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения пароля: %w", err)
			}

			fmt.Print("Мастер-пароль: ")
			masterPassword, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
			}

			if err := (*app).AuthService.Login(cmd.Context(), login, password, masterPassword); err != nil {
				return err
			}

			cmd.Println("Вход выполнен!")

			// Автоматическая синхронизация после входа
			cmd.Println("Синхронизация данных...")

			encryptor, err := (*app).AuthService.GetEncryptor(masterPassword)
			if err != nil {
				return fmt.Errorf("ошибка инициализации шифрования: %w", err)
			}

			secretService := service.NewSecretService((*app).Client, (*app).Storage, encryptor)
			if err := secretService.Sync(cmd.Context()); err != nil {
				cmd.PrintErrf("Предупреждение: не удалось синхронизировать данные: %v\n", err)
				return nil // Не фейлим логин из-за ошибки синхронизации
			}

			cmd.Println("Синхронизация завершена!")
			return nil
		},
	}

	return cmd
}

// newLogoutCmd создаёт команду выхода.
func newLogoutCmd(app **App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Выход из системы",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := (*app).AuthService.Logout(cmd.Context()); err != nil {
				return err
			}
			cmd.Println("Выход выполнен!")
			return nil
		},
	}
}

// newSyncCmd создаёт команду синхронизации.
func newSyncCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Синхронизировать данные с сервером",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !(*app).AuthService.IsLoggedIn() {
				return fmt.Errorf("необходимо войти в систему")
			}

			fmt.Print("Мастер-пароль: ")
			masterPassword, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
			}

			encryptor, err := (*app).AuthService.GetEncryptor(masterPassword)
			if err != nil {
				return fmt.Errorf("неверный мастер-пароль: %w", err)
			}

			secretService := service.NewSecretService((*app).Client, (*app).Storage, encryptor)

			if err := secretService.Sync(cmd.Context()); err != nil {
				return err
			}

			cmd.Println("Синхронизация завершена!")
			return nil
		},
	}

	return cmd
}

// requireAuth проверяет авторизацию и возвращает SecretService.
func requireAuth(app *App, masterPassword string) (*service.SecretService, error) {
	if !app.AuthService.IsLoggedIn() {
		return nil, fmt.Errorf("необходимо войти в систему (gophkeeper login)")
	}

	encryptor, err := app.AuthService.GetEncryptor(masterPassword)
	if err != nil {
		return nil, fmt.Errorf("неверный мастер-пароль")
	}

	return service.NewSecretService(app.Client, app.Storage, encryptor), nil
}

// getMasterPassword запрашивает мастер-пароль интерактивно.
func getMasterPassword() string {
	fmt.Print("Мастер-пароль: ")
	password, err := readPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения пароля: %v\n", err)
		return ""
	}
	return password
}

// newCredentialsCmd создаёт группу команд для работы с логинами/паролями.
func newCredentialsCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credentials",
		Aliases: []string{"cred"},
		Short:   "Управление логинами и паролями",
	}

	// Add
	var addURL, addName string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить новую пару логин/пароль",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if addName == "" {
				fmt.Print("Название: ")
				fmt.Scanln(&addName)
			}

			var addLogin string
			fmt.Print("Логин: ")
			fmt.Scanln(&addLogin)

			fmt.Print("Пароль: ")
			addPassword, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения пароля: %w", err)
			}

			creds := &model.CredentialsData{
				Login:    addLogin,
				Password: addPassword,
				URL:      addURL,
			}

			if err := secretService.AddCredentials(cmd.Context(), addName, creds, nil); err != nil {
				return err
			}

			cmd.Println("Учётные данные сохранены!")
			return nil
		},
	}
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Название записи")
	addCmd.Flags().StringVarP(&addURL, "url", "u", "", "URL сайта")
	cmd.AddCommand(addCmd)

	// List
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список сохранённых учётных данных",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			secrets, err := secretService.ListByType(cmd.Context(), model.SecretTypeCredentials)
			if err != nil {
				return err
			}

			if len(secrets) == 0 {
				cmd.Println("Нет сохранённых учётных данных")
				return nil
			}

			cmd.Println("Учётные данные:")
			for _, s := range secrets {
				syncStatus := "✗"
				if s.IsSynced {
					syncStatus = "✓"
				}
				cmd.Printf("  [%s] %s\n", syncStatus, s.Name)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	// Get
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Получить учётные данные по имени",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			creds, metadata, err := secretService.GetCredentials(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			cmd.Printf("Название: %s\n", args[0])
			cmd.Printf("Логин:    %s\n", creds.Login)
			cmd.Printf("Пароль:   %s\n", creds.Password)
			if creds.URL != "" {
				cmd.Printf("URL:      %s\n", creds.URL)
			}
			if len(metadata) > 0 {
				cmd.Println("Метаданные:")
				for k, v := range metadata {
					cmd.Printf("  %s: %s\n", k, v)
				}
			}
			return nil
		},
	}
	cmd.AddCommand(getCmd)

	// Delete
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить учётные данные",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if err := secretService.Delete(cmd.Context(), args[0]); err != nil {
				return err
			}

			cmd.Println("Учётные данные удалены!")
			return nil
		},
	}
	cmd.AddCommand(deleteCmd)

	return cmd
}

// newTextCmd создаёт группу команд для работы с текстовыми данными.
func newTextCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "text",
		Short: "Управление текстовыми заметками",
	}

	// Add
	var addName, addContent string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить текстовую заметку",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if addName == "" {
				fmt.Print("Название: ")
				fmt.Scanln(&addName)
			}
			if addContent == "" {
				fmt.Print("Текст: ")
				fmt.Scanln(&addContent)
			}

			text := &model.TextData{Content: addContent}

			if err := secretService.AddText(cmd.Context(), addName, text, nil); err != nil {
				return err
			}

			cmd.Println("Заметка сохранена!")
			return nil
		},
	}
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Название")
	addCmd.Flags().StringVarP(&addContent, "content", "c", "", "Содержимое")
	cmd.AddCommand(addCmd)

	// List
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список заметок",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			secrets, err := secretService.ListByType(cmd.Context(), model.SecretTypeText)
			if err != nil {
				return err
			}

			if len(secrets) == 0 {
				cmd.Println("Нет сохранённых заметок")
				return nil
			}

			cmd.Println("Заметки:")
			for _, s := range secrets {
				syncStatus := "✗"
				if s.IsSynced {
					syncStatus = "✓"
				}
				cmd.Printf("  [%s] %s\n", syncStatus, s.Name)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	// Get
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Получить заметку по имени",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			text, _, err := secretService.GetText(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			cmd.Printf("Название: %s\n", args[0])
			cmd.Printf("Содержимое:\n%s\n", text.Content)
			return nil
		},
	}
	cmd.AddCommand(getCmd)

	// Delete
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить заметку",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if err := secretService.Delete(cmd.Context(), args[0]); err != nil {
				return err
			}

			cmd.Println("Заметка удалена!")
			return nil
		},
	}
	cmd.AddCommand(deleteCmd)

	return cmd
}

// newBinaryCmd создаёт группу команд для работы с бинарными данными.
func newBinaryCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "binary",
		Aliases: []string{"bin"},
		Short:   "Управление бинарными файлами",
	}

	// Add
	var addName string
	addCmd := &cobra.Command{
		Use:   "add [file]",
		Short: "Добавить бинарный файл",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			filePath := args[0]
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("не удалось прочитать файл: %w", err)
			}

			if addName == "" {
				addName = filePath
			}

			binary := &model.BinaryData{
				FileName: filePath,
				Data:     data,
			}

			if err := secretService.AddBinary(cmd.Context(), addName, binary, nil); err != nil {
				return err
			}

			cmd.Println("Файл сохранён!")
			return nil
		},
	}
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Название (по умолчанию имя файла)")
	cmd.AddCommand(addCmd)

	// List
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список файлов",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			secrets, err := secretService.ListByType(cmd.Context(), model.SecretTypeBinary)
			if err != nil {
				return err
			}

			if len(secrets) == 0 {
				cmd.Println("Нет сохранённых файлов")
				return nil
			}

			cmd.Println("Файлы:")
			for _, s := range secrets {
				syncStatus := "✗"
				if s.IsSynced {
					syncStatus = "✓"
				}
				cmd.Printf("  [%s] %s\n", syncStatus, s.Name)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	// Get
	var getOutput string
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Скачать файл",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			binary, _, err := secretService.GetBinary(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			outputPath := getOutput
			if outputPath == "" {
				outputPath = binary.FileName
			}

			if err := os.WriteFile(outputPath, binary.Data, 0600); err != nil {
				return fmt.Errorf("не удалось записать файл: %w", err)
			}

			cmd.Printf("Файл сохранён: %s\n", outputPath)
			return nil
		},
	}
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "", "Путь для сохранения")
	cmd.AddCommand(getCmd)

	// Delete
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить файл",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if err := secretService.Delete(cmd.Context(), args[0]); err != nil {
				return err
			}

			cmd.Println("Файл удалён!")
			return nil
		},
	}
	cmd.AddCommand(deleteCmd)

	return cmd
}

// newCardCmd создаёт группу команд для работы с банковскими картами.
func newCardCmd(app **App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Управление банковскими картами",
	}

	// Add
	var addName string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить банковскую карту",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if addName == "" {
				fmt.Print("Название: ")
				fmt.Scanln(&addName)
			}

			var addNumber string
			fmt.Print("Номер карты: ")
			fmt.Scanln(&addNumber)

			var addHolder string
			fmt.Print("Имя держателя: ")
			fmt.Scanln(&addHolder)

			var addMonth int
			fmt.Print("Месяц (MM): ")
			fmt.Scanln(&addMonth)

			var addYear int
			fmt.Print("Год (YYYY): ")
			fmt.Scanln(&addYear)

			fmt.Print("CVV: ")
			addCVV, err := readPassword()
			if err != nil {
				return fmt.Errorf("ошибка чтения CVV: %w", err)
			}

			card := &model.CardData{
				Number:      addNumber,
				HolderName:  addHolder,
				ExpiryMonth: addMonth,
				ExpiryYear:  addYear,
				CVV:         addCVV,
			}

			if err := secretService.AddCard(cmd.Context(), addName, card, nil); err != nil {
				return err
			}

			cmd.Println("Карта сохранена!")
			return nil
		},
	}
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Название")
	cmd.AddCommand(addCmd)

	// List
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список карт",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			secrets, err := secretService.ListByType(cmd.Context(), model.SecretTypeCard)
			if err != nil {
				return err
			}

			if len(secrets) == 0 {
				cmd.Println("Нет сохранённых карт")
				return nil
			}

			cmd.Println("Карты:")
			for _, s := range secrets {
				syncStatus := "✗"
				if s.IsSynced {
					syncStatus = "✓"
				}
				cmd.Printf("  [%s] %s\n", syncStatus, s.Name)
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	// Get
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Получить данные карты",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			card, _, err := secretService.GetCard(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			cmd.Printf("Название:    %s\n", args[0])
			cmd.Printf("Номер:       %s\n", card.Number)
			cmd.Printf("Держатель:   %s\n", card.HolderName)
			cmd.Printf("Срок:        %02d/%d\n", card.ExpiryMonth, card.ExpiryYear)
			cmd.Printf("CVV:         %s\n", card.CVV)
			return nil
		},
	}
	cmd.AddCommand(getCmd)

	// Delete
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить карту",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(*app, getMasterPassword())
			if err != nil {
				return err
			}

			if err := secretService.Delete(cmd.Context(), args[0]); err != nil {
				return err
			}

			cmd.Println("Карта удалена!")
			return nil
		},
	}
	cmd.AddCommand(deleteCmd)

	return cmd
}
