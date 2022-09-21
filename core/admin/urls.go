package admin

import (
	"sync"

	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/settings"
)

var once = sync.Once{}

func UrlPatterns(r *kamux.Router) {
	r.GET("/mon/ping", func(c *kamux.Context) { c.Status(200).Text("pong") })
	r.GET("/offline", OfflineView)
	r.GET("/manifest.webmanifest", ManifestView)
	r.GET("/sw.js", ServiceWorkerView)
	r.GET("/robots.txt", RobotsTxtView)
	r.GET("/admin", kamux.Admin(IndexView))
	r.GET("/admin/login", kamux.Auth(LoginView))
	r.POST("/admin/login", kamux.Auth(LoginPOSTView))
	r.GET("/admin/logout", LogoutView)
	r.POST("/admin/delete/row", kamux.Admin(DeleteRowPost))
	r.POST("/admin/update/row", kamux.Admin(UpdateRowPost))
	r.POST("/admin/create/row", kamux.Admin(CreateModelView))
	r.POST("/admin/drop/table", kamux.Admin(DropTablePost))
	r.GET("/admin/table/model:str", kamux.Admin(AllModelsGet))
	r.POST("/admin/table/model:str/search", kamux.Admin(AllModelsSearch))
	r.POST("/admin/table/model:str", kamux.Admin(AllModelsPost))
	r.GET("/admin/get/model:str/id:int", kamux.Admin(SingleModelGet))
	r.GET("/admin/export/table:str", kamux.Admin(ExportView))
	r.POST("/admin/import", kamux.Admin(ImportView))
	if settings.Config.Logs {
		once.Do(func() {
			r.UseMiddlewares(kamux.LOGS)
		})
		r.GET("/logs", kamux.Admin(LogsGetView))
		r.SSE("/sse/logs", kamux.Admin(LogsSSEView))
	}
}
