package markdown

import (
	"github.com/NiuStar/Basic"
	"github.com/NiuStar/markdown"
	"github.com/NiuStar/log"
	reflect2 "github.com/NiuStar/reflect"
	"github.com/NiuStar/xsql4/Type"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

//写入API接口文档
type API struct {
	name          string //method name
	router        string //import path
	interfaceName string
	overwrite     bool
	result        interface{}
	error         interface{}
	note          string
	request       Type.IHandler
}

type APIMD struct {
	list map[string][]*API
}

func (apis *APIMD) Add(router string, request Type.IHandler, method func() interface{}, result interface{}, err interface{}, note string) {

	if apis.list == nil {
		apis.list = make(map[string][]*API)
	}

	a := &API{}

	fn := runtime.FuncForPC(reflect.ValueOf(method).Pointer()).Name()

	a.name = fn[strings.LastIndex(fn, ".")+1 : strings.LastIndex(fn, "-")]

	a.interfaceName = fn[strings.LastIndex(fn, "(")+1 : strings.LastIndex(fn, ")")]

	a.router = router
	//a.path = fn[:strings.Index(fn,".")][:strings.LastIndex(fn,"/")]

	a.overwrite = false

	a.request = request

	a.result = result

	a.error = err

	a.note = note

	key := fn[strings.LastIndex(fn, "/")+1 : strings.Index(fn, ".")]
	apis.list[key] = append(apis.list[key], a)

}

func (apis *APIMD) Write(mainRouter string) {

	defer func() {
		if r := recover(); r != nil {
			log.Warnln("捕获到的错误：%s\n", r)
		}
		return
	}()
	dir, _ := os.Getwd()
	mk := markdown.NewMarkDown("api", dir)
	var apiIndex int64 = 0
	for _, values := range apis.list {

		if len(values) <= 0 {
			continue
		}
		//fmt.Println("value:",values[0].path)
		for _, api := range values {

			if api.overwrite {
				continue
			}

			//fmt.Println("" + api.interfaceName + ") " + api.name + "() interface{} {")

			//fmt.Println("*QRCode) Login() interface{} {:",strings.Index(code," *QRCode) Login() interface{} {"))

			//fmt.Println("index 1:",index)
			{
				api.overwrite = true

				{

					note := api.note
					//fmt.Println("note:",note)
					apiIndex++

					mk.WriteTitle(3, strconv.FormatInt(apiIndex, 10)+"."+" "+api.name+"  "+note+"\r\n")

					mk.WriteCode("URL:  "+Basic.GetServerConfig().ServerConfig.Domain+"/"+mainRouter+"/"+api.router+"/"+api.name+"\r\n\r\n", "go")
					mk.WriteCode("URL:  "+Basic.GetServerConfig().ServerConfig.Domain+"/"+mainRouter+"?subject="+api.router+"&action="+api.name+"\r\n\r\n", "go")
				}

				{
					mk.WriteContent("\r\n请求参数：\r\n")
					var params [][]string
					{
						var param []string
						param = append(param, "参数名")
						param = append(param, "类型")
						param = append(param, "备注")
						params = append(params, param)
					}

					//fmt.Println("params_list:",params_list)
					{

						_type := reflect2.GetReflectType(api.request)
						//_value := reflect2.GetReflectValue(api.request)

						for i := 0; i < _type.NumField(); i++ {
							jsonName := _type.Field(i).Name
							if []byte(jsonName)[0] < 'A' || []byte(jsonName)[0] > 'Z' {
								continue
							}
							if _type.Field(i).Tag.Get("json") != "" {
								jsonName = _type.Field(i).Tag.Get("json")
							}

							//fmt.Println("jsonName:",jsonName)
							var param []string
							param = append(param, jsonName)
							param = append(param, _type.Field(i).Type.String())
							param = append(param, _type.Field(i).Tag.Get("comment"))
							params = append(params, param)

						}

						mk.WriteForm(params)
					}
					//mk.WriteContent("\r\n空\r\n")

				}

				{
					mk.WriteContent("\r\n返回值：\r\n")
					var params [][]string
					{
						var param []string
						param = append(param, "参数名")
						param = append(param, "类型")
						param = append(param, "备注")
						params = append(params, param)
					}
					if api.result != nil {
						params = append(params, writeResult(reflect2.GetReflectValue(api.result), "")...)
					}

					mk.WriteForm(params)
				}

			}
		}
	}
	mk.Save()
}

func writeResult(result reflect.Value, preStr string) (params [][]string) {

	if !result.IsValid() {
		return
	}
	_type := result.Type()
	_value := result

	if _type.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < _type.NumField(); i++ {
		jsonName := _type.Field(i).Name
		if []byte(jsonName)[0] < 'A' || []byte(jsonName)[0] > 'Z' {
			continue
		}
		if _type.Field(i).Tag.Get("json") != "" {
			jsonName = _type.Field(i).Tag.Get("json")
		}

		if jsonName == _value.Field(i).Type().Name() {
			continue
		}

		var param []string
		param = append(param, preStr+jsonName)
		param = append(param, _value.Field(i).Type().Name())
		param = append(param, _type.Field(i).Tag.Get("comment"))
		params = append(params, param)

		parType := _type.Field(i).Type
		for parType.Kind() == reflect.Ptr {

			parType = parType.Elem()
		}

		if !Type.IsTabelType(parType) {

			//fmt.Println("parType.Kind():",parType.Kind())
			if parType.Kind() == reflect.Struct {
				params = append(params, writeResult(_value.Field(i), preStr+"&nbsp;&nbsp;&nbsp;&nbsp;")...)
			} else if parType.Kind() == reflect.Slice {
				//fmt.Println("result type: reflect.Slice",_value.Field(i).CanSet())
				//_value.Field(i).SetLen(1)
				//v.SetCap(2)
				//fmt.Println("result type: reflect.Slice",_value.Field(i).Slice(0,0))
				/*for j := 0; j < _value.Field(i).NumField(); j++ {
					fmt.Printf("result type: Field %d: %v\n", i, _value.Field(i).Field(j).Type())
				}*/

				//	fmt.Println("result type: reflect.Slice")
				//params = append(params,writeResult(_value.Field(i),preStr + "&nbsp;&nbsp;&nbsp;&nbsp;")...)
			}
		}
	}
	return params
}
