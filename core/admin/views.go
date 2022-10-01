package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/encryptor"
	"github.com/kamalshkeir/kago/core/utils/encryption/hash"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

var PAGINATION_PER = 6

var IndexView = func(c *kamux.Context) {
	allTables := orm.GetAllTables()
	c.Html("admin/admin_index.html", map[string]any{
		"tables": allTables,
	})
}

var LoginView = func(c *kamux.Context) {
	c.Html("admin/admin_login.html", nil)
}

var LoginPOSTView = func(c *kamux.Context) {
	requestData := c.BodyJson()
	email := requestData["email"]
	passRequest := requestData["password"]

	data, err := orm.Table("users").Where("email = ?", email).One()
	if err != nil {
		c.Status(500).Json(map[string]any{
			"error": err.Error(),
		})
		return
	}
	if data["email"] == "" || data["email"] == nil {
		c.Status(http.StatusNotFound).Json(map[string]any{
			"error": "User doesn not Exist",
		})
		return
	}
	if data["is_admin"] == int64(0) || data["is_admin"] == 0 || data["is_admin"] == false {
		c.Status(http.StatusForbidden).Json(map[string]any{
			"error": "Not Allowed to access this page",
		})
		return
	}

	if passDB, ok := data["password"].(string); ok {
		if pp, ok := passRequest.(string); ok {
			match, err := hash.ComparePasswordToHash(pp, passDB)
			if !match || err != nil {
				c.Status(http.StatusForbidden).Json(map[string]any{
					"error": "Wrong Password",
				})
				return
			} else {
				if uuid, ok := data["uuid"].(string); ok {
					if kamux.SESSION_ENCRYPTION {
						uuid, err = encryptor.Encrypt(uuid)
						logger.CheckError(err)
					}
					c.SetCookie("session", uuid)
					c.Json(map[string]any{
						"success": "U Are Logged In",
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
	model, ok := c.Params["model"]
	if !ok {
		c.Json(map[string]any{
			"error": "Error: No model given in params",
		})
		return
	}
	idString := "id"
	t, _ := orm.GetMemoryTable(model,orm.DefaultDB)
	if t.Pk != "" && t.Pk != "id" {
		idString = t.Pk
	}
	rows, err := orm.Table(model).OrderBy("-" + idString).Limit(PAGINATION_PER).Page(1).All()
	if err != nil {
		rows, err = orm.Table(model).All()
		if err != nil {
			// usualy should not use error string because it divulge information, but here only admin use it, so no worry
			if err.Error() != "no data found" {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error": err.Error(),
				})
				return
			}
		}
	}
	dbCols := orm.GetAllColumnsTypes(model,orm.DefaultDB)
	if settings.Config.Db.Type != "" {
		c.Html("admin/admin_all_models.html", map[string]any{
			"dbType":     settings.Config.Db.Type,
			"model_name": model,
			"rows":       rows,
			"columns":    t.ModelTypes,
			"dbcolumns":  dbCols,
			"pk":         t.Pk,
			"columnsOrdered":t.Columns,
		})
	} else {
		logger.Error("dbType not known, do you have .env", settings.Config.Db.Type, err)
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "dbType not found",
		})
	}
}

var AllModelsSearch = func(c *kamux.Context) {
	model, ok := c.Params["model"]
	if !ok {
		c.Json(map[string]any{
			"error": "Error: No model given in params",
		})
		return
	}

	body := c.BodyJson()
	
	oB := ""
	t, _ := orm.GetMemoryTable(model,orm.DefaultDB)
	if orderby,ok := body["orderby"];ok {
		if v,ok := orderby.(string);ok {
			oB=v
		} 
	} 
	if oB == "" && t.Pk != ""{
		oB="-"+t.Pk
	}
	if query,ok := body["query"];ok {
		blder := orm.Table(model).Where(query.(string))
		if oB != "" {
			blder.OrderBy(oB)
		} 
		data,err := blder.Limit(PAGINATION_PER).Page(1).All()
		if logger.CheckError(err) {
			c.Json(map[string]any{
				"error":err.Error(),
			})
			return
		}
		c.Json(map[string]any{
			"rows":data,
			"cols":t.Columns,
		})
		return
	}
	c.SetStatus(400)
}

var AllModelsPost = func(c *kamux.Context) {
	model, ok := c.Params["model"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "No model given in params",
		})
		return
	}
	received := c.BodyJson()
	if received != nil {
		if v, ok := received["page_num"]; ok {
			if page, ok := v.(string); !ok {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error": "expecting page_num to be a sring",
				})
				return
			} else {
				pagenum, err := strconv.Atoi(page)
				if err == nil {
					idString := "id"
					t, _ := orm.GetMemoryTable(model,orm.DefaultDB)
					if t.Pk != "" && t.Pk != "id" {
						idString = t.Pk
					}
					orderBY := "-" + idString
					if orderby,ok := received["orderby"];ok {
						orderBY=orderby.(string)
					}
					rows, err := orm.Table(model).OrderBy(orderBY).Limit(PAGINATION_PER).Page(pagenum).All()
					if err == nil {
						c.Json(map[string]any{
							"rows": rows,
							"cols":t.Columns,
						})
					}
				}
			}
		} else {
			logger.Error("page_num not given", received)
		}
	} else {
		c.Json([]map[string]any{})
	}
}

