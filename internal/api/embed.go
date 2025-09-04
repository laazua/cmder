package api

// 存储嵌入的内容
var (
	embeddedIndexHTML string // 设置要嵌入的单个html文件
	// embeddedWebFS     embed.FS        // 设置要嵌入的静态资源目录
	// webFileSystem     http.FileSystem // 设置文件系统
)

// InitWebContent 初始化嵌入的 web 内容
// 如果需要嵌入静态资源问则,给方法InitWebContent加入对应的参数webFS embed.FS, 即:
//
//	func InitWebContent(indexHTML string, webFS embed.FS)
func InitWebContent(indexHTML string) {
	embeddedIndexHTML = indexHTML
	// embeddedWebFS = webFS

	// 创建静态文件服务所需的文件系统
	// subFS, err := fs.Sub(webFS, "web")
	// if err != nil {
	// 	panic("创建 web 子文件系统失败: " + err.Error())
	// }
	// webFileSystem = http.FS(subFS)
}
