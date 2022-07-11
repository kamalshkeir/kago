package admin

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/middlewares"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/encryptor"
	"github.com/kamalshkeir/kago/core/utils/encryption/hash"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

var PAGINATION_PER=6

var IndexView = func(c *kamux.Context) {
	allTables := orm.GetAllTables()
	c.Html("admin/admin_index.html", map[string]interface{}{
		"tables": allTables,
	})
}

var LoginView = func(c *kamux.Context) {
	c.Html("admin/admin_login.html", map[string]interface{}{})
}

var LoginPOSTView = func(c *kamux.Context) {
	requestData := c.GetJson()
	email := requestData["email"]
	passRequest := requestData["password"]

	data,err := orm.Database().Table("users").Where("email = ?",email).One()
	if err != nil {
		c.Json(500,map[string]interface{}{
			"error":err.Error(),
		})
		return
	}
	if data["email"] == "" || data["email"] == nil {
		c.Json(404, map[string]interface{}{
			"error":"User doesn not Exist",
		})
		return
	}
	if data["is_admin"] == int64(0) || data["is_admin"] == 0 || data["is_admin"] == false {
		c.Json(http.StatusForbidden, map[string]interface{}{
			"error":"Not Allowed to access this page",
		})
		return
	}


	if passDB,ok := data["password"].(string);ok {
		match, err := hash.ComparePasswordToHash(passRequest.(string), passDB)
		if !match || err != nil {
			c.Json(http.StatusForbidden, map[string]interface{}{
				"error":"Wrong Password",
			})
			return
		} else {
			if uuid,ok := data["uuid"].(string);ok {
				if middlewares.SESSION_ENCRYPTION {
					uuid,err = encryptor.Encrypt(uuid)
					logger.CheckError(err)
				}
				c.SetCookie("session",uuid)
				c.Json(200,map[string]interface{}{
					"success":"U Are Logged In",
				})
				return
			}
		}
	}
}

var LogoutView = func(c *kamux.Context) {
	c.DeleteCookie("session")
	c.Redirect("/",http.StatusSeeOther)
}

var AllModelsGet = func(c *kamux.Context) {
	model,ok := c.Params["model"]
	if !ok {
		c.Json(http.StatusBadRequest,map[string]interface{}{
			"error":"No model given in params",
		})
		return
	}
	rows,err :=orm.Database().Table(model).OrderBy("-id").Limit(PAGINATION_PER).Page(1).All()
	if err != nil {
		rows,_ =orm.Database().Table(model).All()
	}
	columns := orm.GetAllColumns(model)
	if settings.GlobalConfig.DbType != "" {
		c.Html("admin/admin_all_models.html", map[string]interface{}{
			"dbType":settings.GlobalConfig.DbType,
			"model_name":model,
			"rows":rows,
			"columns":columns,
		})
	} else {
		c.Json(200,map[string]any{
			"error":"dbType not known, do you have .env ?",
		})
	}
}

var AllModelsPost = func(c *kamux.Context) {
	model,ok := c.Params["model"]
	if !ok {
		c.Json(http.StatusBadRequest,map[string]interface{}{
			"error":"No model given in params",
		})
		return
	}
	received := c.GetJson()
	page := received["page_num"].(string)
	pagenum,err := strconv.Atoi(page)
	if logger.CheckError(err) {
		return
	}
	rows,err :=orm.Database().Table(model).OrderBy("-id").Limit(PAGINATION_PER).Page(pagenum).All()
	if err != nil {
		return
	}
	c.Json(200,map[string]interface{}{
		"rows":rows,
	})
}

