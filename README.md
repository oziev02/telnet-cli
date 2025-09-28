# Telnet CLI


Небольшой telnet-клиент на Go для подключения к любому TCP-серверу и интерактивного обмена данными.


## Установка
```bash
go build -o telnet .
```


## Использование
```
./telnet [--timeout=10s] <host> <port>
./telnet <host:port>
```


Примеры:
```bash
./telnet tcpbin.com 4242
./telnet --timeout=3s smtp.gmail.com 25
```


## Горячие клавиши
- **Ctrl+D** — закрыть запись (FIN), продолжить чтение ответов сервера.
- **Ctrl+C** — завершить программу (graceful shutdown).


## Makefile (опционально)
```bash
# Запустить клиент к echo-серверу tcpbin.com:4242
make run


# Поменять параметры
make run HOST=smtp.gmail.com PORT=25 TIMEOUT=3s


# Форматирование и проверки
make fmt vet
```


## Схема
```
stdin ──▶ telnet (stdin→socket | socket→stdout) ──▶ TCP-сервер