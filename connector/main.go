package main

import (
	"common/config"
	"common/metrics"
	"connector/app"
	"context"
	"fmt"
	"framework/game"
	"log"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "connector",
	Short: "connector 管理连接，session以及路由请求",
	Long:  `connector 管理连接，session以及路由请求`,
	Run: func(cmd *cobra.Command, args []string) {
	},
	PostRun: func(cmd *cobra.Command, args []string) {
	},
}

var (
	configFile    string
	gameConfigDir string
	serverId      string
)

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "application.yaml", "app config yml file")
	rootCmd.Flags().StringVar(&gameConfigDir, "gameDir", "../config", "game config dir")
	rootCmd.Flags().StringVar(&serverId, "serverId", "connector001", "app server id， required")
	_ = rootCmd.MarkFlagRequired("serverId")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	// 加载配置
	config.InitConfig(configFile)
	game.InitConfig(gameConfigDir)
	fmt.Printf("gameConf = %+v", game.Conf)
	// 启动监控
	go func() {
		err := metrics.Serve(fmt.Sprintf("localhost:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
	}()
	fmt.Println("serverId = ", serverId)
	err := app.Run(context.Background(), serverId)
	if err != nil {
		zap.L().Error("启动服务失败")
		os.Exit(-1)
	}
}
