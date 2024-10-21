package vo

type SimpleFile struct {
	Filename string `json:"filename"`
	FileKey  string `json:"fileKey"`
	Size     int64  `json:"size"`
}
