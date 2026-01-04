# Мониторинг Envedour Bot

Руководство по мониторингу системы, логированию и метрикам.

## Содержание

- [Логирование](#логирование)
- [Скрипты мониторинга](#скрипты-мониторинга)
- [Метрики производительности](#метрики-производительности)
- [Проверка здоровья](#проверка-здоровья)
- [Алерты](#алерты)
- [Автоматический мониторинг](#автоматический-мониторинг)

## Логирование

### Просмотр логов бота

#### Все логи

```bash
sudo journalctl -u envedour-down-bot
```

#### Последние N строк

```bash
# Последние 100 строк
sudo journalctl -u envedour-down-bot -n 100

# Последние 50 строк
sudo journalctl -u envedour-down-bot -n 50
```

#### Следить за логами в реальном времени

```bash
sudo journalctl -u envedour-down-bot -f
```

#### Логи за период

```bash
# За последний час
sudo journalctl -u envedour-down-bot --since "1 hour ago"

# За последний день
sudo journalctl -u envedour-down-bot --since "1 day ago"

# С конкретной даты
sudo journalctl -u envedour-down-bot --since "2025-01-01 00:00:00"
```

#### Фильтрация по уровню

```bash
# Только ошибки
sudo journalctl -u envedour-down-bot -p err

# Критические ошибки
sudo journalctl -u envedour-down-bot -p crit

# Предупреждения и выше
sudo journalctl -u envedour-down-bot -p warning
```

#### Поиск в логах

```bash
# Поиск по тексту
sudo journalctl -u envedour-down-bot | grep "error"

# Поиск с контекстом
sudo journalctl -u envedour-down-bot | grep -A 5 -B 5 "error"
```

### Просмотр логов Telegram Bot API

```bash
# Все логи
sudo journalctl -u telegram-bot-api

# В реальном времени
sudo journalctl -u telegram-bot-api -f

# Последние 100 строк
sudo journalctl -u telegram-bot-api -n 100
```

### Просмотр логов Redis

```bash
# Все логи
sudo journalctl -u redis-server

# В реальном времени
sudo journalctl -u redis-server -f
```

### Экспорт логов

```bash
# В файл
sudo journalctl -u envedour-down-bot > logs.txt

# С датами
sudo journalctl -u envedour-down-bot --since "2025-01-01" > logs.txt

# В JSON (для анализа)
sudo journalctl -u envedour-down-bot -o json > logs.json
```

## Скрипты мониторинга

### monitor.sh - Полный мониторинг системы

**Расположение**: `scripts/monitor.sh`

**Использование**:
```bash
./scripts/monitor.sh
```

**Показывает**:
- Температуру CPU (все ядра)
- Частоты всех ядер
- Использование памяти (общее, свободное, использовано)
- Использование tmpfs
- Статус Redis (подключение, память)
- Статус сервисов (envedour-down-bot, telegram-bot-api)
- Статистику сети (если доступно)

**Пример вывода**:
```
=== System Monitoring ===
CPU Temperature: 45°C
CPU Frequencies: 1500 MHz (all cores)
Memory: 2.1G / 4.0G used (52%)
tmpfs: 500M / 2.0G used (25%)
Redis: Connected, 50M used
Services: envedour-down-bot: active, telegram-bot-api: active
```

### health-check.sh - Проверка здоровья

**Расположение**: `scripts/health-check.sh`

**Использование**:
```bash
./scripts/health-check.sh
```

**Проверяет**:
- Статус сервисов (должны быть active)
- Температуру CPU (предупреждение при >85°C)
- Доступную память (предупреждение при <512MB)
- Монтирование tmpfs
- Подключение к Redis

**Exit codes**:
- `0` - Все в порядке
- `1` - Есть проблемы

**Использование в cron**:
```bash
# Каждые 5 минут
*/5 * * * * /path/to/scripts/health-check.sh >> /var/log/envedour-bot-health.log 2>&1
```

### test-thermal.sh - Тест термального управления

**Расположение**: `scripts/test-thermal.sh`

**Использование**:
```bash
./scripts/test-thermal.sh
```

**Выполняет**:
- Запускает стресс-тест CPU на 60 секунд
- Показывает изменение температуры
- Проверяет работу термального throttling

**Требования**: `stress-ng` должен быть установлен

### test-memory.sh - Тест памяти

**Расположение**: `scripts/test-memory.sh`

**Использование**:
```bash
./scripts/test-memory.sh
```

**Показывает**:
- Использование памяти
- Использование tmpfs
- Доступную память
- Рекомендации

## Метрики производительности

### Проверка очереди

```bash
# Количество задач в очереди
redis-cli LLEN job_queue

# Просмотр задач (первые 10)
redis-cli LRANGE job_queue 0 9

# Очистка очереди (осторожно!)
redis-cli DEL job_queue
```

### Использование памяти

#### Общая память

```bash
# Краткая информация
free -h

# Детальная информация
free -m

# Непрерывный мониторинг
watch -n 1 free -h
```

#### Память процессов

```bash
# Топ процессов по памяти
ps aux --sort=-%mem | head -10

# Память бота
ps aux | grep envedour-bot-arm64

# Память Redis
redis-cli INFO memory
```

#### Память tmpfs

```bash
# Использование tmpfs
df -h /dev/shm/videos

# Детальная информация
du -sh /dev/shm/videos/*
```

### Использование CPU

```bash
# Текущая загрузка
top

# Краткая информация
uptime

# По процессам
ps aux --sort=-%cpu | head -10

# Непрерывный мониторинг
htop
```

### Температура CPU

```bash
# Текущая температура
cat /sys/class/thermal/thermal_zone0/temp

# В градусах Цельсия
echo $(($(cat /sys/class/thermal/thermal_zone0/temp) / 1000))°C

# Непрерывный мониторинг
watch -n 1 'echo $(($(cat /sys/class/thermal/thermal_zone0/temp) / 1000))°C'

# Все термальные зоны
for i in /sys/class/thermal/thermal_zone*/temp; do
    echo "$i: $(($(cat $i) / 1000))°C"
done
```

### Использование сети

```bash
# Статистика сети
ifconfig

# Или
ip -s link

# Мониторинг трафика
iftop

# Статистика по интерфейсу
cat /proc/net/dev
```

### Использование диска

```bash
# Использование дисков
df -h

# Использование tmpfs
df -h /dev/shm

# Детальная информация
du -sh /opt/envedour-bot/*
```

## Проверка здоровья

### Статус сервисов

```bash
# Статус бота
sudo systemctl status envedour-down-bot

# Статус Telegram Bot API
sudo systemctl status telegram-bot-api

# Статус Redis
sudo systemctl status redis-server

# Все сервисы
sudo systemctl status envedour-down-bot telegram-bot-api redis-server
```

### Проверка подключений

```bash
# Подключение к Redis
redis-cli ping
# Должно вернуть: PONG

# Подключение к Telegram Bot API
curl http://localhost:8089/health
# Должно вернуть статус

# Проверка портов
netstat -tuln | grep -E "6379|8089"
```

### Проверка зависимостей

```bash
# yt-dlp
yt-dlp --version

# aria2c
aria2c --version

# ffmpeg
ffmpeg -version

# Python
python3 --version
```

### Проверка файлов

```bash
# Существование бинарника
ls -lh /opt/envedour-bot/envedour-bot-arm64

# Права доступа
ls -l /opt/envedour-bot/

# Существование .env
ls -l /opt/envedour-bot/.env

# Существование cookies
ls -l /opt/envedour-bot/*_cookies.txt
```

## Алерты

### Настройка алертов

Создайте скрипт для отправки алертов:

```bash
#!/bin/bash
# /opt/envedour-bot/scripts/alert.sh

MESSAGE="$1"
# Отправка через Telegram, email, или другой способ
# Например, через curl к Telegram Bot API
```

### Алерты в health-check.sh

Модифицируйте `health-check.sh` для отправки алертов:

```bash
if [ $TEMP -gt 85 ]; then
    /opt/envedour-bot/scripts/alert.sh "CPU temperature is ${TEMP}°C"
fi
```

### Мониторинг через внешние системы

#### Prometheus

Экспорт метрик для Prometheus (требует дополнительной настройки).

#### Grafana

Визуализация метрик через Grafana.

#### Nagios / Zabbix

Интеграция с системами мониторинга.

## Автоматический мониторинг

### Настройка cron для health-check

```bash
# Редактировать crontab
sudo crontab -e

# Добавить строку (каждые 5 минут)
*/5 * * * * /opt/envedour-bot/scripts/health-check.sh >> /var/log/envedour-bot-health.log 2>&1
```

### Настройка cron для мониторинга

```bash
# Каждый час
0 * * * * /opt/envedour-bot/scripts/monitor.sh >> /var/log/envedour-bot-monitor.log 2>&1
```

### Ротация логов

Настройте logrotate для автоматической ротации логов:

```bash
# /etc/logrotate.d/envedour-bot
/var/log/envedour-bot-*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

### Systemd таймеры

Создайте systemd timer для автоматического мониторинга:

```ini
# /etc/systemd/system/envedour-bot-monitor.timer
[Unit]
Description=Envedour Bot Monitor Timer

[Timer]
OnCalendar=*:0/5
OnBootSec=5min

[Install]
WantedBy=timers.target
```

```ini
# /etc/systemd/system/envedour-bot-monitor.service
[Unit]
Description=Envedour Bot Monitor

[Service]
Type=oneshot
ExecStart=/opt/envedour-bot/scripts/health-check.sh
```

```bash
sudo systemctl enable envedour-bot-monitor.timer
sudo systemctl start envedour-bot-monitor.timer
```

## Рекомендации

1. **Регулярный мониторинг**: Запускайте `health-check.sh` каждые 5 минут
2. **Логирование**: Сохраняйте логи для анализа
3. **Алерты**: Настройте уведомления о критических проблемах
4. **Резервное копирование**: Регулярно создавайте резервные копии
5. **Обновления**: Следите за обновлениями зависимостей

---

**Следующий шаг**: См. [TROUBLESHOOTING.md](TROUBLESHOOTING.md) при обнаружении проблем
