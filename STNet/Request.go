package STNet

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"github.com/NiuStar/NiuServer/markdown"
	"github.com/NiuStar/log"
	. "github.com/NiuStar/reflect"
	. "github.com/NiuStar/xsql4/Type"
	"github.com/gin-gonic/gin"
	"io"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

var TAG = "<REQUEST>"
var apis = markdown.APIMD{}

const ContextName = "Context"
const ResponseTypeName = "ResponseType"
const BODY = "Body"
const SUBJECT = "subject"
const ACTION = "action"

type NServer struct {
	routers    map[string]map[string]*reqAndResponse
	engine     *gin.Engine
	mainRouter string
}

func newServer() *NServer {
	ser := &NServer{}
	ser.routers = make(map[string]map[string]*reqAndResponse)
	ser1 := gin.Default()
	ser1.Static("/View", "./View")

	ser.engine = ser1
	return ser
}

var (
	gServer *NServer
)

//初始化;
func init() {
	gServer = newServer()
}

func DefaultServer() *NServer {
	return gServer
}

func (ser *NServer) GetEngine() *gin.Engine {
	return ser.engine
}

func (ser *NServer) Run(mainRouter string, address ...string) {
	registerOver(mainRouter)
	ser.mainRouter = "/" + mainRouter
	//ser1.POST("device",)
	ser.engine.Use(ser.RequestRouter())
	log.Init(false)
	log.Info(TAG, fmt.Sprintf("Server with pid %d. start successful ,Listening Serving on:", os.Getpid()), address)
	ser.engine.Run(address...)
}

//要求返回值结构体里必须有这个，才可以自动解析出给客户端的数据格式
type IResult struct {
	ResponseType ResponseType
}

func (result *IResult) SetResponseType(t ResponseType) {
	result.ResponseType = t
}

type Request_TYPE int32

const (
	REQ_JSON          Request_TYPE = iota
	REQ_FORMDATA      Request_TYPE = REQ_JSON + 1
	REQ_FORMURLENCODE Request_TYPE = REQ_JSON + 2
)

type Request struct {
	body     string
	type_    Request_TYPE
	response interface{}
}

//对客户端来源数据进行解析，这样子只支持了Content-Type为application/json的数据
func (req *Request) DealReq(r *http.Request) error {

	var err error
	var s []byte
	if len(r.Header["Content-Type"]) >= 1 && strings.Contains(r.Header["Content-Type"][0], "multipart/form-data;") {
		req.type_ = REQ_FORMDATA
	} else if len(r.Header["Content-Type"]) >= 1 && strings.HasPrefix(r.Header["Content-Type"][0], "application/x-www-form-urlencoded") {
		req.type_ = REQ_FORMURLENCODE
	} else {
		s, err = ioutil.ReadAll(r.Body) //把  body 内容读入字符串 s
		req.body = string(s)
		req.type_ = REQ_JSON
	}
	return err
}

