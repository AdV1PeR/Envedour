# Быстрый старт

## Минимальная установка для OrangePi Zero3

### 1. Подготовка системы (5 минут)

```bash
# Установка зависимостей
sudo ./deploy/setup.sh

# Настройка сети (опционально)
sudo ./deploy/network-tuning.sh
```

### 2. Настройка переменных окружения

```bash
# Создайте .env файл
cp .env.example .env
nano .env

# Обязательно установите:
# BOT_TOKEN=ваш_токен_от_BotFather
```

### 3. Сборка

```bash
# Установка Go зависимостей
go mod download

# Сборка для ARM64
make build-arm
```

### 4. Установка сервиса

```bash
# Установка systemd сервиса
sudo ./deploy/install.sh

# Настройка переменных окружения для сервиса
sudo systemctl edit envedour-down-bot

# Добавьте в файл:
[Service]
Environment="BOT_TOKEN=ваш_токен"
Environment="REDIS_ADDR=localhost:6379"
```

### 5. Запуск

```bash
# Включить автозапуск
sudo systemctl enable envedour-down-bot

# Запустить
sudo systemctl start envedour-down-bot

# Проверить статус
sudo systemctl status envedour-down-bot

# Просмотр логов
sudo journalctl -u envedour-down-bot -f
```

## Проверка работы

```bash
# Мониторинг системы
./scripts/monitor.sh

# Проверка здоровья
./scripts/health-check.sh
```

## Docker установка (альтернатива)

```bash
# Настройте .env
cp .env.example .env
nano .env

# Запуск
docker-compose up -d

# Логи
docker-compose logs -f bot
```

## Первое использование

1. Найдите вашего бота в Telegram
2. Отправьте `/start`
3. Отправьте ссылку на видео (YouTube, VK, и т.д.)
4. Дождитесь скачивания и отправки

## Решение проблем

### Бот не отвечает
- Проверьте логи: `sudo journalctl -u envedour-down-bot -n 50`
- Убедитесь что BOT_TOKEN правильный
- Проверьте что Redis запущен: `sudo systemctl status redis`

### Недостаточно памяти
- Уменьшите WORKER_COUNT в .env
- Уменьшите MAX_FILE_SIZE_MB
- Проверьте: `free -h`

### Высокая температура
- Проверьте: `cat /sys/class/thermal/thermal_zone0/temp`
- Добавьте охлаждение
- Уменьшите нагрузку