var DeleteRowPost = func(c *kamux.Context) {
	data := c.BodyJson()
	if data["mission"] == "delete_row" {
		if model, ok := data["model_name"]; ok {
			if mm, ok := model.(string); ok {
				idString := "id"
				t, _ := orm.GetMemoryTable(mm,orm.DefaultDB)
				if t.Pk != "" && t.Pk != "id" {
					idString = t.Pk
				}
				modelDB, err := orm.Table(mm).Where(idString+" = ?", data["id"]).One()
				if logger.CheckError(err) {
					logger.Info("data received DeleteRowPost:", data)
					c.Status(http.StatusBadRequest).Json(map[string]any{
						"error": err.Error(),
					})
					return
				}
				if val, ok := modelDB["image"]; ok {
					if vv, ok := val.(string); ok && vv != "" {
						_ = c.DeleteFile(vv)
					}
				}

				if idS, ok := data["id"].(string); ok {
					_, err = orm.Table(mm).Where(idString+" = ?", idS).Delete()

					if err != nil {
						c.Status(http.StatusBadRequest).Json(map[string]any{
							"error": err.Error(),
						})
					} else {
						c.Json(map[string]any{
							"success": "Done !",
							"id":      data["id"],
						})
						return
					}
				}
			} else {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error": "expecting model_name to be string",
				})
				return
			}
		} else {
			c.Status(http.StatusBadRequest).Json(map[string]any{
				"error": "no model_name found in request body",
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

	defer func() {
		err := c.Request.MultipartForm.RemoveAll()
		logger.CheckError(err)
	}()

	model := data["table"][0]

	fields := []string{}
	values := []any{}
	for key, val := range data {
		switch key {
		case "table":
			continue
		case "uuid":
			uuid, err := utils.GenerateUUID()
			logger.CheckError(err)
			fields = append(fields, key)
			values = append(values, uuid)
		case "password":
			hash, _ := hash.GenerateHash(val[0])
			fields = append(fields, key)
			values = append(values, hash)
		case "":
		default:
			if key != "" && val[0] != "" && val[0] != "null" {
				fields = append(fields, key)
				values = append(values, val[0])
			} 
		}
	}

	_, err := orm.Table(model).Insert(
		strings.Join(fields, ","),
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
	model, ok := c.Params["model"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "param model not defined",
		})
		return
	}
	id, ok := c.Params["id"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "param id not defined",
		})
		return
	}
	idString := "id"
	t, _ := orm.GetMemoryTable(model,orm.DefaultDB)
	if t.Pk != "" && t.Pk != "id" {
		idString = t.Pk
	}

	modelRow, err := orm.Table(model).Where(idString+" = ?", id).One()
	if logger.CheckError(err) {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": err.Error(),
		})
		return
	}
	dbCols := orm.GetAllColumnsTypes(model, orm.DefaultDB)
	c.Html("admin/admin_single_model.html", map[string]any{
		"model":      modelRow,
		"model_name": model,
		"id":         id,
		"columns":    t.ModelTypes,
		"dbcolumns":  dbCols,
		"pk":         t.Pk,
	})
}

