package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/admin/models"
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
	c.Html("admin/admin_index.html", map[string]any{
		"tables": allTables,
	})
}

var LoginView = func(c *kamux.Context) {
	c.Html("admin/admin_login.html", map[string]any{})
}

var LoginPOSTView = func(c *kamux.Context) {
	requestData := c.BodyJson()
	email := requestData["email"]
	passRequest := requestData["password"]

	data,err := orm.Table("users").Where("email = ?",email).One()
	if err != nil {
		c.Status(500).Json(map[string]any{
			"error":err.Error(),
		})
		return
	}
	if data["email"] == "" || data["email"] == nil {
		c.Status(http.StatusNotFound).Json( map[string]any{
			"error":"User doesn not Exist",
		})
		return
	}
	if data["is_admin"] == int64(0) || data["is_admin"] == 0 || data["is_admin"] == false {
		c.Status(http.StatusForbidden).Json(map[string]any{
			"error":"Not Allowed to access this page",
		})
		return
	}


	if passDB,ok := data["password"].(string);ok {
		if pp,ok := passRequest.(string);ok {
			match, err := hash.ComparePasswordToHash(pp, passDB)
			if !match || err != nil {
				c.Status(http.StatusForbidden).Json(map[string]any{
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
					c.Json(map[string]any{
						"success":"U Are Logged In",
					})
					return
				}
			}
		}
	}
}

var LogoutView = func(c *kamux.Context) {
	c.DeleteCookie("session")
	c.Status(http.StatusTemporaryRedirect).Redirect("/")
}

var AllModelsGet = func(c *kamux.Context) {
	model,ok := c.Params["model"]
	if !ok {
		c.Json(map[string]any{
			"error":"Error: No model given in params",
		})
		return
	}
	
	rows,err :=orm.Table(model).OrderBy("-id").Limit(PAGINATION_PER).Page(1).All()
	if err != nil {
		rows,err =orm.Table(model).All()
		if err != nil {
			// usualy should not use error string because it divulge information, but here only admin use it, so no worry
			if err.Error() != "no data found" {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error":err.Error(),
				})
				return
			}
		}
	}
	columns := orm.GetAllColumns(model)
	if settings.GlobalConfig.DbType != "" {
		c.Html("admin/admin_all_models.html", map[string]any{
			"dbType":settings.GlobalConfig.DbType,
			"model_name":model,
			"rows":rows,
			"columns":columns,
		})
	} else {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":"dbType not known, do you have .env ?",
		})
	}
}

var AllModelsPost = func(c *kamux.Context) {
	model,ok := c.Params["model"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":"No model given in params",
		})
		return
	}
	received := c.BodyJson()
	if received != nil {
		if v,ok := received["page_num"];ok {
			if page,ok := v.(string);!ok {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error":"expecting page_num to be a sring",
				})
				return
			} else {
				pagenum,err := strconv.Atoi(page)
				if err == nil {
					rows,err :=orm.Table(model).OrderBy("-id").Limit(PAGINATION_PER).Page(pagenum).All()
					if err == nil {
						c.Json(map[string]any{
							"rows":rows,
						})
					}
				}	
			}
		} else {
			logger.Error("page_num not given",received)
		}
	} else {
		c.Json([]map[string]any{})
	}
}

var DeleteRowPost = func(c *kamux.Context) {
	data := c.BodyJson()
	if data["mission"] == "delete_row" {
		if model,ok := data["model_name"];ok {
			if mm,ok := model.(string);ok {
				orm.Model[models.User]().Delete()
				modelDB,err := orm.Table(mm).Where("id = ?",data["id"]).One() 
				if logger.CheckError(err) {
					logger.Info("data received DeleteRowPost:", data)
					c.Status(http.StatusBadRequest).Json(map[string]any{
						"error":err.Error(),
					})
					return
				}
				if val,ok := modelDB["image"]; ok {
					if vv,ok := val.(string);ok && vv != "" {
						_ = c.DeleteFile(vv)
					} 
				} 

				if idS,ok := data["id"].(string);ok {
					_,err = orm.Table(mm).Where("id = ?",idS).Delete()

					if err != nil {
						c.Status(http.StatusBadRequest).Json(map[string]any{
							"error": err.Error(),
						})
					} else {
						c.Json(map[string]any{
							"success": "Done !",
							"id":data["id"],
						})
						return
					}
				}
				
			} else {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error":"expecting model_name to be string",
				})
				return
			}
		} else {
			c.Status(http.StatusBadRequest).Json(map[string]any{
				"error":"no model_name found in request body",
			})
			return
		}
	}	
}

var CreateModelView = func(c *kamux.Context) {
	parseErr := c.Request.ParseMultipartForm(int64(kamux.MultipartSize))
	if parseErr != nil {
		logger.Error("Parse error = ", parseErr)
	}
	data := c.Request.Form

	model := data["table"][0]

	fields := []string{}
	values := []any{}
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
	_,err := orm.Table(model).Insert(
		strings.Join(fields,","),		
		values,
	)
	if logger.CheckError(err) {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": err.Error(),
		})
		return
	}

	c.Json(map[string]any{
		"success": "Done !",
	})	
}

var SingleModelGet = func(c *kamux.Context) {
	model,ok := c.Params["model"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "param model not defined",
		})
		return
	}
	id,ok := c.Params["id"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "param id not defined",
		})
		return
	}
	modelRow,err := orm.Table(model).Where("id = ?",id).One()
	if logger.CheckError(err) {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":err.Error(),
		})
		return
	}
	columns := orm.GetAllColumns(model)
	c.Html("admin/admin_single_model.html", map[string]any{
		"model":modelRow,
		"model_name":model,
		"id":id,
		"columns":columns,
	})
}

