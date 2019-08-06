package myutil

// iota 初始化后会自动递增
const (
	// C -> S signal
	Request4File = iota // 客户端提出文件请求
	Request4SplitFile	// 客户端请求子文件数据
)
const (
	// S -> C signal
	FileReady = iota		// 文件已就绪
	FileNotFound	// 文件未找到
	NoPermission	// 没有权限
	ServerError		// 服务器内部错误
	TokenError		// 令牌错误
	RequestError	// 请求错误
	SplitFileNotFound	// 子文件不存在
	SplitFileData	// 子文件数据传送

	SessionIdLength = 32
)