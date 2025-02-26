package main

import (
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "unicode/utf8"

    "github.com/joho/godotenv"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Конфигурация
type Config struct {
    BotToken    string
    LecturesDir string
}

// Функция для загрузки конфигурации из переменных окружения
func LoadConfig() (Config, error) {
    err := godotenv.Load()
    if err != nil {
        log.Println("Ошибка загрузки файла .env, используем переменные окружения по умолчанию.")
    }

    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    if token == "" {
        log.Println("TELEGRAM_BOT_TOKEN not found in environment, using default. The bot will likely fail to start.")
        return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN not found")
    }

    lecturesDir := os.Getenv("LECTURES_DIR")
    if lecturesDir == "" {
        lecturesDir = "lectures"
        log.Println("LECTURES_DIR not found in environment, using default: 'lectures'")
    }

    return Config{
        BotToken:    token,
        LecturesDir: lecturesDir,
    }, nil
}

var userClass = make(map[int64]string)         
var userCurrentLecture = make(map[int64]string) 
func main() {
    config, err := LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %s", err)
    }

    if len(config.BotToken) < 30 {
        log.Fatalf("Invalid Telegram Bot Token: '%s'. Please check your environment variable.", config.BotToken)
    }

    bot, err := tgbotapi.NewBotAPI(config.BotToken)
    if err != nil {
        log.Fatalf("Error initializing bot: %s. Check your TELEGRAM_BOT_TOKEN.", err)
    }

    bot.Debug = true

    log.Printf("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            if update.CallbackQuery != nil {
                // Handle callback query (button press)
                callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
                if _, err := bot.Request(callback); err != nil {
                    panic(err)
                }
                handleCallback(bot, &update, config)
            }
            continue
        }

        log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

        command := update.Message.Text
        var responseText string

        if command == "/start" {
            responseText = "Привет! Я бот, который выдает конспекты по информатике. Выберите класс:"
            keyboard := buildClassKeyboard()
            msg := tgbotapi.NewMessage(update.Message.Chat.ID, responseText)
            msg.ReplyMarkup = keyboard
            _, err := bot.Send(msg)
            if err != nil {
                log.Println(err)
            }
            continue
        } else {
            responseText = "Я не понимаю эту команду. Попробуйте /start."
        }

        msg := tgbotapi.NewMessage(update.Message.Chat.ID, responseText)
        _, err = bot.Send(msg)
        if err != nil {
            log.Println(err)
        }
    }
}

func buildClassKeyboard() tgbotapi.InlineKeyboardMarkup {
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("7 Класс", "class_7"),
            tgbotapi.NewInlineKeyboardButtonData("8 Класс", "class_8"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("9 Класс", "class_9"),
            tgbotapi.NewInlineKeyboardButtonData("10 Класс", "class_10"),
        ),
    )
    return keyboard
}

func buildLectureKeyboard(lecturesDir string, config Config) tgbotapi.InlineKeyboardMarkup {
    var keyboard [][]tgbotapi.InlineKeyboardButton

    files, err := os.ReadDir(lecturesDir)
    if err != nil {
        log.Printf("Ошибка при чтении директории с конспектами: %s", err)
        return tgbotapi.InlineKeyboardMarkup{} // Возвращаем пустую клавиатуру
    }

    row := []tgbotapi.InlineKeyboardButton{}
    for _, file := range files {
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

            btn := tgbotapi.NewInlineKeyboardButtonData(buttonText, "lecture_"+relativePath)
            row = append(row, btn)

            if len(row) == 2 {
                keyboard = append(keyboard, row)
                row = []tgbotapi.InlineKeyboardButton{}
            }
        }
    }

    if len(row) > 0 {
        keyboard = append(keyboard, row)
    }

    return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

