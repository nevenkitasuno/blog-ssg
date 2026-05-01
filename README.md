# blog-ssg

Static site generator блога.

[Пример блога](https://nevenkitasuno.github.io/blog-ssg/) , сгенерированного по содержимому папки `/content`

- Читает Markdown контент из `content/`
- Поддержка тегов и картинок
- Инкрементальная сборка (только новые/изменённые файлы)

## Запуск

```bash
go run ./cmd/ssg
```

Можно переопределять пути:

```bash
go run ./cmd/ssg -content content -templates templates -output output
```

## Формат контента

Темы лежат в `content/`.

Структура папок:

```text
content/
  TopicName/
    2026 01 Untitled/
      1.md
      2.md
      image.png
```

Где:

- `TopicName` — название темы
- `2026 01 Untitled` — запись в формате `YYYY MM Заголовок`
- `1.md`, `2.md`, ... — страницы записи
- Также можно сохранять в папке ресурсы для записи

Пустые темы и пустые записи пропускаются.

## Теги

Теги задаются только на первой странице записи через YAML front matter:

```md
---
tags:
  - Go
  - HTML
---

Текст страницы
```

## Конфигурация темы и ссылки

Настройки темы и адрес темы задаются в `content/<Topic>/meta/Config.yaml`.

Пример:

```yaml
link_name: configured-link-name
background: "#f5f7fa"
accent: "#123456"
```

`link_name` управляет URL темы: страница темы будет сгенерирована в `output/topics/configured-link-name/index.html`.

## Изображения

Изображения можно вставлять в Markdown в стиле Obsidian:

```md
![[image.png]]
```

Файл `image.png` должен лежать внутри папки записи.

## Генерируемые страницы

Генератор создает:

- главную страницу `output/index.html`
- страницу темы `output/topics/<topic>/index.html`
- страницу тега `output/topics/<topic>/tags/<tag>/index.html`
- страницы записи:
  - первая страница: `output/topics/<topic>/<entry>/index.html`
  - следующие страницы: `output/topics/<topic>/<entry>/<page>/index.html`
