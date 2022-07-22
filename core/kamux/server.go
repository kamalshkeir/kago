package kamux

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	gf "github.com/kamalshkeir/kago/core/middlewares/grafana"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/shell"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/prometheus/client_golang/prometheus"
)


var midwrs []func(http.Handler) http.Handler

// InitServer init the server with midws,
func (router *Router) initServer() {
	var handler http.Handler
	if len(midwrs) != 0 {
		handler = midwrs[0](router)
		for i := 1; i < len(midwrs); i++ {
			handler = midwrs[i](handler)
		}
	} else {
		handler = router
	}
	// add global cors
	host := settings.GlobalConfig.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := settings.GlobalConfig.Port
	if port == "" {
		port = "9313"
	}
	// check monitoring
	if settings.GlobalConfig.Monitoring {
		// handler latency added in servehttp
		handler = gf.Latency(handler)
		prometheus.MustRegister(gf.LatencySummary)
	}
	// Setup Server
	server := http.Server{
		Addr:         host + ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  20 * time.Second,
	}
	fmt.Printf(logger.Yellow, logger.Ascii7)
	fmt.Printf(logger.Blue, "-------⚡🚀 http://"+host+":"+port+" 🚀⚡-------")
	router.Server=&server
}

// UseMiddlewares chain global middlewares applied on the router
func (router *Router) UseMiddlewares(midws ...func(http.Handler) http.Handler) {
	midwrs = append(midwrs, midws...)
}


// Run start the server
func (router *Router) Run() {
	// init orm shell
	if shell.InitShell() {os.Exit(0)}
	// init templates and assets
	initTemplatesAndAssets(router)
	// init server
	router.initServer()
	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()

	if err := router.Server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("Unable to shutdown the server : ",err)
	} else {
		fmt.Printf(logger.Green,"Server Off !")
	}
}


// RunTLS start the server TLS
func (router *Router) RunTLS(certFile string,keyFile string) {
	// init orm shell
	if shell.InitShell() {os.Exit(0)}
	// init templates and assets
	initTemplatesAndAssets(router)
	// init server
	router.initServer()
	// graceful Shutdown server + db if exist
	go router.gracefulShutdown()

	if err := router.Server.ListenAndServeTLS(certFile,keyFile); err != http.ErrServerClosed {
		logger.Error("Unable to shutdown the server : ",err)
	} else {
		fmt.Printf(logger.Green,"Server Off !")
	}
}

// Graceful Shutdown
func (router *Router) gracefulShutdown() {
	err := utils.GracefulShutdown(func() error {
		// Close databases
		if err := orm.ShutdownDatabases();err != nil {
			logger.Error("unable to shutdown databases:",err)
		} else {
			fmt.Printf(logger.Blue,"Databases Closed")
		}
		// Shutdown server
		router.Server.SetKeepAlivesEnabled(false)
		err := router.Server.Shutdown(context.Background())
		if logger.CheckError(err) {return err}
		return nil
	})
	if logger.CheckError(err) {os.Exit(1)}
}





