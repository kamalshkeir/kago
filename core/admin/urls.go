package admin

import (
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func UrlPatterns(r *kamux.Router) {
	// Admin
	r.Get("/offline",OfflineView) 
	r.Get("/manifest.webmanifest",ManifestView) 
	r.Get("/sw.js",ServiceWorkerView) 
	r.Get("/robots.txt",RobotsTxtView) 
	r.Get("/admin", middlewares.Admin(IndexView))
	r.Get("/admin/login",middlewares.Auth(LoginView))
	r.Post("/admin/login",middlewares.Auth(LoginPOSTView))
	r.Get("/admin/logout", LogoutView)
	r.Post("/admin/delete/row", middlewares.Admin(DeleteRowPost))
	r.Post("/admin/update/row", middlewares.Admin(UpdateRowPost))
	r.Post("/admin/create/row", middlewares.Admin(CreateModelView))
	r.Post("/admin/drop/table", middlewares.Admin(DropTablePost))
	r.Get("/admin/table/model:string", middlewares.Admin(AllModelsGet))
	r.Post("/admin/table/model:string", middlewares.Admin(AllModelsPost))
	r.Get("/admin/get/model:string/id:int", middlewares.Admin(SingleModelGet))
	r.Get("/admin/export/table:string", middlewares.Admin(ExportView))
	r.Post("/admin/import", middlewares.Admin(ImportView))
	if settings.GlobalConfig.Monitoring {
		r.Get("/metrics", func(c *kamux.Context) {
			promhttp.Handler().ServeHTTP(c.ResponseWriter,c.Request)
		})
	}
}