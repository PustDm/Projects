# pico-samba

Максимально простой read-only SMB-сервис на чистом Go для Orange Pi и совместимых плат.

## Параметры (из командной строки)

| Параметр | Описание | По умолчанию |
|----------|----------|--------------|
| `-name` | Имя шары для клиентов (NetBIOS) | pico-samba |
| `-port` | TCP-порт | 445 |
| `-dir` | Корневая директория с контентом | (обязательный) |

## Особенности

- **Только чтение** — изменение файлов запрещено
- **Одна корневая директория** с поддиректориями
- **Симлинки запрещены** — при обнаружении возвращается ошибка
- **Гостевой доступ** без пароля
- **Без зависимостей** — один бинарник, не требует Samba

## Сборка

```bash
make build

# Для Orange Pi (linux/riscv64)
make build-riscv
```

## Запуск

```bash
./pico-samba -name=myshare -port=445 -dir=/srv/content
```

## Установка systemd

```bash
sudo make install
sudo systemctl edit pico-samba  # -name, -port, -dir
sudo systemctl enable --now pico-samba
```

Для порта 445 требуется root. Для непривилегированного порта используйте `-port=1445` и `User=nobody` в unit.

## Подключение клиентов

**Linux (smbclient):**
```bash
smbclient //IP-адрес/pico-samba -U guest%   # guest с пустым паролем
smbclient -L //IP-адрес -U guest%           # список шаров
```

**Linux (mount):**
```bash
mount -t cifs //IP-адрес/pico-samba /mnt -o guest,uid=1000
```

**Windows:** По умолчанию Windows передаёт текущего пользователя. Для гостевого доступа:
```cmd
net use Z: \\IP-адрес\pico-samba /user:guest
```
(при запросе пароля — Enter для пустого)

Либо включить: `reg add "HKLM\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters" /v AllowInsecureGuestAuth /t REG_DWORD /d 1 /f`

## Лицензия

See [LICENSE](LICENSE).
