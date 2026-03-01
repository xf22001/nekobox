package constant

const (
	RunMode_Other = iota
	RunMode_NekoRay_Core
	RunMode_NekoBox_Core
	RunMode_NekoBoxForAndroid
)

var RunMode int = RunMode_NekoBox_Core