var UpdateRowPost = func(c *kamux.Context) {
	// parse the form and get data values + files
	data, files := utils.ParseMultipartForm(c.Request)
	// id from string to int
	id := data["row_id"][0]
	//handle file upload
	//get model from database
	idString := "id"
	t, _ := orm.GetMemoryTable(data["table"][0],orm.DefaultDB)
	if t.Pk != "" && t.Pk != "id" {
		idString = t.Pk
	}
	err := handleFilesUpload(files, data["table"][0], id, c, idString)
	if err != nil {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": err.Error(),
		})
		return
	}

	modelDB, err := orm.Table(data["table"][0]).Where(idString+" = ?", id).One()

	if err != nil {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": err.Error(),
		})
		return
	}

	ignored := []string{idString, "uuid", "file", "image", "photo", "img", "fichier", "row_id", "table"}
	toUpdate := map[string]any{}
	for key, val := range data {
		if !utils.SliceContains(ignored, key) {
			if modelDB[key] == val[0] {
				// no changes for bool
				continue
			}
			toUpdate[key] = val[0]
		}
	}

	s := ""
	values := []any{}
	if len(toUpdate) > 0 {
		for col, v := range toUpdate {
			if s == "" {
				s += col + "= ?"
			} else {
				s += "," + col + "= ?"
			}
			values = append(values, v)
		}
	}
	if s != "" {
		_, err := orm.Table(data["table"][0]).Where(idString+" = ?", id).Set(s, values...)
		if err != nil {
			c.Status(http.StatusBadRequest).Json(map[string]any{
				"error": err.Error(),
			})
			return
		}
	}
	s = ""
	if len(files) > 0 {
		for f := range files {
			if s == "" {
				s += f
			} else {
				s += "," + f
			}
		}
	}
	if len(toUpdate) > 0 {
		for k := range toUpdate {
			if s == "" {
				s += k
			} else {
				s += "," + k
			}
		}
	}
	c.Json(map[string]string{
		"success": s + " updated successfully",
	})
}