//根据设定的传参结构体及返回值的结构体及返回客户端的数据格式，进行自动解析
func (req *Request) ParseParam(param IHandler, handlerName string, c *gin.Context) error {

	t := reflect.ValueOf(param)
	v := reflect.New(t.Elem().Type()).Interface()

	pt := GetReflectType(param)
	if req.type_ == REQ_JSON {
		if len(req.body) > 0 {
			isxml := false
			{
				err := json.Unmarshal([]byte(req.body), v.(IHandler))
				if err != nil {
					err2 := xml.Unmarshal([]byte(req.body), v.(IHandler))
					if err2 != nil {
						log.Warnln("[Warning Request]","xml解析error：", err)
						return err
					}
					isxml = true
				}

				t.Elem().Set(reflect.ValueOf(v).Elem())
			}

			var iter interface{}
			if !isxml {

				err := json.Unmarshal([]byte(req.body), &iter)
				if err != nil {
					log.Warnln("[Warning Request]",err)
					return err
				}
				str := verificationStruct(pt, iter.(map[string]interface{}))
				if str != nil {
					log.Warnln("[Warning]:", "缺少了必要的字段："+*str)
					return errors.New("必要字段丢失:" + *str)
				}
			} else {
				//xml暂不支持元素审查
			}
		}
	} else {
		for i := 0; i < pt.NumField(); i++ {
			jsonName := pt.Field(i).Tag.Get("json")
			required := pt.Field(i).Tag.Get("required") == "yes"
			if len(jsonName) > 0 {
				ok := false
				if pt.Field(i).Type.Name() == "string" {
					tempValue, had := c.GetPostForm(jsonName)
					if had {
						t.Elem().Field(i).SetString(tempValue)
						ok = true
					}
				} else if pt.Field(i).Type.Name() == "bool" {
					tempValue, had := c.GetPostForm(jsonName)
					if had {
						x, err := strconv.ParseBool(tempValue)
						if err == nil {
							t.Elem().Field(i).SetBool(x)
							ok = true
						} else {
							return err
						}
					}
				} else if strings.HasPrefix(pt.Field(i).Type.Name(), "int") {
					tempValue, had := c.GetPostForm(jsonName)
					if had {
						x, err := strconv.ParseInt(tempValue, 10, 64)
						if err == nil {
							t.Elem().Field(i).SetInt(x)
							ok = true
						} else {
							return err
						}
					}

				} else if pt.Field(i).Type.Name() == "float32" {
					tempValue, had := c.GetPostForm(jsonName)
					if had {
						x, err := strconv.ParseFloat(tempValue, 32)
						if err == nil {
							t.Elem().Field(i).SetFloat(x)
							ok = true
						} else {
							return err
						}
					}
				} else if pt.Field(i).Type.Name() == "float64" {
					tempValue, had := c.GetPostForm(jsonName)
					if had {
						x, err := strconv.ParseFloat(tempValue, 64)
						if err == nil {
							t.Elem().Field(i).SetFloat(x)
							ok = true
						} else {
							return err
						}
					}
				} else if pt.Field(i).Type.String() == "*os.File" {

					if req.type_ == REQ_FORMDATA {
						file, _, err := c.Request.FormFile(jsonName)
						if err == nil {
							//var tempFile os.File = os.File{}
							tempFile, err := ioutil.TempFile("", jsonName)
							if err == nil {
								defer tempFile.Close()
								_, err = io.Copy(tempFile, file)
								if err == nil {
									t.Elem().Field(i).Set(reflect.ValueOf(tempFile))
									ok = true
								}
							}

						}

						if err != nil {
							log.Warnln("[Warning Request]", err)
						}
					}

				}
				if required && !ok {
					log.Warnln("[Warning Request]", "缺少了必要的字段："+jsonName)
					return errors.New("必要字段丢失:" + jsonName)
				}
			}
		}
	}
	results := t.MethodByName(handlerName).Call([]reflect.Value{})
	//函数返回值是否为空，空的话就代表处理客户端请求成功
	if results[0].IsNil() || !results[0].IsValid() {
		req.response = nil
	} else {
		error := reflect.New(results[0].Type()).Interface()
		error_t := reflect.ValueOf(error)
		error_t.Elem().Set(results[0])
		req.response = error
	}
	return nil
}

type reqAndResponse struct {
	funcName string
	request  *reflect.Type
}

//var routers = make(map[string]*reqAndResponse)
func registerOver(mainRouter string) {
	apis.Write(mainRouter)
}

/*注册路由，根据路由来设置对应的接收参数和返回值
@param op 路由名称
@param request 接口调用对象
@param method 接口处理函数
@param result 接口正确返回值
@param err 接口错误返回值
@param note 备注
*/
func RegisterFunc(router string, request IHandler, method func() interface{}, result interface{}, err interface{}, note string) {

	//fmt.Println("reflect.TypeOf(request).Name():",reflect.TypeOf(request).Name())

	apis.Add(router, request, method, result, err, note)

	fn := runtime.FuncForPC(reflect.ValueOf(method).Pointer()).Name()

	//fmt.Println("fn:",fn)

	name := fn[strings.LastIndex(fn, ".")+1 : strings.LastIndex(fn, "-")]

	RegisterFuncByName(router, request, name)
}

//注册路由，根据路由来设置对应的接收参数和返回值
func RegisterFuncByName(router string, request IHandler, methodName string) {
	req := GetReflectType(request)

	log.Infoln("注册路由成功：",SUBJECT+"="+router + "  action="+methodName)
	if DefaultServer().routers[router] == nil {
		DefaultServer().routers[router] = make(map[string]*reqAndResponse)
	}
	DefaultServer().routers[router][methodName] = &reqAndResponse{request: &req, funcName: methodName}
}

