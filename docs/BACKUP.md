# Резервное копирование Envedour Bot

Руководство по настройке и управлению резервным копированием.

## Содержание

- [Что нужно резервировать](#что-нужно-резервировать)
- [Автоматическое резервное копирование Redis](#автоматическое-резервное-копирование-redis)
- [Резервное копирование конфигурации](#резервное-копирование-конфигурации)
- [Восстановление](#восстановление)
- [Автоматизация](#автоматизация)
- [Рекомендации](#рекомендации)

## Что нужно резервировать

### Критически важно

1. **Конфигурация (.env файлы)**
   - `/opt/envedour-bot/.env`
   - `/opt/telegram-bot-api/.env`

2. **Cookies файлы**
   - `/opt/envedour-bot/tiktok_cookies.txt`
   - `/opt/envedour-bot/instagram_cookies.txt`
   - `/opt/envedour-bot/youtube_cookies.txt`

### Опционально

3. **Данные Redis** (если нужно сохранить очередь)
   - Обычно не требуется, так как очередь временная

4. **Бинарник** (если нужна конкретная версия)
   - `/opt/envedour-bot/envedour-bot-arm64`

## Автоматическое резервное копирование Redis

### Скрипт backup.sh

**Расположение**: `scripts/backup.sh`

**Функции**:
- Создает резервные копии данных Redis
- Сохраняет в `/opt/backups/envedour-bot/`
- Автоматически удаляет старые копии (через 7 дней)

### Использование

```bash
# Ручной запуск
./scripts/backup.sh

# Или с полным путем
/opt/envedour-bot/scripts/backup.sh
```

### Что делает скрипт

1. **Ожидает сохранения Redis**
   - Проверяет `LASTSAVE` в Redis
   - Ждет, пока Redis сохранит данные

2. **Создает резервную копию**
   - Копирует `/var/lib/redis/dump.rdb`
   - Сохраняет с временной меткой

3. **Очистка старых копий**
   - Удаляет копии старше 7 дней

### Формат файлов

```
/opt/backups/envedour-bot/redis_backup_YYYYMMDD_HHMMSS.rdb
```

**Пример**: `redis_backup_20250115_143022.rdb`

## Резервное копирование конфигурации

### Ручное резервное копирование

```bash
# Создайте директорию для резервных копий
sudo mkdir -p /opt/backups/envedour-bot/config

# Создайте резервную копию
sudo tar -czf /opt/backups/envedour-bot/config/config-backup-$(date +%Y%m%d).tar.gz \
    /opt/envedour-bot/.env \
    /opt/telegram-bot-api/.env \
    /opt/envedour-bot/*_cookies.txt

# Установите права
sudo chown botuser:botuser /opt/backups/envedour-bot/config/*
```

### Автоматическое резервное копирование

Создайте скрипт для автоматического резервного копирования:

```bash
#!/bin/bash
# /opt/envedour-bot/scripts/backup-config.sh

BACKUP_DIR="/opt/backups/envedour-bot/config"
DATE=$(date +%Y%m%d_%H%M%S)

# Создайте директорию
mkdir -p "$BACKUP_DIR"

# Создайте резервную копию
tar -czf "$BACKUP_DIR/config-backup-$DATE.tar.gz" \
    /opt/envedour-bot/.env \
    /opt/telegram-bot-api/.env \
    /opt/envedour-bot/*_cookies.txt 2>/dev/null

# Удалите старые копии (старше 30 дней)
find "$BACKUP_DIR" -name "config-backup-*.tar.gz" -mtime +30 -delete

echo "Config backup created: config-backup-$DATE.tar.gz"
```

**Сделайте исполняемым**:
```bash
chmod +x /opt/envedour-bot/scripts/backup-config.sh
```

## Восстановление

### Восстановление Redis

```bash
# 1. Остановите сервисы
sudo systemctl stop envedour-down-bot redis-server

# 2. Найдите нужную резервную копию
ls -lh /opt/backups/envedour-bot/redis_backup_*.rdb

# 3. Скопируйте резервную копию
sudo cp /opt/backups/envedour-bot/redis_backup_YYYYMMDD_HHMMSS.rdb \
    /var/lib/redis/dump.rdb

# 4. Установите права
sudo chown redis:redis /var/lib/redis/dump.rdb
sudo chmod 644 /var/lib/redis/dump.rdb

# 5. Запустите Redis
sudo systemctl start redis-server

# 6. Запустите бота
sudo systemctl start envedour-down-bot
```

### Восстановление конфигурации

```bash
# 1. Остановите сервисы
sudo systemctl stop envedour-down-bot telegram-bot-api

# 2. Распакуйте резервную копию
sudo tar -xzf /opt/backups/envedour-bot/config/config-backup-YYYYMMDD.tar.gz -C /

# 3. Установите права
sudo chown botuser:botuser /opt/envedour-bot/.env
sudo chown botuser:botuser /opt/telegram-bot-api/.env
sudo chown botuser:botuser /opt/envedour-bot/*_cookies.txt
sudo chmod 600 /opt/envedour-bot/.env
sudo chmod 600 /opt/telegram-bot-api/.env
sudo chmod 644 /opt/envedour-bot/*_cookies.txt

# 4. Запустите сервисы
sudo systemctl start telegram-bot-api envedour-down-bot
```

### Восстановление из полной резервной копии

Если у вас есть полная резервная копия системы:

```bash
# 1. Остановите все сервисы
sudo systemctl stop envedour-down-bot telegram-bot-api redis-server

# 2. Восстановите файлы
sudo tar -xzf full-backup.tar.gz -C /

# 3. Восстановите права
sudo chown -R botuser:botuser /opt/envedour-bot
sudo chown -R botuser:botuser /opt/telegram-bot-api

# 4. Запустите сервисы
sudo systemctl start redis-server
sudo systemctl start telegram-bot-api
sudo systemctl start envedour-down-bot
```

## Автоматизация

### Cron для резервного копирования Redis

```bash
# Редактировать crontab
sudo crontab -e

# Добавить строку (каждый день в 2:00)
0 2 * * * /opt/envedour-bot/scripts/backup.sh >> /var/log/envedour-bot-backup.log 2>&1
```

### Cron для резервного копирования конфигурации

```bash
# Добавить в crontab
0 3 * * * /opt/envedour-bot/scripts/backup-config.sh >> /var/log/envedour-bot-config-backup.log 2>&1
```

### Systemd таймеры

Создайте systemd timer для более надежного резервного копирования:

```ini
# /etc/systemd/system/envedour-bot-backup.timer
[Unit]
Description=Envedour Bot Backup Timer

[Timer]
OnCalendar=daily
OnBootSec=10min

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/envedour-bot-backup.service
[Unit]
Description=Envedour Bot Backup

[Service]
Type=oneshot
User=botuser
ExecStart=/opt/envedour-bot/scripts/backup.sh
ExecStart=/opt/envedour-bot/scripts/backup-config.sh
```

```bash
# Включите таймер
sudo systemctl enable envedour-bot-backup.timer
sudo systemctl start envedour-bot-backup.timer
```

## Рекомендации

### Частота резервного копирования

- **Конфигурация**: Ежедневно или при изменениях
- **Redis**: Ежедневно (если нужно сохранить очередь)
- **Cookies**: При обновлении

### Хранение резервных копий

1. **Локально**: `/opt/backups/envedour-bot/`
2. **Удаленно**: Копируйте на другой сервер или в облако
3. **Срок хранения**: 7-30 дней (настраивается)

### Безопасность резервных копий

1. **Права доступа**: Только для владельца
   ```bash
   sudo chmod 600 /opt/backups/envedour-bot/**/*.tar.gz
   ```

2. **Шифрование**: Для чувствительных данных
   ```bash
   # Создание зашифрованной копии
   tar -czf - /opt/envedour-bot/.env | gpg -c > backup.tar.gz.gpg
   ```

3. **Удаленное хранение**: Копируйте на другой сервер
   ```bash
   # SCP копирование
   scp backup.tar.gz user@remote-server:/backups/
   ```

### Проверка резервных копий

Регулярно проверяйте, что резервные копии создаются:

```bash
# Проверка последней копии
ls -lh /opt/backups/envedour-bot/

# Проверка содержимого
tar -tzf /opt/backups/envedour-bot/config/config-backup-*.tar.gz
```

### Тестирование восстановления

Периодически тестируйте восстановление:

1. Создайте тестовую среду
2. Восстановите резервную копию
3. Проверьте, что все работает

---

**Важно**: Резервное копирование бесполезно, если вы не можете восстановить данные. Регулярно тестируйте восстановление.
