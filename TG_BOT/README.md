Summarize the following text:  Telegram Bot для Конспектов по Информатике

Этот проект представляет собой Telegram-бота, который предоставляет доступ к конспектам по информатике для учеников разных классов. Бот позволяет пользователям выбирать класс и получать доступ к соответствующим лекциям в текстовом формате (`.txt`).

 Функциональность

- Выбор класса: Бот приветствует пользователя и предлагает выбрать класс (7-10).
- Выбор конспекта: После выбора класса пользователю отображается список доступных конспектов.
- Чтение конспектов: Бот отправляет содержимое лекции по частям, позволяя пользователю легко переваривать информацию.
- Навигация между конспектами: Пользователь может перемещаться между предыдущими и следующими конспектами, а также получать список всех конспектов.
- Интерфейс: Поддерживает инлайн-кнопки для взаимодействия, чтобы сделать навигацию более удобной.

 Установка

 Необходимые зависимости

Для работы бота вам понадобятся:

- Go (версии 1.17 или выше)
- Telegram Bot API (используется библиотека `github.com/go-telegram-bot-api/telegram-bot-api/v5`)
- Godotenv (используется библиотека `github.com/joho/godotenv` для загрузки переменных окружения из файла `.env`)

 Шаги для установки

1. Склонируйте репозиторий на свой локальный компьютер:

   ```bash
   git clone <url-репозитория>
   cd <папка-репозитория>
   ```

2. Установите необходимые зависимости:

   ```bash
   go mod tidy
   ```

3. Создайте файл `.env` в корне проекта и укажите свои переменные окружения:

   ```env
   TELEGRAM_BOT_TOKEN=ваш_токен_бота
   LECTURES_DIR=путь_к_директории_с_конспектами
   ```

4. Создайте директорию для лекций (по умолчанию `lectures`):

   ```bash
   mkdir lectures
   ```

5. Запустите бота:

   ```bash
   go run main.go
   ```

 Использование

1. Запустите бота в Telegram и нажмите на кнопку "Старт".
2. Выберите ваш класс из предложенных.
3. Перейдите к нужному конспекту и читайте его.
4. Используйте навигационные кнопки, чтобы двигаться между конспектами или возвращаться к выбору класса.

