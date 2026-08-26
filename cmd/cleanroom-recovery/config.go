package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	addr, dataDir    string
	selfCheck        bool
	selfCheckTimeout time.Duration
}

func parseConfig(args []string) (config, error) {
	defaultAddr := "127.0.0.1:19091"
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil || port < 1 || port > 65535 {
			return config{}, errors.New("PORT 必须是 1 至 65535 的端口号")
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", p)
	}
	fs := flag.NewFlagSet("cleanroom-recovery", flag.ContinueOnError)
	var c config
	fs.StringVar(&c.addr, "addr", defaultAddr, "HTTP 监听地址")
	fs.StringVar(&c.dataDir, "data", "./data", "本地数据目录")
	fs.BoolVar(&c.selfCheck, "self-check", false, "执行真实 HTTP 全流程自检后退出")
	fs.DurationVar(&c.selfCheckTimeout, "self-check-timeout", 12*time.Second, "自检超时")
	if err := fs.Parse(args); err != nil {
		return c, err
	}
	if fs.NArg() != 0 {
		return c, fmt.Errorf("未知位置参数: %s", strings.Join(fs.Args(), " "))
	}
	if err := validateAddress(c.addr); err != nil {
		return c, err
	}
	if c.selfCheckTimeout <= 0 {
		return c, errors.New("self-check-timeout 必须大于零")
	}
	return c, nil
}
func validateAddress(addr string) error {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("addr 必须为 host:port: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return errors.New("addr 不得省略主机")
	}
	port, err := strconv.Atoi(p)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("addr 端口无效")
	}
	return nil
}
