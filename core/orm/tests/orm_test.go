package tests

import (
	"testing"
	"time"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

const otherDB = "db2"

func init() {
	// load env
	r := kamux.Router{}
	r.LoadEnv("../../../.env")
	// init 'db'
	err := orm.InitDB()
	if logger.CheckError(err) {
		return
	}
	//migrate 'db'
	err = orm.Migrate()
	if logger.CheckError(err) {
		return
	}
	// init another db named 'db2'
	err = orm.NewDatabaseFromDSN(orm.SQLITE, otherDB, "")
	if logger.CheckError(err) {
		return
	}
	// migrate user to 'db2'
	err = orm.AutoMigrate[models.User]("users", otherDB)
	if logger.CheckError(err) {
		return
	}
	// create user if not exist in database 'db'
	users, _ := orm.Table("users").All()
	if len(users) == 0 {
		orm.CreateUser("kamal@gmail.com", "olaola", 1)
	}
	// create user if not exist in database 'db'
	userss, _ := orm.Table("users").Database(otherDB).All()

	if len(userss) == 0 {
		uu, _ := orm.GenerateUUID()
		values := []any{uu, "db2@gmail.com", "olaola", 1}
		_, err := orm.Table("users").Database(otherDB).Insert("uuid,email,password,is_admin", values)
		if logger.CheckError(err) {
			return
		}
	}
}

func TestInitDB(t *testing.T) {
	// load env
	r := kamux.Router{}
	r.LoadEnv("../../../.env")
	// init db
	err := orm.InitDB()
	if err != nil {
		t.Error(err)
	}
	conn := orm.GetConnection()
	if conn == nil {
		t.Error("connection not found after initDB")
	}
}

func TestNewDatabaseFromDSN(t *testing.T) {
	err := orm.NewDatabaseFromDSN("sqlite", otherDB, "")
	if err == nil {
		t.Error("should error because", otherDB, "registered before")
		return
	}
	databases := orm.GetMemoryDatabases()
	if len(databases) == 0 {
		t.Error("no database found")
		return
	}
	conn := orm.GetConnection(otherDB)
	if conn == nil {
		t.Error("no connection found for", otherDB)
	}
}

func TestStructAll(t *testing.T) {
	users, err := orm.Model[models.User]().All()
	if err != nil {
		t.Error(err)
		return
	}
	if len(users) == 0 {
		t.Error("no data found")
	}
}

func TestGetConnection(t *testing.T) {
	conn := orm.GetConnection(settings.Config.Db.Name)
	if conn == nil {
		t.Error("no connection found")
	}
}

func TestGetDatabases(t *testing.T) {
	dbs := orm.GetMemoryDatabases()
	if len(dbs) < 1 {
		t.Error("no database found")
	}
}

func TestGetAllTables(t *testing.T) {
	tables := orm.GetAllTables()
	if len(tables) == 0 {
		t.Error("no table found:", len(tables))
	}
}

func TestGetAllColumns(t *testing.T) {
	colsTypes := orm.GetAllColumnsTypes("users")
	if len(colsTypes) == 0 {
		t.Error("no column found:", colsTypes)
	}
}

func TestDatabaseS(t *testing.T) {
	email := "db2@gmail.com"
	u, err := orm.Model[models.User]().Database(otherDB).Where("email = ?", email).One()
	if err != nil {
		t.Error(err)
		return
	}
	if u.Email != email {
		t.Error("user email different then", email)
	}
}

func TestDatabaseM(t *testing.T) {
	email := "db2@gmail.com"
	u, err := orm.Table("users").Database(otherDB).Where("email = ?", email).One()
	if err != nil {
		t.Error(err)
		return
	}
	if v, ok := u["email"]; ok {
		if v != email {
			t.Error("user email different then", email)
		}
	}
}

type TestModel struct {
	Id        uint      `orm:"autoinc"`
	Content   string    `orm:"size:200;default:''"`
	UserId    int       `orm:"fk:users.id:cascade"`
	CreatedAt time.Time `orm:"now"`
}

func TestAutoMigrate(t *testing.T) {
	err := orm.AutoMigrate[TestModel]("test_model", otherDB)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestCheckAutoMigrate(t *testing.T) {
	// get or create first row
	testModel, err := orm.Model[TestModel]().Database(otherDB).Limit(1).One()
	if err != nil {
		// should work
		_, err := orm.Model[TestModel]().Database(otherDB).Insert(&TestModel{
			Content:   "test",
			UserId:    1,
			CreatedAt: time.Now(),
		})
		if err != nil {
			t.Error(err)
			return
		}
		// should fail
		_, err = orm.Model[TestModel]().Database(otherDB).Insert(&TestModel{
			Content:   "test",
			UserId:    10,
			CreatedAt: time.Now(),
		})
		if err == nil {
			t.Error("foreign_key not working")
			return
		}
	}

	if testModel == (TestModel{}) {
		t.Error("test model is empty")
		return
	}

	if testModel.CreatedAt == (time.Time{}) {
		t.Error("created_at is empty")
		return
	}
	if testModel.UserId != 1 {
		t.Error("user_id doesn't match")
		return
	}
}