func handleFilesUpload(files map[string][]*multipart.FileHeader, model string, id string, c *kamux.Context, idString string) error {
	if len(files) > 0 {
		for key, val := range files {
			file, _ := val[0].Open()
			defer file.Close()
			uploadedImage, err := utils.UploadMultipartFile(file, val[0].Filename, settings.MEDIA_DIR+"/uploads/")
			if err != nil {
				return err
			}
			row, err := orm.Table(model).Where(idString+" = ?", id).One()
			if err != nil {
				return err
			}
			database_image := row[key]

			if database_image == uploadedImage {
				return errors.New("uploadedImage is the same")
			} else {
				if v, ok := database_image.(string); ok {
					err := c.DeleteFile(v)
					if err != nil {
						//le fichier existe pas
						_, err := orm.Table(model).Where(idString+" = ?", id).Set(key+" = ?", uploadedImage)
						logger.CheckError(err)
						continue
					} else {
						//le fichier existe et donc supprimer
						_, err := orm.Table(model).Where(idString+" = ?", id).Set(key+" = ?", uploadedImage)
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
	if table, ok := data["table"]; ok && table != "" {
		if t, ok := data["table"].(string); ok {
			_, err := orm.Table(t).Drop()
			if logger.CheckError(err) {
				c.Status(http.StatusBadRequest).Json(map[string]any{
					"error": err.Error(),
				})
				return
			}
		} else {
			c.Status(http.StatusBadRequest).Json(map[string]any{
				"error": "expecting 'table' to be string",
			})
		}
	} else {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "missing 'table' in body request",
		})
	}
	c.Json(map[string]any{
		"success": fmt.Sprintf("table %s Deleted !", data["table"]),
	})
}

var ExportView = func(c *kamux.Context) {
	table, ok := c.Params["table"]
	if !ok {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "no param table found",
		})
		return
	}
	data, err := orm.Table(table).All()
	logger.CheckError(err)

	data_bytes, err := json.Marshal(data)
	logger.CheckError(err)

	c.Download(data_bytes, table+".json")
}

var ImportView = func(c *kamux.Context) {
	// get table name
	table := c.Request.FormValue("table")
	if table == "" {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": "no table !",
		})
		return
	}
	// upload file and return bytes of file
	_, dataBytes, err := c.UploadFile("thefile", "backup", "json")
	if logger.CheckError(err) {
		c.Status(http.StatusBadRequest).Json(map[string]any{
			"error": err.Error(),
		})
		return
	}

	// get old data and backup
	modelsOld,_ := orm.Table(table).All()
	if len(modelsOld) > 0 {
		modelsOldBytes,err := json.Marshal(modelsOld)
		if !logger.CheckError(err) {
			_ = os.MkdirAll(settings.MEDIA_DIR+"/backup/", 0664)
			dst, err := os.Create(settings.MEDIA_DIR + "/backup/" +table+"-"+ time.Now().Format("2006-01-02")+".json")
			logger.CheckError(err)
			defer dst.Close()
			_,err = dst.Write(modelsOldBytes)
			logger.CheckError(err)
		}
	}
	

	// fill list_map
	list_map := []map[string]any{}
	json.Unmarshal(dataBytes, &list_map)
	// create models in database
	cols := []string{}
	values := []any{}
	for _, m := range list_map {
		for k, v := range m {
			cols = append(cols, k)
			values = append(values, v)
		}
		_, _ = orm.Table(table).Insert(strings.Join(cols, ","), values)
		cols = cols[:0]
		values = values[:0]
	}

	c.Json(map[string]any{
		"success": fmt.Sprintf("Import Done , you can see uploaded backups at ./%s/backup folder",settings.MEDIA_DIR),
	})
}

var ManifestView = func(c *kamux.Context) {
	if settings.Config.Embed.Static {
		f, err := kamux.Static.ReadFile(settings.STATIC_DIR + "/manifest.json")
		if err != nil {
			logger.Error("cannot embed manifest.json from static", err)
			return
		}
		c.ServeEmbededFile("application/json; charset=utf-8", f)
	} else {
		c.ServeFile("application/json; charset=utf-8", settings.STATIC_DIR+"/manifest.json")
	}
}

var ServiceWorkerView = func(c *kamux.Context) {
	if settings.Config.Embed.Static {
		f, err := kamux.Static.ReadFile(settings.STATIC_DIR + "/sw.js")
		if err != nil {
			logger.Error("cannot embed sw.js from static", err)
			return
		}
		c.ServeEmbededFile("application/javascript; charset=utf-8", f)
	} else {
		c.ServeFile("application/javascript; charset=utf-8", settings.STATIC_DIR+"/sw.js")
	}
}

var RobotsTxtView = func(c *kamux.Context) {
	c.ServeFile("text/plain; charset=utf-8", "./static/robots.txt")
}

var OfflineView = func(c *kamux.Context) {
	c.Text("<h1>YOUR ARE OFFLINE, check connection</h1>")
}

var LogsSSEView = func(c *kamux.Context) {
	lenStream := len(logger.StreamLogs)
	if lenStream > 0 {
		err := c.StreamResponse(logger.StreamLogs[lenStream-1])
		if err != nil {
			logger.Error(err)
		}
		if lenStream > 2 {
			err := c.StreamResponse(logger.StreamLogs[lenStream-2])
			if err != nil {
				logger.Error(err)
			}
		} else if lenStream > 50 {
			err := c.StreamResponse(logger.StreamLogs[lenStream-2])
			if err != nil {
				logger.Error(err)
			}
			logger.StreamLogs = []string{}
		}
	}
}

var LogsGetView = func(c *kamux.Context) {
	c.Html("admin/logs.html", nil)
}