func handleCallback(bot *tgbotapi.BotAPI, update *tgbotapi.Update, config Config) {
    callbackData := update.CallbackQuery.Data
    chatID := update.CallbackQuery.Message.Chat.ID

    if strings.HasPrefix(callbackData, "class_") {
        class := strings.TrimPrefix(callbackData, "class_")
        userClass[chatID] = class 

        lecturesDir := filepath.Join(config.LecturesDir, class)

        if _, err := os.Stat(lecturesDir); os.IsNotExist(err) {
            msgText := fmt.Sprintf("Конспекты для %s класса пока не доступны.", class)
            msg := tgbotapi.NewMessage(chatID, msgText)
            bot.Send(msg)
            return
        }

        keyboard := buildLectureKeyboard(lecturesDir, config)
        msgText := fmt.Sprintf("Выберите конспект для %s класса:", class)
        msg := tgbotapi.NewMessage(chatID, msgText)
        msg.ReplyMarkup = keyboard
        _, err := bot.Send(msg)
        if err != nil {
            log.Println(err)
        }
    } else if strings.HasPrefix(callbackData, "lecture_") {
        lecturePath := strings.TrimPrefix(callbackData, "lecture_")
        fullLecturePath := filepath.Join(config.LecturesDir, lecturePath)
        userCurrentLecture[chatID] = fullLecturePath 

        content, err := readFileContent(fullLecturePath)
        if err != nil {
            log.Printf("Ошибка при чтении файла лекции: %s", err)
            msgText := "Не удалось загрузить лекцию. Попробуйте позже."
            msg := tgbotapi.NewMessage(chatID, msgText)
            bot.Send(msg)
            return
        }

        if len(content) == 0 {
            msgText := "Мы еще не прошли эту тему."
            msg := tgbotapi.NewMessage(chatID, msgText)
            _, err = bot.Send(msg)
            if err != nil {
                log.Printf("Ошибка при отправке сообщения: %s", err)
            }
            askForNextAction(bot, chatID, false) // Здесь флаг, чтобы не показывать кнопку классов
            return
        }

        // Отправляем содержание лекции по частям
        sendMessageInParts(bot, chatID, content)

        askForNextAction(bot, chatID, true) // Здесь класс будет показываться

    } else if callbackData == "next_lecture" {
        nextLecture(bot, config, chatID)
    } else if callbackData == "prev_lecture" { // Обработка кнопки "Предыдущий конспект"
        prevLecture(bot, config, chatID)
    } else if callbackData == "show_classes" { // Обработка кнопки "Классы"
        showDataClasses(bot, chatID)
    } else if callbackData == "show_lecture_list" { // Обработка кнопки "Список конспектов"
        showLectureList(bot, config, chatID)
    }
}

// Функция для показа кнопок с классами
func showDataClasses(bot *tgbotapi.BotAPI, chatID int64) {
    responseText := "Выберите класс:"
    keyboard := buildClassKeyboard()
    msg := tgbotapi.NewMessage(chatID, responseText)
    msg.ReplyMarkup = keyboard
    if _, err := bot.Send(msg); err != nil {
        log.Println(err)
    }
}

func buildNextLectureKeyboard(showClasses bool) tgbotapi.InlineKeyboardMarkup {
    var buttons []tgbotapi.InlineKeyboardButton
    buttons = append(buttons,
        tgbotapi.NewInlineKeyboardButtonData("Предыдущий конспект", "prev_lecture"),
        tgbotapi.NewInlineKeyboardButtonData("Следующий конспект", "next_lecture"),
        tgbotapi.NewInlineKeyboardButtonData("Список конспектов", "show_lecture_list"), // Добавляем кнопку "Список конспектов"
    )
    if showClasses {
        buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Классы", "show_classes")) // Добавляем кнопку "Классы"
    }

    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(buttons...),
    )

    return keyboard
}

func readFileContent(filename string) (string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer file.Close()

    content, err := io.ReadAll(file)
    if err != nil {
        return "", err
    }

    return string(content), nil
}

// Функция отправки длинного сообщения по частям
func sendMessageInParts(bot *tgbotapi.BotAPI, chatID int64, content string) {
    const maxMessageLength = 4096
    for start := 0; start < len(content); start += maxMessageLength {
        end := start + maxMessageLength
        if end > len(content) {
            end = len(content)
        }
        msgText := content[start:end]
        msg := tgbotapi.NewMessage(chatID, msgText)
        _, err := bot.Send(msg)
        if err != nil {
            log.Printf("Ошибка при отправке сообщения: %s", err)
        }
    }
}

func askForNextAction(bot *tgbotapi.BotAPI, chatID int64, showClasses bool) {
    // Учитываем, нужно ли показывать кнопку классов
    nextLectureKeyboard := buildNextLectureKeyboard(showClasses)
    nextMsgText := "Хотите следующий конспект или посмотреть список конспектов?"
    nextMsg := tgbotapi.NewMessage(chatID, nextMsgText)
    nextMsg.ReplyMarkup = nextLectureKeyboard
    _, err := bot.Send(nextMsg)
    if err != nil {
        log.Printf("Ошибка при отправке сообщения: %s", err)
    }
}

