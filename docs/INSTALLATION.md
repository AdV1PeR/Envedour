# Установка Envedour Bot

Подробная инструкция по установке и настройке Envedour Bot на ARM64 системе.

## Содержание

- [Предварительные требования](#предварительные-требования)
- [Шаг 1: Подготовка системы](#шаг-1-подготовка-системы)
- [Шаг 2: Установка Telegram Bot API](#шаг-2-установка-telegram-bot-api)
- [Шаг 3: Настройка переменных окружения](#шаг-3-настройка-переменных-окружения)
- [Шаг 4: Сборка проекта](#шаг-4-сборка-проекта)
- [Шаг 5: Установка сервисов](#шаг-5-установка-сервисов)
- [Шаг 6: Запуск сервисов](#шаг-6-запуск-сервисов)
- [Шаг 7: Проверка работы](#шаг-7-проверка-работы)
- [Обновление](#обновление)
- [Удаление](#удаление)

## Предварительные требования

### Системные требования

- **ОС**: Debian 11+ / Ubuntu 20.04+ (aarch64)
- **Память**: Минимум 4GB RAM
- **Диск**: 10GB+ свободного места
- **Сеть**: Доступ в интернет
- **Права**: sudo доступ

### Программное обеспечение

Перед установкой убедитесь, что установлены:

- Go 1.22 или выше
- Git (для клонирования репозитория)

Проверка:

```bash
go version  # Должно быть 1.22+
git --version
```

## Шаг 1: Подготовка системы

### Клонирование репозитория

```bash
git clone <repository-url>
cd bot
```

### Установка зависимостей

Запустите скрипт установки зависимостей:

```bash
chmod +x deploy/setup.sh
sudo ./deploy/setup.sh
```

Скрипт выполнит:

1. **Обновление списка пакетов**
   ```bash
   sudo apt-get update
   ```

2. **Установку базовых зависимостей**:
   - `ffmpeg` - обработка видео
   - `aria2` - ускоренная загрузка
   - `python3` и `python3-pip` - для yt-dlp
   - `redis-server` - очередь задач
   - `build-essential` - инструменты сборки
   - `curl` и `wget` - для загрузки файлов

3. **Установку yt-dlp с поддержкой curl-cffi**:
   ```bash
   sudo pip3 install --upgrade "yt-dlp[default,curl-cffi]"
   ```
   Это необходимо для работы с TikTok.

4. **Настройку аппаратного ускорения**:
   - Добавление пользователя в группу `video`

5. **Создание tmpfs директории**:
   - `/dev/shm/videos` для временных файлов
   - Права доступа 1777

6. **Настройку Redis**:
   - Создание конфигурации для ARM
   - Оптимизация памяти
   - Запуск сервиса

### Опциональная настройка сети

Для оптимизации сетевых параметров:

```bash
chmod +x deploy/network-tuning.sh
sudo ./deploy/network-tuning.sh
```

## Шаг 2: Установка Telegram Bot API

Локальный Bot API рекомендуется для снижения нагрузки на серверы Telegram и увеличения пропускной способности.

### Получение API credentials

1. Перейдите на https://my.telegram.org/apps
2. Войдите в свой аккаунт Telegram
3. Создайте новое приложение:
   - **App title**: Любое название (например, "Envedour Bot")
   - **Short name**: Любое короткое имя
   - **Platform**: Desktop
   - **Description**: Опционально
4. Скопируйте `api_id` и `api_hash`

### Установка telegram-bot-api

```bash
# Скачайте бинарник для ARM64
wget https://github.com/tdlib/telegram-bot-api/releases/download/v7.0.0/telegram-bot-api_Linux_arm64 -O /tmp/telegram-bot-api

# Установите
sudo mv /tmp/telegram-bot-api /usr/local/bin/telegram-bot-api
sudo chmod +x /usr/local/bin/telegram-bot-api

# Проверьте установку
telegram-bot-api --version
```

**Примечание**: Скрипт `install.sh` может предложить установить telegram-bot-api автоматически.

## Шаг 3: Настройка переменных окружения

### Создание .env файла

```bash
# Создайте .env файл в корне проекта
nano .env
```

### Минимальная конфигурация

Добавьте обязательные параметры:

```bash
# Обязательные параметры
BOT_TOKEN=your_bot_token_from_botfather
TELEGRAM_API_ID=your_api_id
TELEGRAM_API_HASH=your_api_hash

# Базовые настройки
REDIS_ADDR=localhost:6379
WORKER_COUNT=4
TMPFS_PATH=/dev/shm/videos
MAX_FILE_SIZE_MB=2048
LOCAL_API_URL=http://localhost:8089
MIN_FREE_MEM_MB=256
```

### Полная конфигурация

Для полной функциональности добавьте:

```bash
# Cookies для платформ (рекомендуется)
TIKTOK_COOKIES=/opt/envedour-bot/tiktok_cookies.txt
INSTAGRAM_COOKIES=/opt/envedour-bot/instagram_cookies.txt
YOUTUBE_COOKIES=/opt/envedour-bot/youtube_cookies.txt

# Доноры (опционально)
DONOR_CHAT_IDS=123456789,987654321

# Настройки Telegram Bot API (опционально)
TELEGRAM_LOCAL=true
TELEGRAM_STAT=0
TELEGRAM_FILTER=0
TELEGRAM_MAX_WORKERS=1000
TELEGRAM_HTTP_PORT=8089
```

**Подробнее о всех параметрах**: См. [CONFIGURATION.md](CONFIGURATION.md)

### Получение BOT_TOKEN

1. Откройте https://t.me/BotFather в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям:
   - Введите имя бота
   - Введите username бота (должен заканчиваться на `bot`)
4. Скопируйте полученный токен (формат: `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)

### Настройка cookies (опционально, но рекомендуется)

Для работы с TikTok и Instagram нужны cookies файлы:

1. **Установите расширение браузера**:
   - Chrome: [Get cookies.txt](https://chrome.google.com/webstore/detail/get-cookiestxt/nmckokihipjgplolmcmjakknndddifde)
   - Firefox: [cookies.txt](https://addons.mozilla.org/en-US/firefox/addon/cookies-txt/)

2. **Экспортируйте cookies**:
   - Войдите на нужную платформу в браузере
   - Используйте расширение для экспорта cookies
   - Сохраните в формате Netscape

3. **Загрузите на сервер**:
   ```bash
   # Создайте файлы (если еще не созданы)
   sudo touch /opt/envedour-bot/tiktok_cookies.txt
   sudo touch /opt/envedour-bot/instagram_cookies.txt
   sudo touch /opt/envedour-bot/youtube_cookies.txt
   
   # Скопируйте содержимое cookies файлов
   sudo nano /opt/envedour-bot/tiktok_cookies.txt
   
   # Установите права
   sudo chown botuser:botuser /opt/envedour-bot/*_cookies.txt
   sudo chmod 644 /opt/envedour-bot/*_cookies.txt
   ```

## Шаг 4: Сборка проекта

### Установка Go зависимостей

```bash
go mod download
```

### Сборка для ARM64

```bash
make build-arm
```

После успешной сборки будет создан бинарник `envedour-bot-arm64`.

**Проверка**:
```bash
file envedour-bot-arm64
# Должно показать: envedour-bot-arm64: ELF 64-bit LSB executable, ARM aarch64
```

### Альтернативная сборка

Если `make` недоступен:

```bash
GOOS=linux GOARCH=arm64 \
    go build \
    -ldflags="-s -w -extldflags '-Wl,--hash-style=gnu'" \
    -o envedour-bot-arm64 .
```

## Шаг 5: Установка сервисов

### Автоматическая установка

```bash
chmod +x deploy/install.sh
sudo ./deploy/install.sh
```

Скрипт выполнит:

1. **Создание пользователя `botuser`** (если не существует)
   - Системный пользователь без shell
   - Группа `botuser`

2. **Создание директорий**:
   - `/opt/envedour-bot` - для бота
   - `/opt/telegram-bot-api/data` - для Bot API

3. **Копирование файлов**:
   - Бинарник `envedour-bot-arm64`
   - Файл `.env`

4. **Настройка cookies файлов**:
   - Создание файлов если не существуют
   - Установка правильных прав доступа

5. **Установка systemd сервисов**:
   - `telegram-bot-api.service`
   - `envedour-down-bot.service`

6. **Настройка системы**:
   - sysctl оптимизации
   - limits для пользователя
   - tmpfs в `/etc/fstab`

### Ручная установка (если нужно)

Если автоматическая установка не подходит:

```bash
# Создайте пользователя
sudo useradd -r -s /bin/false botuser

# Создайте директории
sudo mkdir -p /opt/envedour-bot
sudo mkdir -p /opt/telegram-bot-api/data
sudo chown -R botuser:botuser /opt/envedour-bot
sudo chown -R botuser:botuser /opt/telegram-bot-api

# Скопируйте файлы
sudo cp envedour-bot-arm64 /opt/envedour-bot/
sudo cp .env /opt/envedour-bot/.env
sudo cp .env /opt/telegram-bot-api/.env
sudo chown -R botuser:botuser /opt/envedour-bot
sudo chown -R botuser:botuser /opt/telegram-bot-api

# Установите сервисы
sudo cp deploy/telegram-bot-api.service /etc/systemd/system/
sudo cp deploy/envedour-down-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
```

## Шаг 6: Запуск сервисов

### Включение автозапуска

```bash
sudo systemctl enable telegram-bot-api envedour-down-bot
```

### Запуск сервисов

```bash
# Запустите Telegram Bot API
sudo systemctl start telegram-bot-api

# Дождитесь готовности Bot API (обычно 10-30 секунд)
sleep 15

# Запустите бота
sudo systemctl start envedour-down-bot
```

### Проверка статуса

```bash
# Статус Telegram Bot API
sudo systemctl status telegram-bot-api

# Статус бота
sudo systemctl status envedour-down-bot

# Оба сервиса должны быть в статусе "active (running)"
```

## Шаг 7: Проверка работы

### Проверка логов

```bash
# Логи Telegram Bot API
sudo journalctl -u telegram-bot-api -f

# Логи бота
sudo journalctl -u envedour-down-bot -f
```

**Ожидаемый вывод в логах бота**:
```
Connecting to Redis at localhost:6379...
✓ Redis connected
Connecting to http://localhost:8089...
✓ Bot initialized
```

### Проверка здоровья системы

```bash
# Запустите скрипт проверки
./scripts/health-check.sh

# Должен вернуть exit code 0 если все в порядке
```

### Мониторинг системы

```bash
# Полный мониторинг
./scripts/monitor.sh
```

### Тест работы бота

1. Найдите вашего бота в Telegram
2. Отправьте команду `/start`
3. Должно появиться меню с кнопками
4. Отправьте тестовую ссылку на YouTube видео
5. Должна появиться клавиатура выбора качества

## Обновление

### Обновление бота

```bash
# 1. Остановите сервис
sudo systemctl stop envedour-down-bot

# 2. Обновите код
git pull

# 3. Пересоберите
make build-arm

# 4. Переустановите
sudo ./deploy/install.sh

# 5. Запустите
sudo systemctl start envedour-down-bot
```

### Обновление зависимостей

```bash
# Обновите yt-dlp
sudo pip3 install --upgrade "yt-dlp[default,curl-cffi]"

# Обновите Go зависимости
go mod download
go mod tidy
```

### Обновление Telegram Bot API

```bash
# Остановите сервис
sudo systemctl stop telegram-bot-api

# Скачайте новую версию
wget https://github.com/tdlib/telegram-bot-api/releases/download/v7.0.0/telegram-bot-api_Linux_arm64 -O /tmp/telegram-bot-api

# Замените бинарник
sudo mv /tmp/telegram-bot-api /usr/local/bin/telegram-bot-api
sudo chmod +x /usr/local/bin/telegram-bot-api

# Запустите
sudo systemctl start telegram-bot-api
```

## Удаление

### Полное удаление

```bash
# Остановите и отключите сервисы
sudo systemctl stop envedour-down-bot telegram-bot-api
sudo systemctl disable envedour-down-bot telegram-bot-api

# Удалите сервисы
sudo rm /etc/systemd/system/envedour-down-bot.service
sudo rm /etc/systemd/system/telegram-bot-api.service
sudo systemctl daemon-reload

# Удалите файлы
sudo rm -rf /opt/envedour-bot
sudo rm -rf /opt/telegram-bot-api

# Удалите пользователя (опционально)
sudo userdel botuser

# Удалите Redis конфигурацию (опционально)
sudo rm /etc/redis/redis.conf.d/arm-optimized.conf
sudo systemctl restart redis-server
```

### Сохранение данных перед удалением

```bash
# Создайте резервную копию конфигурации
sudo tar -czf envedour-bot-backup-$(date +%Y%m%d).tar.gz \
    /opt/envedour-bot/.env \
    /opt/telegram-bot-api/.env \
    /opt/envedour-bot/*_cookies.txt

# Сохраните резервную копию Redis (если нужно)
./scripts/backup.sh
```

---

**Следующий шаг**: См. [CONFIGURATION.md](CONFIGURATION.md) для детальной настройки
