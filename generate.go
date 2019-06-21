package main

import (
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"strings"
	"table2domain/util"
)

var db *gorm.DB
var h string
var u string
var p string
var P int
var d string
var t string
var o string

func main()  {
	flag.StringVar(&h, "h", "127.0.0.1", "地址")
	flag.StringVar(&u, "u", "root", "用户名")
	flag.StringVar(&p, "p", "", "密码")
	flag.IntVar(&P, "P", 3306, "端口")
	flag.StringVar(&d, "d", "", "数据库")
	flag.StringVar(&t, "t", "", "数据表")
	flag.StringVar(&o, "o", "./domain", "导出目录")
	flag.Parse()

	dsn := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", u, p, h, P, d)
	db = mysqlDb(dsn)
	defer db.Close()

	var table []string
	if t == "" {
		var err error
		table, err = tables()

		if err != nil {
			log.Println("没有可用表")
			os.Exit(1)
		}
	} else {
		if strings.Index(t, ",") > -1 {
			table = strings.Split(t, ",")
		} else {
			table = []string{t}
		}
	}

	for _, tt := range table {
		info := tableInfo(tt)

		generateStruct(tt, info)
	}
	log.Println("生成完成...")
}

func tables() ([]string, error) {
	rows, err := db.Raw("show tables").Rows()
	defer rows.Close()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var ts []string
	for rows.Next() {
		var table string
		rows.Scan(&table)
		ts = append(ts, table)
	}

	return ts, nil
}

func mysqlDb(dsn string) *gorm.DB {
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		log.Println("数据库连接错误", err)
		os.Exit(100)
	}
	return db
}

func tableInfo(table string) [][]string {
	rows, _ := db.Raw(fmt.Sprintf("DESC %s", table)).Rows()
	var columnInfo [][]string
	for rows.Next() {
		var field string
		var t string
		var isNull string
		var k string
		var df string
		var extra string

		rows.Scan(&field, &t, &isNull, &k, &df, &extra)
		info := []string{field, t, isNull, k, df, extra}
		columnInfo = append(columnInfo, info)
	}
	return columnInfo
}

func generateStruct(tableName string, info [][]string)  {
	builder := strings.Builder{}
	headBuilder := strings.Builder{}
	headBuilder.WriteString("package domain\n\n")
	builder.WriteString(fmt.Sprintf("type %s struct {\n", util.CamelString(tableName)))

	for _, v := range info {
		fieldName := util.CamelString(v[0])
		fieldType := parseType(v[1])
		h := headBuilder.String()
		if strings.Index(fieldType, "time") > -1 && strings.Index(h, "time") < 0 {
			if strings.Index(h, "import") > -1 {
				headBuilder.WriteString(fmt.Sprintf("    \"time\"\n"))
			} else {
				headBuilder.WriteString("import (\n    \"time\"\n")
			}
		}

		builder.WriteString(fmt.Sprintf("    %s %s `json:\"%s\" gorm:\"column:%s\"`\n", fieldName, fieldType, util.SnakeString(v[0]), v[0]))
	}
	builder.WriteString("}")

	// 生成文件
	fileName := fmt.Sprintf("%s%c%s.go", o, os.PathSeparator, util.SnakeString(tableName))
	if _, err := os.Stat(o); err != nil && os.IsNotExist(err) {
		os.MkdirAll(o, os.ModePerm)
	}

	fp, err := os.OpenFile(fileName, os.O_RDWR | os.O_CREATE, 0644)
	defer fp.Close()
	if err != nil {
		log.Println("文件生成失败", err)
		os.Exit(2)
	}
	if strings.Index(headBuilder.String(), "import") > -1 {
		headBuilder.WriteString(")\n\n")
	}
	fp.WriteString(headBuilder.String()+builder.String())
}

func parseType(dbType string) string {
	dbType = strings.ToUpper(dbType)

	// 字符串
	stringType := []string{"CHAR", "VARCHAR", "TEXT", "LONGTEXT", "BLOB", "LONGBLOB"}
	for _, v := range stringType {
		if strings.Index(dbType, v) > -1 {
			return "string"
		}
	}

	// 日期
	dateType := []string{"DATE", "DATETIME", "TIMESTAMP"}
	for _, v := range dateType {
		if strings.Index(dbType, v) > -1 {
			return "time.Time"
		}
	}

	// 整形
	if strings.Index(dbType, "TINYINT") > -1 {
		return "int8"
	}
	if strings.Index(dbType, "SMALLINT") > -1 {
		return "int8"
	}
	if strings.Index(dbType, "MEDIUMINT") > -1 {
		return "int16"
	}
	intType := []string{"INT", "INTEGER"}
	for _, v := range intType {
		if strings.Index(dbType, v) > -1 {
			return "int"
		}
	}
	if strings.Index(dbType, "BIGINT") > -1 {
		return "int64"
	}

	// 符点
	if strings.Index(dbType, "FLOAT") > -1 {
		return "float32"
	}
	float64Type := []string{"DOUBLE", "DECIMAL", "NUMERIC"}
	for _, v := range float64Type {
		if strings.Index(dbType, v) > -1 {
			return "float64"
		}
	}

	return "string"
}


