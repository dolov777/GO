package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "unicode/utf8"

    "github.com/joho/godotenv"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Config struct {
    BotToken    string
    LecturesDir string
}

func LoadConfig() (Config, error) {
    err := godotenv.Load()
    if err != nil {
        log.Println("Ошибка загрузки файла .env, используем переменные окружения по умолчанию.")
    }

    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    if token == "" {
        return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN не найден в окружении")
    }

    lecturesDir := os.Getenv("LECTURES_DIR")
    if lecturesDir == "" {
        lecturesDir = "lectures"
        log.Println("LECTURES_DIR не найден в окружении, используем значение по умолчанию: 'lectures'")
    }

    return Config{
        BotToken:    token,
        LecturesDir: lecturesDir,
    }, nil
}

var userClass = make(map[int64]string)

type LectureInfo struct {
    Class      string
    LectureNum int
}

var userLecturesInfo = make(map[int64]LectureInfo)

func main() {
    config, err := LoadConfig()
    if err != nil {
        log.Fatalf("Ошибка конфигурации: %v", err)
    }

    bot, err := tgbotapi.NewBotAPI(config.BotToken)
    if err != nil {
        log.Fatalf("Ошибка инициализации бота: %v", err)
    }

    webhookURL := os.Getenv("WEBHOOK_URL")
    if webhookURL == "" {
        log.Fatal("WEBHOOK_URL не найден в переменных окружения")
    }

    webhookConfig, err := tgbotapi.NewWebhook(webhookURL)
    if err != nil {
        log.Fatalf("Ошибка при создании Webhook: %v", err)
    }

    _, err = bot.Request(webhookConfig)
    if err != nil {
        log.Fatalf("Ошибка при отправке запроса Webhook: %v", err)
    }

    info, err := bot.GetWebhookInfo()
    if err != nil {
        log.Fatalf("Ошибка при получении информации о Webhook: %v", err)
    }
    if info.LastErrorDate != 0 {
        log.Printf("Телеграм сообщает об ошибке вебхука: %s", info.LastErrorMessage)
    }

    http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
        update, err := bot.HandleUpdate(r)
        if err != nil {
            log.Printf("Ошибка обработки обновления: %v", err)
            return
        }
        if update != nil {
            if update.Message != nil {
                handleMessage(bot, *update, config)
            } else if update.CallbackQuery != nil {
                handleCallback(bot, *update, config)
            }
        }
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8443"
    }

    log.Printf("Сервер запущен на порту %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, config Config) {
    command := update.Message.Text
    var responseText string

    switch command {
    case "/start":
        responseText = "Привет! Я бот, который выдает конспекты по информатике. Выберите класс:"
        keyboard := buildClassKeyboard()
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, responseText)
        msg.ReplyMarkup = keyboard
        _, err := bot.Send(msg)
        if err != nil {
            log.Println(err)
        }
    
    case "7 Класс", "8 Класс", "9 Класс", "10 Класс":
        userClass[update.Message.Chat.ID] = command
        lecturesDir := filepath.Join(config.LecturesDir, command)

        if _, err := os.Stat(lecturesDir); os.IsNotExist(err) {
            msgText := fmt.Sprintf("Конспекты для %sа пока не доступны.", command)
            msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
            bot.Send(msg)
            return
        }

        keyboard := buildLectureKeyboard(lecturesDir, config)
        msgText := fmt.Sprintf("Выберите конспект для %sа:", command)
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
        msg.ReplyMarkup = keyboard
        _, err := bot.Send(msg)
        if err != nil {
            log.Println(err)
        }

    default:
        responseText = "Я не понимаю эту команду. Попробуйте /start."
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, responseText)
        _, err := bot.Send(msg)
        if err != nil {
            log.Println(err)
        }
    }
}

func buildClassKeyboard() tgbotapi.ReplyKeyboardMarkup {
    return tgbotapi.NewReplyKeyboard(
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("7 Класс"),
            tgbotapi.NewKeyboardButton("8 Класс"),
        ),
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("9 Класс"),
            tgbotapi.NewKeyboardButton("10 Класс"),
        ),
    )
}

