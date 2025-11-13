package tcp

import (
	"context"
	"go_redis/interface/tcp"
	"go_redis/lib/logger"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	Address string // TCP server address:port
}

func ListenAndServeWithSignal(cfg *Config, handler tcp.Handler) error {

	listener, err := net.Listen("tcp", cfg.Address)
	closeChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1) // signal.Notify期待带缓冲，否则极端条件会丢失信号
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-sigChan // 阻塞等待信号
		switch sig {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			closeChan <- struct{}{}
		}
	}()
	if err != nil {
		return err
	}
	logger.Info("tcp server start listen...")
	ListenAndServe(listener, handler, closeChan)
	return nil
}

func ListenAndServe(listener net.Listener, handler tcp.Handler, closeChan <-chan struct{}) {
	// closeChan 用于优雅关闭，当程序被kill时，业务能够停止
	go func() {
		<-closeChan // 当程序被杀死，堵塞态将转为运行
		logger.Info("shutting down...")
		_ = listener.Close()
		_ = handler.Close()
	}()
	defer func() {
		_ = listener.Close()
		_ = handler.Close()
	}() // 处理关闭逻辑
	ctx := context.Background()
	var waitDone sync.WaitGroup // 防止连接中断导致业务中断
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		logger.Info("accepted tcp link...")
		waitDone.Add(1)
		go func() {
			defer func() {
				waitDone.Done()
			}()
			handler.Handler(ctx, conn)
		}()
	}
	waitDone.Wait()
}
