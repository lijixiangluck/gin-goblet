package httpd

import "syscall"

var (
	SIGINT  = syscall.SIGINT
	SIGTERM = syscall.SIGTERM
	SIGUSR2 = syscall.SIGTERM
)