func buildLectureKeyboard(lecturesDir string, config Config) tgbotapi.InlineKeyboardMarkup {
    var keyboard [][]tgbotapi.InlineKeyboardButton

    files, err := os.ReadDir(lecturesDir)
    if err != nil {
        log.Printf("Ошибка при чтении директории с конспектами: %s", err)
        return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
    }

    for i, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
            buttonText := strings.TrimSuffix(file.Name(), ".txt")
            const maxButtonLength = 64
            if utf8.RuneCountInString(buttonText) > maxButtonLength {
                buttonText = string([]rune(buttonText)[:maxButtonLength-3]) + "..."
            }
            relativePath, err := filepath.Rel(config.LecturesDir, filepath.Join(lecturesDir, file.Name()))
            if err != nil {
                log.Printf("Ошибка при получении относительного пути: %s", err)
                continue
            }
            btn := tgbotapi.NewInlineKeyboardButtonData(buttonText, "lecture_"+relativePath+"_"+fmt.Sprint(i))
            keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{btn})
        }
    }

    return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

func handleLectureSelection(bot *tgbotapi.BotAPI, chatID int64, relativePath string, class string, lectureNum int, config Config) {
    fullLecturePath := filepath.Join(config.LecturesDir, relativePath)

    if _, err := os.Stat(fullLecturePath); os.IsNotExist(err) {
        log.Printf("Файл лекции не найден: %s", fullLecturePath)
        msg := tgbotapi.NewMessage(chatID, "Не удалось загрузить лекцию. Попробуйте позже.")
        bot.Send(msg)
        return
    }

    file := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(fullLecturePath))
    _, err := bot.Send(file)
    if err != nil {
        log.Printf("Ошибка при отправке лекции: %s", err)
        msg := tgbotapi.NewMessage(chatID, "Не удалось отправить лекцию. Попробуйте позже.")
        bot.Send(msg)
        return
    }

    userLecturesInfo[chatID] = LectureInfo{Class: class, LectureNum: lectureNum}
    sendLectureNavigation(bot, chatID, class, lectureNum, getTotalLecturesForClass(class, config))
}

func getTotalLecturesForClass(class string, config Config) int {
    lecturesDir := filepath.Join(config.LecturesDir, class)
    files, err := os.ReadDir(lecturesDir)
    if err != nil {
        log.Printf("Ошибка при чтении директории с конспектами: %s", err)
        return 0
    }
    count := 0
    for _, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
            count++
        }
    }
    return count
}

func sendLectureNavigation(bot *tgbotapi.BotAPI, chatID int64, class string, lectureNum int, totalLectures int) {
    var buttons []tgbotapi.InlineKeyboardButton
    if lectureNum > 0 {
        buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("prev_%s_%d", class, lectureNum-1)))
    }
    if lectureNum < totalLectures-1 {
        buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Вперед", fmt.Sprintf("next_%s_%d", class, lectureNum+1)))
    }

    keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
    msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
    msg.ReplyMarkup = keyboard
    bot.Send(msg)
}

func handleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update, config Config) {
    callbackData := update.CallbackQuery.Data
    chatID := update.CallbackQuery.Message.Chat.ID

    log.Printf("Обрабатывается колбэк: %s", callbackData)

    if strings.HasPrefix(callbackData, "lecture_") {
        parts := strings.Split(callbackData, "_")
        relativePath := parts[1]
        class := userClass[chatID]
        lectureNum := 0
        if len(parts) > 2 {
            fmt.Sscanf(parts[2], "%d", &lectureNum)
        }
        handleLectureSelection(bot, chatID, relativePath, class, lectureNum, config)
    } else if strings.HasPrefix(callbackData, "prev_") || strings.HasPrefix(callbackData, "next_") {
        parts := strings.Split(callbackData, "_")
        class := parts[1]
        lectureNum := 0
        fmt.Sscanf(parts[2], "%d", &lectureNum)

        relativePath := getLecturePathForClass(class, lectureNum, config)
        handleLectureSelection(bot, chatID, relativePath, class, lectureNum, config)
    }

    callbackMsg := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
    bot.Send(callbackMsg)
}

func getLecturePathForClass(class string, lectureNum int, config Config) string {
    lecturesDir := filepath.Join(config.LecturesDir, class)
    files, err := os.ReadDir(lecturesDir)
    if err != nil {
        log.Printf("Ошибка при чтении директории с конспектами: %s", err)
        return ""
    }
    if lectureNum < 0 || lectureNum >= len(files) {
        return ""
    }
    return filepath.Join(class, files[lectureNum].Name())
}