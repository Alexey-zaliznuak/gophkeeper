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

var app *App

// NewRootCmd создаёт корневую команду CLI-клиента.
func NewRootCmd(version, buildDate string) *cobra.Command {
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
			return initApp()
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return closeApp()
		},
	}

	// Добавление подкоманд
	rootCmd.AddCommand(newVersionCmd(version, buildDate))
	rootCmd.AddCommand(newRegisterCmd())
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newCredentialsCmd())
	rootCmd.AddCommand(newTextCmd())
	rootCmd.AddCommand(newBinaryCmd())
	rootCmd.AddCommand(newCardCmd())

	return rootCmd
}

// initApp инициализирует приложение.
func initApp() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	storage, err := sqlite.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to init storage: %w", err)
	}

	client, err := api.NewClient(cfg.ServerAddress, cfg.TLSCertFile)
	if err != nil {
		storage.Close()
		return fmt.Errorf("failed to create client: %w", err)
	}

	app = &App{
		Config:      cfg,
		Storage:     storage,
		Client:      client,
		AuthService: service.NewAuthService(client, storage),
	}

	// Восстанавливаем токен из сохранённой сессии
	if session, err := storage.GetSession(); err == nil {
		client.SetAccessToken(session.AccessToken)
	}

	return nil
}

// closeApp закрывает ресурсы.
func closeApp() error {
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
func newRegisterCmd() *cobra.Command {
	var login, password, masterPassword string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Регистрация нового пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" {
				fmt.Print("Логин: ")
				fmt.Scanln(&login)
			}
			if password == "" {
				fmt.Print("Пароль: ")
				var err error
				password, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения пароля: %w", err)
				}
			}
			if masterPassword == "" {
				fmt.Print("Мастер-пароль (для шифрования): ")
				var err error
				masterPassword, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
				}
			}

			if err := app.AuthService.Register(cmd.Context(), login, password, masterPassword); err != nil {
				return err
			}

			cmd.Println("Регистрация успешна!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "Логин пользователя")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Пароль")
	cmd.Flags().StringVarP(&masterPassword, "master", "m", "", "Мастер-пароль для шифрования")

	return cmd
}

// newLoginCmd создаёт команду входа.
func newLoginCmd() *cobra.Command {
	var login, password, masterPassword string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Вход в систему",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" {
				fmt.Print("Логин: ")
				fmt.Scanln(&login)
			}
			if password == "" {
				fmt.Print("Пароль: ")
				var err error
				password, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения пароля: %w", err)
				}
			}
			if masterPassword == "" {
				fmt.Print("Мастер-пароль: ")
				var err error
				masterPassword, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
				}
			}

			if err := app.AuthService.Login(cmd.Context(), login, password, masterPassword); err != nil {
				return err
			}

			cmd.Println("Вход выполнен!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&login, "login", "l", "", "Логин пользователя")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Пароль")
	cmd.Flags().StringVarP(&masterPassword, "master", "m", "", "Мастер-пароль")

	return cmd
}

// newLogoutCmd создаёт команду выхода.
func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Выход из системы",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.AuthService.Logout(cmd.Context()); err != nil {
				return err
			}
			cmd.Println("Выход выполнен!")
			return nil
		},
	}
}

// newSyncCmd создаёт команду синхронизации.
func newSyncCmd() *cobra.Command {
	var masterPassword string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Синхронизировать данные с сервером",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !app.AuthService.IsLoggedIn() {
				return fmt.Errorf("необходимо войти в систему")
			}

			if masterPassword == "" {
				fmt.Print("Мастер-пароль: ")
				var err error
				masterPassword, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
				}
			}

			encryptor, err := app.AuthService.GetEncryptor(masterPassword)
			if err != nil {
				return fmt.Errorf("неверный мастер-пароль: %w", err)
			}

			secretService := service.NewSecretService(app.Client, app.Storage, encryptor)

			if err := secretService.Sync(cmd.Context()); err != nil {
				return err
			}

			cmd.Println("Синхронизация завершена!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&masterPassword, "master", "m", "", "Мастер-пароль")

	return cmd
}

// requireAuth проверяет авторизацию и возвращает SecretService.
func requireAuth(masterPassword string) (*service.SecretService, error) {
	if !app.AuthService.IsLoggedIn() {
		return nil, fmt.Errorf("необходимо войти в систему (gophkeeper login)")
	}

	encryptor, err := app.AuthService.GetEncryptor(masterPassword)
	if err != nil {
		return nil, fmt.Errorf("неверный мастер-пароль")
	}

	return service.NewSecretService(app.Client, app.Storage, encryptor), nil
}

// getMasterPassword запрашивает мастер-пароль.
func getMasterPassword(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	fmt.Print("Мастер-пароль: ")
	password, err := readPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения пароля: %v\n", err)
		return ""
	}
	return password
}