var UpdateRowPost = func(c *kamux.Context) {
	// parse the form and get data values + files
	data,files := utils.ParseMultipartForm(c.Request)
	// id from string to int
	id := data["row_id"][0]
	//handle file upload
	err := handleFilesUpload(files,data["table"][0],id,c)
	if err != nil {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":err.Error(),
		})
		return
	}

	//get model from database
	modelDB,err := orm.Table(data["table"][0]).Where("id = ?",id).One()
	
	if err != nil {
		c.Status(http.StatusBadRequest).Json(map[string]any{
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
				_,err := orm.Table(data["table"][0]).Where("id = ?",id).Set(key+" = ?",val[0])
				
				if err != nil {
					c.Status(http.StatusBadRequest).Json(map[string]any{
						"error": err.Error(),
					})
					return
				}
				c.Json(map[string]any{
					"success": key + " successfully Updated !",
				})
				return
			}
		default:
			if modelDB[key] != val[0] {
				_,err := orm.Table(data["table"][0]).Where("id = ?",id).Set(key+" = ?",val[0])
				if err != nil {
					c.Json(map[string]any{
						"error":err.Error(),
					})
					return
				}
			}
			c.Json(map[string]any{
				"success": key + " Updated successfully !",
				"data":data,
			})	
			return
		}//switch
	}

	if len(files) > 0 {
		c.Json(map[string]any{
			"success":"Update Done",
		})	
	}
	
}

func handleFilesUpload(files map[string][]*multipart.FileHeader,model string,id string,c *kamux.Context) error {
	if len(files) > 0 {
		for key,val := range files {
			file,_ := val[0].Open()
			defer file.Close()
			uploadedImage,err := utils.UploadMultipartFile(file,val[0].Filename,"media/uploads/")
			if err != nil {
				return err
			}
			row,err := orm.Table(model).Where("id = ?",id).One()
			if err != nil {
				return err
			}
			database_image := row[key]

			if database_image == uploadedImage {
				return errors.New("uploadedImage is the same")
			} else {
				if v,ok := database_image.(string);ok {
					err := c.DeleteFile(v)
					if err != nil {
						//le fichier existe pas
						_,err := orm.Table(model).Where("id = ?",id).Set(key+" = ?",uploadedImage)
						logger.CheckError(err)
						continue
					} else {
						//le fichier existe et donc supprimer
						_,err := orm.Table(model).Where("id = ?",id).Set(key+" = ?",uploadedImage)
						logger.CheckError(err)
						continue
					}
				}
			}
			
		}
	}
	return nil
}

var DropTablePost = func(c *kamux.Context) {
	data := c.BodyJson()
	if table,ok := data["table"];ok && table != ""{
		if t,ok := data["table"].(string);ok {
			_,err := orm.Table(t).Drop()
			if logger.CheckError(err) {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error":err.Error(),
				})
				return
			}
		} else {
			c.Status(http.StatusBadRequest).Json(map[string]any{
				"error":"expecting 'table' to be string",
			})
		}
	} else {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":"missing 'table' in body request",
		})
	}
	c.Json(map[string]any{
		"success":fmt.Sprintf("table %s Deleted !",data["table"]),
	})
}

var ExportView= func(c *kamux.Context) {
	table,ok := c.Params["table"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":"no param table found",
		})
		return
	}
	data,err := orm.Table(table).All()
	logger.CheckError(err)

	data_bytes,err := json.Marshal(data)
	logger.CheckError(err)
	
	c.Download(data_bytes,table+".json")
}

var ImportView= func(c *kamux.Context) {
	// get table name
	table := c.Request.FormValue("table")
	if table == "" {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":"no table !",
		})
		return
	}
	// upload file and return bytes of file
	_,dataBytes,err := c.UploadFile("thefile","backup","json")
	if logger.CheckError(err) {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error":err.Error(),
		})
		return
	}
	// fill list_map
	list_map := []map[string]any{}
	json.Unmarshal(dataBytes,&list_map)
	// create models in database
	cols := []string{}
	values := []any{}
	for _,m := range list_map {
		for k,v := range m {		
			cols = append(cols, k)
			values = append(values, v)
		}
		
		_,_ = orm.Table(table).Insert(strings.Join(cols,","),values)
		cols = cols[:0]
		values = values[:0]
	} 

	c.Json(map[string]any{
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
		c.ServeEmbededFile("application/json; charset=utf-8",f)
	} else {
		c.ServeFile("application/json; charset=utf-8","./assets/static/manifest.json")
	}
}

var ServiceWorkerView = func(c *kamux.Context) {
	if settings.GlobalConfig.EmbedStatic  {
		f,err := kamux.Static.ReadFile("assets/static/sw.js")
		if err != nil {
			logger.Error("cannot embed sw.js from static",err)
			return
		}
		c.ServeEmbededFile("application/javascript; charset=utf-8",f)
	} else {
		c.ServeFile("application/javascript; charset=utf-8","./assets/static/sw.js")
	}
}

var RobotsTxtView = func(c *kamux.Context) {
	c.ServeFile("text/plain; charset=utf-8","./static/robots.txt")
}

var OfflineView= func (c *kamux.Context) {
	c.Text("<h1>YOUR ARE OFFLINE, check connection</h1>")
}


var LogsSSEView = func(c *kamux.Context) {
	lenStream := len(logger.StreamLogs)
	if lenStream > 0 {
		err := c.StreamResponse(logger.StreamLogs[lenStream-1])
		if err != nil{
			logger.Error(err)
		}
	}
}

var LogsGetView = func (c *kamux.Context)  {
	c.Html("admin/logs.html",nil)
}