var DeleteRowPost = func(c *kamux.Context) {
	data := c.GetJson()
	if data["mission"] == "delete_row" {
		model := data["model_name"]
		modelDB,err := orm.Database().Table(model.(string)).Where("id = ?",data["id"]).One() 

		if err != nil {
			logger.Error("data received:", data)
			c.Json(200, map[string]interface{}{
				"error":err.Error(),
			})
			return
		}
		if val,ok := modelDB["image"]; ok {
			if val != "" {
				_ = c.DeleteFile(val.(string))
			}
		}

		
		_,err = orm.Database().Table(model.(string)).Where("id = ?",data["id"].(string)).Delete()

		if err != nil {
			c.Json(200, map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			c.Json(200, map[string]interface{}{
				"success": "Done !",
				"id":data["id"],
			})
			return
		}
	}	
}

var CreateModelView = func(c *kamux.Context) {
	parseErr := c.Request.ParseMultipartForm(32 << 20)
	if parseErr != nil {
		logger.Error("Parse error = ", parseErr)
	}
	data := c.Request.Form

	model := data["table"][0]

	fields := []string{}
	values := []interface{}{}
	for key,val := range data {
		switch key {
		case "table":
			continue
		case "uuid":
			uuid,err := utils.GenerateUUID()
			logger.CheckError(err)
			fields = append(fields,key)
			values = append(values,uuid)
		case "password":
			hash,_ := hash.GenerateHash(val[0])
			fields = append(fields,key)
			values = append(values,hash)
		default:
			fields = append(fields,key)
			values = append(values,val[0])
		}

	}
	_,err := orm.Database().Table(model).Insert(
		strings.Join(fields,","),		
		values...
	)
	if logger.CheckError(err) {
		c.Json(200, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	c.Json(200, map[string]interface{}{
		"success": "Done !",
	})	
}

var SingleModelGet = func(c *kamux.Context) {
	model,ok := c.Params["model"]
	if !ok {
		c.Json(200, map[string]interface{}{
			"error": "param model not defined",
		})
		return
	}
	id,ok := c.Params["id"]
	if !ok {
		c.Json(200, map[string]interface{}{
			"error": "param id not defined",
		})
		return
	}
	modelRow,err := orm.Database().Table(model).Where("id = ?",id).One()
	if logger.CheckError(err) {
		c.Json(http.StatusBadRequest,map[string]interface{}{
			"error":err.Error(),
		})
		return
	}
	columns := orm.GetAllColumns(model)
	c.Html("admin/admin_single_model.html", map[string]interface{}{
		"model":modelRow,
		"model_name":model,
		"id":id,
		"columns":columns,
	})
}

var UpdateRowPost = func(c *kamux.Context) {
	// parse the form and get data values + files
	parseErr := c.Request.ParseMultipartForm(32 << 20)
	if parseErr != nil {
		logger.Error("Parse error = ", parseErr)
	}
	data := c.Request.Form
	files := c.Request.MultipartForm.File
	// id from string to int
	id := data["row_id"][0]
	//handle file upload
	handleFilesUpload(files,data["table"][0],id,c)
	//get model from database
	modelDB,err := orm.Database().Table(data["table"][0]).Where("id = ?",id).One()
	
	if err != nil {
		c.Json(200,map[string]interface{}{
			"error":err.Error(),
		})
		return
	}


	for key,val := range data {
		switch key {
		case "id","_id","uuid","file","image","photo","img","fichier","row_id","table":
		case "isadmin","is_admin","isAdmin","IsAdmin":
			isAdminString := fmt.Sprintf("%v",modelDB[key])
			if isAdminString == val[0] {
				continue
			} else {
				_,err := orm.Database().Table(data["table"][0]).Where("id = ?",id).Set(key+" = ?",val[0])
				
				if err != nil {
					c.Json(200, map[string]interface{}{
						"error": err.Error(),
					})
					return
				}
				c.Json(200, map[string]interface{}{
					"success": key + " successfully Updated !",
				})
				return
			}
		default:
			if modelDB[key] != val[0] {
				_,err := orm.Database().Table(data["table"][0]).Where("id = ?",id).Set(key+" = ?",val[0])
				logger.CheckError(err)
			}
		}//switch
	}

	c.Json(200, map[string]interface{}{
		"success": "Update Done !",
		"data":data,
	})	
}

func handleFilesUpload(files map[string][]*multipart.FileHeader,model string,id string,c *kamux.Context) {
	if len(files) > 0 {
		for key,val := range files {
			file,_ := val[0].Open()
			uploadedImage := utils.UploadFile(file,val[0].Filename)
			row,err := orm.Database().Table(model).Where("id = ?",id).One()
			if err != nil {
				c.Json(200,map[string]interface{}{
					"error":err.Error(),
				})
				return
			}
			database_image := row[key]
			if database_image == uploadedImage {
				c.Json(200, map[string]interface{}{
					"error": "uploadedImage is the same !",
				})
				return
			} else {
				err := c.DeleteFile(database_image.(string))
				if err != nil {
					//le fichier existe pas
					_,err := orm.Database().Table(model).Where("id = ?",id).Set(key+" = ?",uploadedImage)
					logger.CheckError(err)
					continue
				} else {
					//le fichier existe et donc supprimer
					_,err := orm.Database().Table(model).Where("id = ?",id).Set(key+" = ?",uploadedImage)
					logger.CheckError(err)
					continue
				}
			}
			
		}
	}
}

var DropTablePost = func(c *kamux.Context) {
	data := c.GetJson()
	if data["table"] != "" {
		_,err := orm.Database().Table(data["table"].(string)).Drop()
		if logger.CheckError(err) {
			c.Json(200,map[string]interface{}{
				"error":err.Error(),
			})
			return
		}
	}

	c.Json(200,map[string]interface{}{
		"success":fmt.Sprintf("table %s Deleted !",data["table"]),
	})
}

var ExportView= func(c *kamux.Context) {
	table,ok := c.Params["table"]
	if !ok {
		c.Json(200,map[string]interface{}{
			"error":"no param table found",
		})
		return
	}
	data,err := orm.Database().Table(table).All()
	logger.CheckError(err)

	data_bytes,err := json.Marshal(data)
	logger.CheckError(err)
	
	c.Download(data_bytes,table+".json")
}

var ImportView= func(c *kamux.Context) {
	// get table name
	table := c.Request.FormValue("table")
	if table == "" {
		c.Json(200,map[string]interface{}{
			"error":"no table !",
		})
		return
	}
	// upload file and return bytes of file
	_,dataBytes,err := c.UploadFileFromFormData("thefile","backup")
	if logger.CheckError(err) {
		c.Json(200,map[string]interface{}{
			"error":err.Error(),
		})
		return
	}
	// fill list_map
	list_map := []map[string]interface{}{}
	json.Unmarshal(dataBytes,&list_map)
	// create models in database
	cols := []string{}
	values := []interface{}{}
	for _,m := range list_map {
		for k,v := range m {		
			cols = append(cols, k)
			values = append(values, v)
		}
		_,err := orm.Database().Table(table).Insert(strings.Join(cols,","),values...)
		if logger.CheckError(err) {
			c.Json(200,map[string]interface{}{
				"error":err.Error(),
			})
			return
		}
		cols = cols[:0]
		values = values[:0]
	} 

	c.Json(200,map[string]interface{}{
		"success":"Import Done , you can find backups at media folder",
	})
}

var ManifestView= func (c *kamux.Context) {
	if settings.GlobalConfig.EmbedStatic {
		f,err := kamux.Static.ReadFile("assets/static/manifest.json")
		if err != nil {
			logger.Error("cannot embed manifest.json from static",err)
			return
		}
		c.EmbedFile("application/json; charset=utf-8",f)
	} else {
		c.File("application/json; charset=utf-8","./assets/static/manifest.json")
	}
}

var ServiceWorkerView = func(c *kamux.Context) {
	if settings.GlobalConfig.EmbedStatic  {
		f,err := kamux.Static.ReadFile("assets/static/sw.js")
		if err != nil {
			logger.Error("cannot embed sw.js from static",err)
			return
		}
		c.EmbedFile("application/javascript; charset=utf-8",f)
	} else {
		c.File("application/javascript; charset=utf-8","./assets/static/sw.js")
	}
}

var RobotsTxtView = func(c *kamux.Context) {
	c.File("text/plain; charset=utf-8","./static/robots.txt")
}

var OfflineView= func (c *kamux.Context) {
	c.Text(200,"<h1>YOUR ARE OFFLINE, check connection</h1>")
}
