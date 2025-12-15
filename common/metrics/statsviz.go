package metrics

import (
	"net/http"

	"github.com/arl/statsviz"
)

// Serve 启动可视化监听指标服务 可视化图表 /debug/statsviz
func Serve(addr string) error {
	mux := http.NewServeMux()
	err := statsviz.Register(mux)
	if err != nil {
		return err
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		return err
	}
	return nil
}
