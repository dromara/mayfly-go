package form

type ImageOp struct {
	Host    string `json:"host" binding:"required"`
	ImageId string `json:"imageId" binding:"required"`
}
