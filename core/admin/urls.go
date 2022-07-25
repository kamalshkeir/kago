package admin

import (
	"sync"

	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares"
	"github.com/kamalshkeir/kago/core/settings"
)
var once = sync.Once{}

func UrlPatterns(r *kamux.Router) {
	r.GET("/mon/ping",func(c *kamux.Context) {c.Status(200).Text("pong")})
	r.GET("/offline",OfflineView) 
	r.GET("/manifest.webmanifest",ManifestView) 
	r.GET("/sw.js",ServiceWorkerView) 
	r.GET("/robots.txt",RobotsTxtView) 
	r.GET("/admin", middlewares.Admin(IndexView))
	r.GET("/admin/login",middlewares.Auth(LoginView))
	r.POST("/admin/login",middlewares.Auth(LoginPOSTView))
	r.GET("/admin/logout", LogoutView)
	r.POST("/admin/delete/row", middlewares.Admin(DeleteRowPost))
	r.POST("/admin/update/row", middlewares.Admin(UpdateRowPost))
	r.POST("/admin/create/row", middlewares.Admin(CreateModelView))
	r.POST("/admin/drop/table", middlewares.Admin(DropTablePost))
	r.GET("/admin/table/model:str", middlewares.Admin(AllModelsGet))
	r.POST("/admin/table/model:str", middlewares.Admin(AllModelsPost))
	r.GET("/admin/get/model:str/id:int", middlewares.Admin(SingleModelGet))
	r.GET("/admin/export/table:str", middlewares.Admin(ExportView))
	r.POST("/admin/import", middlewares.Admin(ImportView))
	if settings.GlobalConfig.Logs {	
		once.Do(func() {
			r.UseMiddlewares(middlewares.LOGS)
		})
		r.GET("/logs",middlewares.Admin(LogsGetView))
		r.SSE("/sse/logs",middlewares.Admin(LogsSSEView))
	}
}