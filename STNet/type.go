package STNet

type ResponseType int

const  (
	JSON = iota
	STRING
	FILE
	XML
)

type ResponseCode string

const  (
	PARAMPORSEERROR = "0"				//客户端传参进入服务器------->传参错误
	PARAMUNMARSHALERROR = "1"			//服务器回传时-------------->序列化失败
)
