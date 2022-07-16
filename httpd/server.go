package httpd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	defaultTimeout = 5 * time.Second
	graceEnv       = "GOBLET=true"
)

var Server *GracefulServer

type GracefulServer struct {
	srv      *http.Server
	listener net.Listener
	timeout  time.Duration
}

func New() *GracefulServer {
	if Server == nil {
		Server = &GracefulServer{}
	}
	return Server
}

func (g *GracefulServer) ListenAndServe(address string, handler http.Handler) error {
	g.srv = &http.Server{
		Addr:    address,
		Handler: handler,
	}
	return g.run()
}

func (g *GracefulServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	return g.srv.Shutdown(ctx)
}

func (g *GracefulServer) getListenerFile() (*os.File, error) {
	switch t := g.listener.(type) {
	case *net.TCPListener:
		return t.File()
	case *net.UnixListener:
		return t.File()
	}
	return nil, fmt.Errorf("不支持的listener %T", g.listener)
}

func (g *GracefulServer) Reload() error {
	//复制socket文件描述符给子进程
	f, err := g.getListenerFile()
	if err != nil {
		return err
	}
	defer f.Close()

	var args []string
	//生成启动参数
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	execName, err := os.Executable()
	if err != nil {
		return err
	}
	execDir := filepath.Dir(execName)

	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), graceEnv)
	cmd.Dir = execDir
	cmd.ExtraFiles = []*os.File{f}
	return cmd.Start()
}

func (g *GracefulServer) run() (err error) {
	if _, ok := syscall.Getenv(strings.Split(graceEnv, "=")[0]); ok {
		// fork出来的子进程，通过文件描述符监听socket
		f := os.NewFile(3, "")
		if g.listener, err = net.FileListener(f); err != nil {
			return
		}
	} else {
		// 全新的进程，监听tcp端口
		if g.listener, err = net.Listen("tcp", g.srv.Addr); err != nil {
			return
		}
	}

	terminate := make(chan error)
	go func() {
		if err := g.srv.Serve(g.listener); err != nil {
			terminate <- err
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit)

	for {
		select {
		case s := <-quit:
			switch s {
			case SIGINT:
				signal.Stop(quit)
				return g.Stop()
			case SIGUSR2:
				err := g.Reload()
				if err != nil {
					return err
				}
				return g.Stop()
			case SIGTERM:
				signal.Stop(quit)
				return g.Stop()
			}
		case err = <-terminate:
			return
		}
	}
}