//根据url中的op字段进行路由，客户端传入参数转化为用户定义的结构体进行回调，返回值将返回至客户端，根据用户的返回值类型不同进行不同的处理
//如需返回特殊的如XML、FILE两种形式，需继承IResult，并对ResponseType进行赋值，设置返回客户端的数据格式
func (ser *NServer) RequestRouter() gin.HandlerFunc {

	return func(c *gin.Context) {

		//c.SaveUploadedFile()
		//fmt.Println("c.Request.URL.RawPath:", c.Request.URL.Path)
		log.Println("[Request] RequestURI:", c.Request.RequestURI)

		if !strings.HasPrefix(c.Request.URL.Path, ser.mainRouter) {
			return
		}

		list := strings.Split(c.Request.URL.Path, "/")

		subject := ""
		action := ""
		if len(list) == 2 {
			subject = c.DefaultQuery(SUBJECT, "")
			action = c.DefaultQuery(ACTION, "")
		} else if len(list) == 4 {
			subject = list[2]
			action = list[3]
		} else {
			return
		}

		//fmt.Println("op method:", subject, action)
		var request IHandler
		var handlerName string
		if ser.routers[subject][action] != nil {
			request = reflect.New(*(ser.routers[subject][action]).request).Interface().(IHandler)
			handlerName = ser.routers[subject][action].funcName
			c.GetPostForm("test")
		} else {
			log.Println("[Waring Request] 没找到符合的处理器", ser.routers)
			c.String(http.StatusNotFound, ``)
			return
		}
		//fmt.Println("op method:", 1)
		req := Request{}
		err := req.DealReq(c.Request)
		if err != nil {
			panic(err)
		}

		//fmt.Println("request type:", GetReflectType(request).Name())

		{
			var t = reflect.ValueOf(request)
			if t.Elem().FieldByName(ContextName).IsValid() {

				//fmt.Println("ContextOver:")
				t.Elem().FieldByName(ContextName).Set(reflect.ValueOf(c))
			}
		}
		err = req.ParseParam(request, handlerName, c)

		if err != nil {
			c.String(http.StatusOK, `{"status":`+PARAMPORSEERROR+`,"error":"数据转换发生异常,`+err.Error()+`"}`)
			return
		}

		//fmt.Println("req.response:", req.response)
		respValue := GetReflectValue(req.response)

		respType := respValue.Type()

		//fmt.Println("req.response respType:", respType)
		for reflect.Ptr == respType.Kind() || reflect.Interface == respType.Kind() {
			respType = respType.Elem()
		}

		if respType.Kind() == reflect.Struct {
			_, have := respType.FieldByName(ResponseTypeName)
			if !have {
				body, err := json.MarshalIndent(req.response, "", "\t")
				if err != nil {
					c.String(http.StatusOK, `{"status":`+PARAMUNMARSHALERROR+`,"error":"服务器返回值转换异常,`+err.Error()+`"}`)
					return
				}
				c.String(http.StatusOK, string(body))
				return
			} else {

				//responseType := respValue.FieldByName(IResultName)
				responseType := respValue.FieldByName(ResponseTypeName)
				switch responseType.Int() {

				case STRING:
					{
						c.String(http.StatusOK, respValue.FieldByName(BODY).String())
					}
				case XML:
					{
						body, err := xml.MarshalIndent(req.response, "", "\t")
						if err != nil {
							c.String(http.StatusOK, `{"status":`+PARAMUNMARSHALERROR+`,"error":"服务器返回值转换异常`+err.Error()+`"}`)
							return
						}

						c.String(http.StatusOK, string(body))
						return
					}
				case FILE:
					{
						c.File(respValue.FieldByName("FilePath").String())
						return
					}
				default:
					{
						body, err := json.MarshalIndent(req.response, "", "\t")
						if err != nil {
							c.String(http.StatusOK, `{"status":`+PARAMUNMARSHALERROR+`,"error":"服务器返回值转换异常`+err.Error()+`"}`)
							return
						}

						c.String(http.StatusOK, string(body))
						return
					}
				}
			}
			return
		} else if respType.Kind() == reflect.String {

			if (*(req.response.(*interface{}))).(string) == "Redirect" {
				return
			}
			//fmt.Println("(*(req.response.(*interface{}))).(string):", (*(req.response.(*interface{}))).(string))
			c.String(http.StatusOK, (*(req.response.(*interface{}))).(string))
			return
		} else if respType.Kind() == reflect.Int {

			c.String(http.StatusOK, strconv.FormatInt(int64((*(req.response.(*interface{}))).(int)), 10))
			return
		} else if reflect.Map == respType.Kind() {
			body, err := json.MarshalIndent(req.response, "", "\t")
			if err != nil {
				c.String(http.StatusOK, `{"status":`+PARAMUNMARSHALERROR+`,"error":"服务器返回值转换异常`+err.Error()+`"}`)
				return
			}

			c.String(http.StatusOK, string(body))
		} else {
			//fmt.Println("123456:", req.response)
			body, err := json.MarshalIndent(req.response, "", "\t")
			if err != nil {
				c.String(http.StatusOK, `{"status":`+PARAMUNMARSHALERROR+`,"error":"服务器返回值转换异常`+err.Error()+`"}`)
				return
			}

			c.String(http.StatusOK, string(body))
		}
		//

	}

}
