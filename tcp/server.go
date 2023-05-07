package tcp

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// handler是应用层服务器的抽象
type Handler interface {
	Handle(ctx context.Context, conn net.Conn)
	Close() error
}

// Config stores tcp server properties
type Config struct {
	Address    string        `yaml:"address"`
	MaxConnect uint32        `yaml:"max-connect"`
	Timeout    time.Duration `yaml:"timeout"`
}

// 监听并提供服务，并在收到closeChan发来的关闭通知后关闭
func ListenAndServe(listener net.Listener, handler Handler, closeChan <-chan struct{}) {
	//监听关闭通知
	go func() {
		<-closeChan
		log.Println() //

		//停止监听,listen.Accept()会立即返回IO.EOF
		_ = listener.Close()
		//关闭应用层服务器
		_ = listener.Close()
	}()

	//异常退出后释放资源
	defer func() {
		// close during  unexpected error
		_ = listener.Close()
		_ = handler.Close()
	}()

	//这是什么
	ctx := context.Background()
	var waitDone sync.WaitGroup //这是什么
	for {
		//监听端口，阻塞直到收到新连接或者出现错误
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		//开启goroutine来处理新连接
		log.Println()
		waitDone.Add(1) //这是什么
		go func() {
			defer func() {
				waitDone.Done()
			}()
			handler.Handle(ctx, conn)
		}()
	}
	waitDone.Wait()
}

func ListenAndServeWithSignal(config *Config, handler Handler) error {
	closeChan := make(chan struct{})
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGALRM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			closeChan <- struct{}{}
		}
	}()
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return err
	}
	log.Println()
	ListenAndServe(listener, handler, closeChan)
	return nil
}
