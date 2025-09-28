package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Версия утилиты
const Version = "1.0.0"

func main() {
	// Флаги командной строки
	var (
		timeout = flag.Duration("timeout", 10*time.Second, "dial timeout (e.g. 5s, 250ms)")
		v       = flag.Bool("v", false, "print version and exit")
	)
	// Настройка справки по команде --help
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [--timeout=10s] <host> <port>\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "   or:  <host:port>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *v {
		fmt.Println(Version)
		return
	}

	// Получаем финальный адрес сервера
	addr, err := parseAddr(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов ОС (Ctrl+C и т.п.)
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-ctx.Done():
			return
		case s := <-sigCh:
			fmt.Fprintf(os.Stderr, "\n[%s] signal received, closing...\n", s)
			cancel()
		}
	}()

	// Подключение к серверу с учетом таймаута
	conn, err := dialWithTimeout(ctx, addr, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Fprintf(os.Stderr, "connected to %s (timeout %s)\n", addr, timeout.String())

	// Для half-close через FIN пробуем привести соединение к TCPConn
	tcp, _ := conn.(*net.TCPConn)

	var (
		wg   sync.WaitGroup
		once sync.Once
		// Функция для безопасного закрытия соединения и контекста, чтобы избежать гонок
		closeAll = func() { once.Do(func() { cancel(); _ = conn.Close() }) }
	)

	// Первая горутина: читаем из сокета и пишем в stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024) // буфер для ускорения копирования
		_, err := io.CopyBuffer(os.Stdout, conn, buf)
		if err != nil && !errors.Is(err, net.ErrClosed) && !isUseOfClosed(err) && !errors.Is(err, io.EOF) {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		}
		fmt.Fprintln(os.Stderr, "\n[connection closed by remote]")
		closeAll() // сервер закрыл соединение, завершаем
	}()

	// Вторая горутина: читаем stdin и пишем в сокет
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader := bufio.NewReader(os.Stdin)
		writer := bufio.NewWriter(conn)
		for {
			b, err := reader.ReadBytes('\n') // читаем строку до Enter
			if len(b) > 0 {
				if _, werr := writer.Write(b); werr != nil {
					break // ошибка записи - прекращаем работу
				}
				if werr := writer.Flush(); werr != nil {
					break
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					// Ctrl+D - закрываем запись (FIN), но продолжаем слушать ответы сервера
					if tcp != nil {
						_ = tcp.CloseWrite()
					} else {
						_ = conn.Close()
					}
					fmt.Fprintln(os.Stderr, "[stdin closed] sending FIN; waiting for remote...")
				} else if !errors.Is(err, net.ErrClosed) {
					fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
				}
				return
			}
		}
	}()

	// Ждём завершения обеих горутин
	wg.Wait()
}

// Разбор аргументов командной строки и формирование host:port
func parseAddr(args []string) (string, error) {
	switch len(args) {
	case 1:
		if !strings.Contains(args[0], ":") {
			return "", fmt.Errorf("single argument must be in host:port form")
		}
		return args[0], nil
	case 2:
		return net.JoinHostPort(args[0], args[1]), nil
	default:
		return "", fmt.Errorf("invalid arguments")
	}
}

// Подключение с учетом таймаута и контекста
func dialWithTimeout(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	return dialer.DialContext(ctx, "tcp", addr)
}

// Проверка на ошибку "use of closed network connection"
func isUseOfClosed(err error) bool {
	return err != nil && strings.Contains(err.Error(), "use of closed network connection")
}
