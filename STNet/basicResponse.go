package STNet

//如果返回客户端为FILE，请使用该结构体
type FILE_Message struct {
	IResult
	FilePath string `json:"filepath"`
}

type State_Message struct {
	IResult
	State   bool   `json:"state"`
	Message string `json:"message"`
}