func nextLecture(bot *tgbotapi.BotAPI, config Config, chatID int64) {
    class, ok := userClass[chatID]
    if !ok {
        msgText := "Пожалуйста, выберите класс сначала (/start)."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    currentLecturePath, ok := userCurrentLecture[chatID]
    if !ok {
        msgText := "Пожалуйста, выберите конспект сначала."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    lecturesDir := filepath.Join(config.LecturesDir, class)

    files, err := os.ReadDir(lecturesDir)
    if err != nil {
        log.Printf("Ошибка при чтении директории с конспектами: %s", err)
        msgText := "Не удалось прочитать список конспектов."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    var lectureFiles []string
    for _, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
            lectureFiles = append(lectureFiles, filepath.Join(lecturesDir, file.Name()))
        }
    }
    sort.Strings(lectureFiles)

    currentIndex := -1
    for i, filePath := range lectureFiles {
        if filePath == currentLecturePath {
            currentIndex = i
            break
        }
    }

    if currentIndex == -1 {
        msgText := "Текущий конспект не найден в списке."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    if currentIndex+1 < len(lectureFiles) {
        nextLecturePath := lectureFiles[currentIndex+1]
        content, err := readFileContent(nextLecturePath)
        if err != nil {
            log.Printf("Ошибка при чтении следующей лекции: %s", err)
            msgText := "Не удалось прочитать следующую лекцию."
            msg := tgbotapi.NewMessage(chatID, msgText)
            bot.Send(msg)
            return
        }
        userCurrentLecture[chatID] = nextLecturePath 

        sendMessageInParts(bot, chatID, content)

        askForNextAction(bot, chatID, true) // Здесь класс будет показываться

    } else {
        msgText := "Это последняя лекция в списке."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        askForNextAction(bot, chatID, true) // Здесь также класс будет показываться
    }
}

// Новый метод для обработки "Предыдущего конспекта"
func prevLecture(bot *tgbotapi.BotAPI, config Config, chatID int64) {
    class, ok := userClass[chatID]
    if !ok {
        msgText := "Пожалуйста, выберите класс сначала (/start)."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    currentLecturePath, ok := userCurrentLecture[chatID]
    if !ok {
        msgText := "Пожалуйста, выберите конспект сначала."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    lecturesDir := filepath.Join(config.LecturesDir, class)

    files, err := os.ReadDir(lecturesDir)
    if err != nil {
        log.Printf("Ошибка при чтении директории с конспектами: %s", err)
        msgText := "Не удалось прочитать список конспектов."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    var lectureFiles []string
    for _, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
            lectureFiles = append(lectureFiles, filepath.Join(lecturesDir, file.Name()))
        }
    }
    sort.Strings(lectureFiles)

    currentIndex := -1
    for i, filePath := range lectureFiles {
        if filePath == currentLecturePath {
            currentIndex = i
            break
        }
    }

    if currentIndex == -1 || currentIndex == 0 {
        msgText := "Это первая лекция в списке."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    prevLecturePath := lectureFiles[currentIndex-1]
    content, err := readFileContent(prevLecturePath)
    if err != nil {
        log.Printf("Ошибка при чтении предыдущей лекции: %s", err)
        msgText := "Не удалось прочитать предыдущую лекцию."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }
    userCurrentLecture[chatID] = prevLecturePath 

    sendMessageInParts(bot, chatID, content)

    askForNextAction(bot, chatID, true) // Здесь класс будет показываться
}

func showLectureList(bot *tgbotapi.BotAPI, config Config, chatID int64) {
    class, ok := userClass[chatID]
    if !ok {
        msgText := "Пожалуйста, выберите класс сначала (/start)."
        msg := tgbotapi.NewMessage(chatID, msgText)
        bot.Send(msg)
        return
    }

    lecturesDir := filepath.Join(config.LecturesDir, class)
    keyboard := buildLectureKeyboard(lecturesDir, config)

    msgText := fmt.Sprintf("Выберите конспект для %s класса:", class)
    msg := tgbotapi.NewMessage(chatID, msgText)
    msg.ReplyMarkup = keyboard
    _, err := bot.Send(msg)
    if err != nil {
        log.Printf("Ошибка при отправке сообщения: %s", err)
    }
}