# pico-dlna

Минимальный DLNA‑сервер для раздачи медиаконтента (видео, музыка, фото).

## Требования

- Go 1.23+
- Целевая платформа: Linux (в т.ч. Ubuntu 24 на Orange Pi RV2, riscv64)

## Использование

```
pico-dlna -name "Имя для клиентов" -port 1338 -path /путь/к/медиа
```

| Параметр | Описание | Обязательный |
|----------|----------|--------------|
| `-name`  | Имя сервера для DLNA‑клиентов | нет (по умолчанию: Pico-DLNA) |
| `-port`  | Порт HTTP (1338 или :1338) | нет (по умолчанию: :1338) |
| `-path`  | Корневая директория с медиа | **да** |

## Особенности

- Одна корневая директория с поддиректориями
- Симлинки не обрабатываются (запрещены)
- Без перекодирования — контент отдаётся без изменений
- Поддержка видео, аудио и изображений

## Сборка

```bash
# Локальная сборка
go build -o pico-dlna .

# Сборка для Orange Pi RV2 (linux/riscv64)
GOOS=linux GOARCH=riscv64 go build -o pico-dlna-riscv64 .
```

## Установка как systemd‑сервис

1. Скопировать бинарник:
   ```bash
   sudo install -m 755 pico-dlna /usr/local/bin/
   ```

2. Скопировать unit‑файл и настроить:
   ```bash
   sudo cp pico-dlna.service /etc/systemd/system/
   sudo ${EDITOR} /etc/systemd/system/pico-dlna.service
   # Изменить -name, -port, -path в ExecStart
   ```

3. Запуск:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable pico-dlna
   sudo systemctl start pico-dlna
   ```