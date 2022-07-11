package tests

import (
	"testing"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/kamux"
	"github.com/kamalshkeir/kago/core/korm"
	"github.com/kamalshkeir/kago/core/utils/logger"
)


func init() {
	r := kamux.Router{}
	r.LoadEnv("../../../.env")
	_ = korm.InitDB()
	//migrate
	err := korm.Migrate()
	if logger.CheckError(err) {return}
	users,_ := korm.Database().Table("users").All()
	if len(users) ==0 {
		korm.CreateUser("kago@gmail.com","olaola",1)
	}
	korm.LinkModel[models.User]("users")
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
BenchmarkGetAllS-4                 13549             84700 ns/op            1992 B/op         61 allocs/op
BenchmarkGetAllM-4                 14166             79705 ns/op            1896 B/op         60 allocs/op
BenchmarkGetRowS-4                 12775             88451 ns/op            2272 B/op         71 allocs/op
BenchmarkGetRowM-4                 13924             85723 ns/op            2176 B/op         70 allocs/op
BenchmarkGetAllTables-4            14155             83979 ns/op             528 B/op         25 allocs/op
BenchmarkGetAllColumns-4           15121             77602 ns/op            1760 B/op         99 allocs/op
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
		_,err := korm.Model[models.User]().All()
		if err != nil {
			b.Error("error BenchmarkGetAllS:",err)
		}
	}
}

func BenchmarkGetAllM(b *testing.B) {	
	b.ReportAllocs()
    b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_,err := korm.Database().Table("users").All()
		if err != nil {
			b.Error("error BenchmarkGetAllM:",err)
		}
	}
}


func BenchmarkGetRowS(b *testing.B) {	
	b.ReportAllocs()
    b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_,err := korm.Model[models.User]().Where("email = ?","kago@gmail.com").One()
		if err != nil {
			b.Error("error BenchmarkGetRowS:",err)
		}
	}
}

func BenchmarkGetRowM(b *testing.B) {	
	b.ReportAllocs()
    b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_,err := korm.Database().Table("users").Where("email = ?","kago@gmail.com").One()
		if err != nil {
			b.Error("error BenchmarkGetRowM:",err)
		}
	}
}



func BenchmarkGetAllTables(b *testing.B) {
	b.ReportAllocs()
    b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := korm.GetAllTables()
		if len(t) == 0 {
			b.Error("error BenchmarkGetAllTables: no data",)
		}
	}
}

func BenchmarkGetAllColumns(b *testing.B) {
	b.ReportAllocs()
    b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := korm.GetAllColumns("users")
		if len(c) == 0 {
			b.Error("error BenchmarkGetAllColumns: no data",)
		}
	}
}




