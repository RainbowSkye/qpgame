package main

import (
	"common/config"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"gateway/app"
)

var configFile = flag.String("config", "application.yaml", "config file")

func main() {
	// 加载配置
	flag.Parse()
	config.InitConfig(*configFile)
	// fmt.Println(config.Conf)
	// 启动监控
	go func() {
		err := metrics.Serve(fmt.Sprintf("localhost:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
	}()
	// 启动http服务
	app.Run(context.Background())
}