// newCredentialsCmd создаёт группу команд для работы с логинами/паролями.
func newCredentialsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credentials",
		Aliases: []string{"cred"},
		Short:   "Управление логинами и паролями",
	}

	// Add
	var addLogin, addPassword, addURL, addMaster, addName string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить новую пару логин/пароль",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(addMaster))
			if err != nil {
				return err
			}

			if addName == "" {
				fmt.Print("Название: ")
				fmt.Scanln(&addName)
			}
			if addLogin == "" {
				fmt.Print("Логин: ")
				fmt.Scanln(&addLogin)
			}
			if addPassword == "" {
				fmt.Print("Пароль: ")
				var err error
				addPassword, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения пароля: %w", err)
				}
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
	addCmd.Flags().StringVarP(&addLogin, "login", "l", "", "Логин")
	addCmd.Flags().StringVarP(&addPassword, "password", "p", "", "Пароль")
	addCmd.Flags().StringVarP(&addURL, "url", "u", "", "URL сайта")
	addCmd.Flags().StringVarP(&addMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(addCmd)

	// List
	var listMaster string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список сохранённых учётных данных",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(listMaster))
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
	listCmd.Flags().StringVarP(&listMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(listCmd)

	// Get
	var getMaster string
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Получить учётные данные по имени",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(getMaster))
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
	getCmd.Flags().StringVarP(&getMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(getCmd)

	// Delete
	var deleteMaster string
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить учётные данные",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(deleteMaster))
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
	deleteCmd.Flags().StringVarP(&deleteMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(deleteCmd)

	return cmd
}

// newTextCmd создаёт группу команд для работы с текстовыми данными.
func newTextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "text",
		Short: "Управление текстовыми заметками",
	}

	// Add
	var addMaster, addName, addContent string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить текстовую заметку",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(addMaster))
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
	addCmd.Flags().StringVarP(&addMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(addCmd)

	// List
	var listMaster string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список заметок",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(listMaster))
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
	listCmd.Flags().StringVarP(&listMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(listCmd)

	// Get
	var getMaster string
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Получить заметку по имени",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(getMaster))
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
	getCmd.Flags().StringVarP(&getMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(getCmd)

	// Delete
	var deleteMaster string
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить заметку",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(deleteMaster))
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
	deleteCmd.Flags().StringVarP(&deleteMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(deleteCmd)

	return cmd
}

// newBinaryCmd создаёт группу команд для работы с бинарными данными.
func newBinaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "binary",
		Aliases: []string{"bin"},
		Short:   "Управление бинарными файлами",
	}

	// Add
	var addMaster, addName string
	addCmd := &cobra.Command{
		Use:   "add [file]",
		Short: "Добавить бинарный файл",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(addMaster))
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
	addCmd.Flags().StringVarP(&addMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(addCmd)

	// List
	var listMaster string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список файлов",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(listMaster))
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
	listCmd.Flags().StringVarP(&listMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(listCmd)

	// Get
	var getMaster, getOutput string
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Скачать файл",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(getMaster))
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
	getCmd.Flags().StringVarP(&getMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(getCmd)

	// Delete
	var deleteMaster string
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить файл",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(deleteMaster))
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
	deleteCmd.Flags().StringVarP(&deleteMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(deleteCmd)

	return cmd
}

// newCardCmd создаёт группу команд для работы с банковскими картами.
func newCardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Управление банковскими картами",
	}

	// Add
	var addMaster, addName, addNumber, addHolder, addCVV string
	var addMonth, addYear int
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить банковскую карту",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(addMaster))
			if err != nil {
				return err
			}

			if addName == "" {
				fmt.Print("Название: ")
				fmt.Scanln(&addName)
			}
			if addNumber == "" {
				fmt.Print("Номер карты: ")
				fmt.Scanln(&addNumber)
			}
			if addHolder == "" {
				fmt.Print("Имя держателя: ")
				fmt.Scanln(&addHolder)
			}
			if addMonth == 0 {
				fmt.Print("Месяц (MM): ")
				fmt.Scanln(&addMonth)
			}
			if addYear == 0 {
				fmt.Print("Год (YYYY): ")
				fmt.Scanln(&addYear)
			}
			if addCVV == "" {
				fmt.Print("CVV: ")
				var err error
				addCVV, err = readPassword()
				if err != nil {
					return fmt.Errorf("ошибка чтения CVV: %w", err)
				}
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
	addCmd.Flags().StringVar(&addNumber, "number", "", "Номер карты")
	addCmd.Flags().StringVar(&addHolder, "holder", "", "Имя держателя")
	addCmd.Flags().IntVar(&addMonth, "month", 0, "Месяц окончания")
	addCmd.Flags().IntVar(&addYear, "year", 0, "Год окончания")
	addCmd.Flags().StringVar(&addCVV, "cvv", "", "CVV код")
	addCmd.Flags().StringVarP(&addMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(addCmd)

	// List
	var listMaster string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Показать список карт",
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(listMaster))
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
	listCmd.Flags().StringVarP(&listMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(listCmd)

	// Get
	var getMaster string
	getCmd := &cobra.Command{
		Use:   "get [name]",
		Short: "Получить данные карты",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(getMaster))
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
	getCmd.Flags().StringVarP(&getMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(getCmd)

	// Delete
	var deleteMaster string
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Удалить карту",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretService, err := requireAuth(getMasterPassword(deleteMaster))
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
	deleteCmd.Flags().StringVarP(&deleteMaster, "master", "m", "", "Мастер-пароль")
	cmd.AddCommand(deleteCmd)

	return cmd
}
