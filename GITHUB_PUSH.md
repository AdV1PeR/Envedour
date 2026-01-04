# Инструкция по загрузке на GitHub

## Шаг 1: Создайте репозиторий на GitHub

1. Откройте https://github.com/new
2. Название: `envedour-bot` (или другое)
3. Описание: "ARM optimized Telegram video downloader bot"
4. Выберите Public или Private
5. **НЕ** добавляйте README, .gitignore или лицензию (они уже есть)
6. Нажмите "Create repository"

## Шаг 2: Добавьте remote и запушьте

После создания репозитория выполните:

```bash
# Замените YOUR_USERNAME на ваш GitHub username
git remote add origin https://github.com/YOUR_USERNAME/envedour-bot.git

# Или если используете SSH:
# git remote add origin git@github.com:YOUR_USERNAME/envedour-bot.git

# Загрузите код
git push -u origin main
```

## Альтернатива: Через GitHub CLI

Если установлен `gh`:

```bash
gh repo create envedour-bot --public --source=. --remote=origin --push
```

## Проверка

После успешной загрузки проверьте:

```bash
git remote -v
# Должно показать ваш репозиторий

git log --oneline
# Должен показать ваш коммит
```
