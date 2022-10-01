package benchmarks

import (
	"testing"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

func init() {
	r := kamux.Router{}
	r.LoadEnv("../../../.env")
	_ = orm.InitDB()
	orm.UseCache = true
	//migrate
	err := orm.Migrate()
	if logger.CheckError(err) {
		return
	}
	users, _ := orm.Table("users").All()
	if len(users) == 0 {
		orm.CreateUser("kamal@gmail.com", "olaola", 1)
	}
}

/*
////////////////////////////////////// postgres without cache
BenchmarkGetAllS-4                  1472            723428 ns/op            5271 B/op         80 allocs/op
BenchmarkGetAllM-4                  1502            716418 ns/op            4912 B/op         85 allocs/op
BenchmarkGetRowS-4                   826           1474674 ns/op            2288 B/op         44 allocs/op
BenchmarkGetRowM-4                   848           1392919 ns/op            2216 B/op         44 allocs/op
BenchmarkGetAllTables-4             1176            940142 ns/op             592 B/op         20 allocs/op
BenchmarkGetAllColumns-4             417           2862546 ns/op            1456 B/op         46 allocs/op
////////////////////////////////////// postgres with cache
BenchmarkGetAllS-4               2825896               427.9 ns/op           208 B/op          2 allocs/op
BenchmarkGetAllM-4               6209617               188.9 ns/op            16 B/op          1 allocs/op
BenchmarkGetRowS-4               2191544               528.1 ns/op           240 B/op          4 allocs/op
BenchmarkGetRowM-4               3799377               305.5 ns/op            48 B/op          3 allocs/op
BenchmarkGetAllTables-4         76298504                21.41 ns/op            0 B/op          0 allocs/op
BenchmarkGetAllColumns-4        59004012                19.92 ns/op            0 B/op          0 allocs/op
///////////////////////////////////// mysql without cache
BenchmarkGetAllS-4                  1221            865469 ns/op            7152 B/op        162 allocs/op
BenchmarkGetAllM-4                  1484            843395 ns/op            8272 B/op        215 allocs/op
BenchmarkGetRowS-4                   427           3539007 ns/op            2368 B/op         48 allocs/op
BenchmarkGetRowM-4                   267           4481279 ns/op            2512 B/op         54 allocs/op
BenchmarkGetAllTables-4              771           1700035 ns/op             832 B/op         26 allocs/op
BenchmarkGetAllColumns-4             760           1537301 ns/op            1392 B/op         44 allocs/op
///////////////////////////////////// mysql with cache
BenchmarkGetAllS-4               2933072               414.5 ns/op           208 B/op          2 allocs/op
BenchmarkGetAllM-4               6704588               180.4 ns/op            16 B/op          1 allocs/op
BenchmarkGetRowS-4               2136634               545.4 ns/op           240 B/op          4 allocs/op
BenchmarkGetRowM-4               4111814               292.6 ns/op            48 B/op          3 allocs/op
BenchmarkGetAllTables-4         58835394                21.52 ns/op            0 B/op          0 allocs/op
BenchmarkGetAllColumns-4        59059225                19.99 ns/op            0 B/op          0 allocs/op
///////////////////////////////////// sqlite without cache
BenchmarkGetAllS-4                 13664             85506 ns/op            2056 B/op         62 allocs/op
BenchmarkGetAllS_GORM-4            10000            101665 ns/op            9547 B/op        155 allocs/op
BenchmarkGetAllM-4                 13747             83989 ns/op            1912 B/op         61 allocs/op
BenchmarkGetAllM_GORM-4            10000            107810 ns/op            8387 B/op        237 allocs/op
BenchmarkGetRowS-4                 12702             91958 ns/op            2192 B/op         67 allocs/op
BenchmarkGetRowM-4                 13256             89095 ns/op            2048 B/op         66 allocs/op
BenchmarkGetAllTables-4            14264             83939 ns/op             672 B/op         32 allocs/op
BenchmarkGetAllColumns-4           15236             79498 ns/op            1760 B/op         99 allocs/op
///////////////////////////////////// sqlite with cache
BenchmarkGetAllS-4               2951642               399.5 ns/op           208 B/op          2 allocs/op
BenchmarkGetAllM-4               6537204               177.2 ns/op            16 B/op          1 allocs/op
BenchmarkGetRowS-4               2248524               531.4 ns/op           240 B/op          4 allocs/op
BenchmarkGetRowM-4               4084453               287.9 ns/op            48 B/op          3 allocs/op
BenchmarkGetAllTables-4         52592826                20.39 ns/op            0 B/op          0 allocs/op
BenchmarkGetAllColumns-4        64293176                20.87 ns/op            0 B/op          0 allocs/op
*/

func BenchmarkGetAllS(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := orm.Model[models.User]().All()
		if err != nil {
			b.Error("error BenchmarkGetAllS:", err)
		}
	}
}

func BenchmarkGetAllM(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := orm.Table("users").All()
		if err != nil {
			b.Error("error BenchmarkGetAllM:", err)
		}
	}
}

func BenchmarkGetRowS(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := orm.Model[models.User]().Where("email = ?", "kamal@gmail.com").One()
		if err != nil {
			b.Error("error BenchmarkGetRowS:", err)
		}
	}
}

func BenchmarkGetRowM(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := orm.Table("users").Where("email = ?", "kamal@gmail.com").One()
		if err != nil {
			b.Error("error BenchmarkGetRowM:", err)
		}
	}
}

func BenchmarkGetAllTables(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := orm.GetAllTables()
		if len(t) == 0 {
			b.Error("error BenchmarkGetAllTables: no data")
		}
	}
}

func BenchmarkGetAllColumns(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := orm.GetAllColumnsTypes("users")
		if len(c) == 0 {
			b.Error("error BenchmarkGetAllColumns: no data")
		}
	}
}
