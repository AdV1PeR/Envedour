# Решение проблем Envedour Bot

Руководство по диагностике и решению типичных проблем.

## Содержание

- [Бот не запускается](#бот-не-запускается)
- [Бот не отвечает на сообщения](#бот-не-отвечает-на-сообщения)
- [Ошибки при скачивании](#ошибки-при-скачивании)
- [Высокая температура CPU](#высокая-температура-cpu)
- [Недостаточно памяти](#недостаточно-памяти)
- [Проблемы с TikTok](#проблемы-с-tiktok)
- [Проблемы с Redis](#проблемы-с-redis)
- [Проблемы с cookies](#проблемы-с-cookies)
- [Общие проблемы](#общие-проблемы)

## Бот не запускается

### Симптомы

- Сервис не стартует
- Сервис сразу падает
- Ошибки в логах при запуске

### Диагностика

```bash
# Проверьте статус
sudo systemctl status envedour-down-bot

# Проверьте логи
sudo journalctl -u envedour-down-bot -n 50

# Попробуйте запустить вручную
sudo -u botuser /opt/envedour-bot/envedour-bot-arm64 --arm-optimized
```

### Возможные причины и решения

#### 1. BOT_TOKEN не установлен

**Ошибка**: `BOT_TOKEN is required`

**Решение**:
```bash
# Проверьте .env файл
sudo cat /opt/envedour-bot/.env | grep BOT_TOKEN

# Если пусто, добавьте
sudo nano /opt/envedour-bot/.env
# Добавьте: BOT_TOKEN=your_token_here

# Проверьте права
sudo chown botuser:botuser /opt/envedour-bot/.env
```

#### 2. Redis не запущен

**Ошибка**: `Failed to connect to Redis`

**Решение**:
```bash
# Запустите Redis
sudo systemctl start redis-server

# Проверьте статус
sudo systemctl status redis-server

# Проверьте подключение
redis-cli ping
# Должно вернуть: PONG
```

#### 3. Нет прав на tmpfs

**Ошибка**: `Permission denied` в `/dev/shm/videos`

**Решение**:
```bash
# Проверьте монтирование
mountpoint /dev/shm/videos

# Если не смонтирован
sudo mkdir -p /dev/shm/videos
sudo mount /dev/shm/videos

# Установите права
sudo chmod 1777 /dev/shm/videos

# Проверьте в /etc/fstab
cat /etc/fstab | grep videos
```

#### 4. Бинарник не найден

**Ошибка**: `No such file or directory`

**Решение**:
```bash
# Проверьте существование
ls -lh /opt/envedour-bot/envedour-bot-arm64

# Если нет, пересоберите
cd /path/to/bot
make build-arm

# Переустановите
sudo ./deploy/install.sh
```

#### 5. Неправильные права доступа

**Ошибка**: `Permission denied`

**Решение**:
```bash
# Установите правильные права
sudo chown -R botuser:botuser /opt/envedour-bot
sudo chmod +x /opt/envedour-bot/envedour-bot-arm64
```

## Бот не отвечает на сообщения

### Симптомы

- Команды не обрабатываются
- Нет реакции на ссылки
- Бот не отвечает

### Диагностика

```bash
# Проверьте, что бот запущен
sudo systemctl is-active envedour-down-bot

# Проверьте логи на ошибки
sudo journalctl -u envedour-down-bot -p err

# Проверьте подключение к Bot API
curl http://localhost:8089/health
```

### Возможные причины и решения

#### 1. Telegram Bot API не запущен

**Решение**:
```bash
# Запустите Bot API
sudo systemctl start telegram-bot-api

# Проверьте статус
sudo systemctl status telegram-bot-api

# Проверьте логи
sudo journalctl -u telegram-bot-api -f
```

#### 2. Неправильный LOCAL_API_URL

**Решение**:
```bash
# Проверьте настройку
sudo cat /opt/envedour-bot/.env | grep LOCAL_API_URL

# Должно быть: LOCAL_API_URL=http://localhost:8089

# Проверьте порт Bot API
sudo cat /opt/telegram-bot-api/.env | grep TELEGRAM_HTTP_PORT

# Если порт другой, обновите LOCAL_API_URL
```

#### 3. Проблемы с сетью

**Решение**:
```bash
# Проверьте доступность Bot API
curl http://localhost:8089/health

# Проверьте доступность Telegram
ping api.telegram.org

# Проверьте порты
netstat -tuln | grep -E "6379|8089"
```

#### 4. Неправильный BOT_TOKEN

**Решение**:
```bash
# Проверьте токен
sudo cat /opt/envedour-bot/.env | grep BOT_TOKEN

# Проверьте формат (должен быть: number:hash)
# Если неправильный, получите новый у @BotFather
```

## Ошибки при скачивании

### Симптомы

- "Ошибка при скачивании"
- Файлы не загружаются
- Ошибки в логах

### Диагностика

```bash
# Проверьте логи на детали ошибок
sudo journalctl -u envedour-down-bot | grep -i "error\|failed"

# Проверьте доступность yt-dlp
yt-dlp --version

# Проверьте доступность aria2c
aria2c --version

# Проверьте доступность ffmpeg
ffmpeg -version
```

### Возможные причины и решения

#### 1. Недостаточно памяти

**Ошибка**: `insufficient memory`

**Решение**:
```bash
# Проверьте использование
free -h

# Уменьшите MAX_FILE_SIZE_MB
sudo nano /opt/envedour-bot/.env
# Измените: MAX_FILE_SIZE_MB=1024

# Или уменьшите WORKER_COUNT
# Измените: WORKER_COUNT=2

# Перезапустите
sudo systemctl restart envedour-down-bot
```

#### 2. TikTok 403 Forbidden

**Ошибка**: `403 Forbidden` или `impersonation warning`

**Решение**:
```bash
# Обновите cookies
# Экспортируйте свежие cookies из браузера
# Замените файл
sudo nano /opt/envedour-bot/tiktok_cookies.txt

# Установите права
sudo chown botuser:botuser /opt/envedour-bot/tiktok_cookies.txt

# Убедитесь, что установлен curl-cffi
pip3 show curl-cffi

# Если нет, установите
sudo pip3 install curl-cffi

# Обновите yt-dlp
sudo pip3 install --upgrade "yt-dlp[default,curl-cffi]"
```

#### 3. Файл слишком большой

**Ошибка**: `file too large`

**Решение**:
```bash
# Увеличьте MAX_FILE_SIZE_MB
sudo nano /opt/envedour-bot/.env
# Измените: MAX_FILE_SIZE_MB=4096

# Или выберите меньшее качество при скачивании
```

#### 4. Проблемы с tmpfs

**Ошибка**: `No space left on device`

**Решение**:
```bash
# Проверьте использование tmpfs
df -h /dev/shm/videos

# Очистите старые файлы
sudo rm -rf /dev/shm/videos/*

# Увеличьте размер tmpfs в /etc/fstab
sudo nano /etc/fstab
# Измените: size=2G на size=4G

# Перемонтируйте
sudo umount /dev/shm/videos
sudo mount /dev/shm/videos
```

#### 5. yt-dlp не может скачать

**Ошибка**: `yt-dlp failed`

**Решение**:
```bash
# Обновите yt-dlp
sudo pip3 install --upgrade yt-dlp

# Проверьте версию
yt-dlp --version

# Попробуйте скачать вручную
yt-dlp <url>
```

## Высокая температура CPU

### Симптомы

- Температура >85°C
- Throttling активен
- Снижение производительности

### Диагностика

```bash
# Текущая температура
cat /sys/class/thermal/thermal_zone0/temp

# В градусах
echo $(($(cat /sys/class/thermal/thermal_zone0/temp) / 1000))°C

# Проверьте throttling
dmesg | grep -i thermal
```

### Решения

#### 1. Добавьте охлаждение

**Пассивное**:
- Установите радиатор на CPU

**Активное**:
- Установите вентилятор
- Улучшите вентиляцию корпуса

#### 2. Уменьшите нагрузку

```bash
# Уменьшите WORKER_COUNT
sudo nano /opt/envedour-bot/.env
# Измените: WORKER_COUNT=2

# Перезапустите
sudo systemctl restart envedour-down-bot
```

#### 3. Ограничьте CPU

```bash
# В systemd service
sudo systemctl edit envedour-down-bot

# Добавьте:
[Service]
CPUQuota=200%
```

#### 4. Используйте меньшее качество по умолчанию

Рекомендуйте пользователям выбирать 720p вместо 1080p.

## Недостаточно памяти

### Симптомы

- Ошибки "insufficient memory"
- OOM killer активируется
- Система зависает

### Диагностика

```bash
# Использование памяти
free -h

# Память процессов
ps aux --sort=-%mem | head -10

# Логи OOM
dmesg | grep -i oom
```

### Решения

#### 1. Уменьшите настройки

```bash
# Уменьшите WORKER_COUNT
WORKER_COUNT=2

# Уменьшите MAX_FILE_SIZE_MB
MAX_FILE_SIZE_MB=1024

# Увеличьте MIN_FREE_MEM_MB
MIN_FREE_MEM_MB=512
```

#### 2. Оптимизируйте Redis

```bash
# Уменьшите maxmemory
sudo nano /etc/redis/redis.conf.d/arm-optimized.conf
# Измените: maxmemory 512mb

# Перезапустите Redis
sudo systemctl restart redis-server
```

#### 3. Добавьте swap (не рекомендуется)

```bash
# Создайте swap файл
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Добавьте в /etc/fstab
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

**Примечание**: Swap снижает производительность, используйте только в крайнем случае.

## Проблемы с TikTok

### Симптомы

- 403 Forbidden
- "impersonation warning"
- Видео не скачиваются

### Решения

#### 1. Установите curl-cffi

```bash
# Проверьте установку
pip3 show curl-cffi

# Если нет, установите
sudo pip3 install curl-cffi

# Обновите yt-dlp с поддержкой curl-cffi
sudo pip3 install --upgrade "yt-dlp[default,curl-cffi]"
```

#### 2. Обновите cookies

```bash
# Экспортируйте свежие cookies из браузера
# Формат: Netscape cookies

# Замените файл
sudo nano /opt/envedour-bot/tiktok_cookies.txt

# Установите права
sudo chown botuser:botuser /opt/envedour-bot/tiktok_cookies.txt
sudo chmod 644 /opt/envedour-bot/tiktok_cookies.txt
```

#### 3. Обновите yt-dlp

```bash
# Обновите до последней версии
sudo pip3 install --upgrade "yt-dlp[default,curl-cffi]"

# Проверьте версию
yt-dlp --version
```

## Проблемы с Redis

### Симптомы

- Ошибки подключения
- Потеря данных
- Медленная работа

### Диагностика

```bash
# Статус Redis
sudo systemctl status redis-server

# Проверка подключения
redis-cli ping

# Информация о памяти
redis-cli INFO memory
```

### Решения

#### 1. Перезапустите Redis

```bash
sudo systemctl restart redis-server
```

#### 2. Проверьте конфигурацию

```bash
# Проверьте maxmemory
redis-cli CONFIG GET maxmemory

# Проверьте политику
redis-cli CONFIG GET maxmemory-policy
```

#### 3. Очистите данные (если нужно)

```bash
# Остановите бота
sudo systemctl stop envedour-down-bot

# Очистите Redis
redis-cli FLUSHALL

# Запустите бота
sudo systemctl start envedour-down-bot
```

## Проблемы с cookies

### Симптомы

- Permission denied
- Cookies не читаются
- 403 Forbidden

### Решения

#### 1. Проверьте права доступа

```bash
# Проверьте права
ls -l /opt/envedour-bot/*_cookies.txt

# Установите правильные права
sudo chown botuser:botuser /opt/envedour-bot/*_cookies.txt
sudo chmod 644 /opt/envedour-bot/*_cookies.txt
```

#### 2. Проверьте формат

```bash
# Проверьте начало файла (должно быть Netscape format)
head -1 /opt/envedour-bot/tiktok_cookies.txt
# Должно начинаться с: # Netscape HTTP Cookie File
```

#### 3. Обновите cookies

Экспортируйте свежие cookies из браузера и замените файлы.

## Общие проблемы

### Проблема: Бот работает медленно

**Решения**:
1. Увеличьте WORKER_COUNT (но не больше ядер)
2. Используйте локальный Bot API
3. Оптимизируйте сеть
4. Проверьте температуру CPU

### Проблема: Файлы не отправляются

**Решения**:
1. Проверьте размер файла (лимит Telegram: 2GB для видео)
2. Проверьте подключение к Bot API
3. Проверьте логи на ошибки отправки

### Проблема: Очередь не обрабатывается

**Решения**:
1. Проверьте статус воркеров
2. Проверьте логи на ошибки
3. Проверьте доступную память
4. Проверьте температуру CPU

---

**Если проблема не решена**: Проверьте логи и создайте issue с деталями ошибки
