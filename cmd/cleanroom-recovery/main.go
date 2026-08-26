package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/store"
	"cleanroom-recovery-ledger/internal/web"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("cleanroom-recovery: %v", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	if cfg.selfCheck {
		return runSelfCheck(cfg)
	}
	repo, err := store.Open(cfg.dataDir)
	if err != nil {
		return err
	}
	handler := web.New(application.New(repo)).Handler()
	server := &http.Server{Addr: cfg.addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return err
	}
	log.Printf("洁净室恢复放行工作台已监听 http://%s", listener.Addr())
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case sig := <-signals:
		log.Printf("收到 %s，正在优雅关闭", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = server.Shutdown(ctx); err != nil {
			return err
		}
		err = <-serveErr
	case err = <-serveErr:
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP 服务异常: %w", err)
	}
	return nil
}
