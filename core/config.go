package core

// Config 应用配置
type Config struct {
	// API选项
	UseTVApi   bool `json:"useTvApi"`
	UseAppApi  bool `json:"useAppApi"`
	UseIntlApi bool `json:"useIntlApi"`
	UseMP4Box  bool `json:"useMp4Box"`

	// 质量和编码选择
	EncodingPriority []string `json:"encodingPriority"`
	QualityPriority  []string `json:"qualityPriority"`

	// 显示选项
	OnlyShowInfo bool `json:"onlyShowInfo"`
	ShowAll      bool `json:"showAll"`
	Interactive  bool `json:"interactive"`
	HideStreams  bool `json:"hideStreams"`
	Debug        bool `json:"debug"`

	// 下载选项
	UseAria2c       bool `json:"useAria2c"`
	MultiThread     bool `json:"multiThread"`
	SimplyMux       bool `json:"simplyMux"`
	VideoOnly       bool `json:"videoOnly"`
	AudioOnly       bool `json:"audioOnly"`
	DanmakuOnly     bool `json:"danmakuOnly"`
	CoverOnly       bool `json:"coverOnly"`
	SubOnly         bool `json:"subOnly"`
	SkipMux         bool `json:"skipMux"`
	SkipSubtitle    bool `json:"skipSubtitle"`
	SkipCover       bool `json:"skipCover"`
	SkipAI          bool `json:"skipAi"`
	DownloadDanmaku bool `json:"downloadDanmaku"`

	// 排序选项
	VideoAscending bool `json:"videoAscending"`
	AudioAscending bool `json:"audioAscending"`

	// 网络选项
	AllowPCDN        bool `json:"allowPcdn"`
	ForceHTTP        bool `json:"forceHttp"`
	ForceReplaceHost bool `json:"forceReplaceHost"`

	// 文件和路径
	FilePattern      string `json:"filePattern"`
	MultiFilePattern string `json:"multiFilePattern"`
	SelectPage       string `json:"selectPage"`
	Language         string `json:"language"`
	WorkDir          string `json:"workDir"`

	// 认证
	UserAgent   string `json:"userAgent"`
	Cookie      string `json:"cookie"`
	AccessToken string `json:"accessToken"`

	// 外部工具
	Aria2cArgs string `json:"aria2cArgs"`
	FFmpegPath string `json:"ffmpegPath"`
	Mp4boxPath string `json:"mp4boxPath"`
	Aria2cPath string `json:"aria2cPath"`

	// API主机
	UposHost     string `json:"uposHost"`
	DelayPerPage string `json:"delayPerPage"`
	Host         string `json:"host"`
	EpHost       string `json:"epHost"`
	TvHost       string `json:"tvHost"`
	Area         string `json:"area"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		UseTVApi:         false,
		UseAppApi:        false,
		UseIntlApi:       false,
		UseMP4Box:        false,
		EncodingPriority: []string{"hevc", "av1", "avc"},
		QualityPriority:  []string{"8K 超高清", "4K 超清", "1080P 高码率", "1080P 高清"},
		OnlyShowInfo:     false,
		ShowAll:          false,
		Interactive:      false,
		HideStreams:      false,
		Debug:            false,
		UseAria2c:        false,
		MultiThread:      true,
		SimplyMux:        false,
		VideoOnly:        false,
		AudioOnly:        false,
		DanmakuOnly:      false,
		CoverOnly:        false,
		SubOnly:          false,
		SkipMux:          false,
		SkipSubtitle:     false,
		SkipCover:        false,
		SkipAI:           true,
		DownloadDanmaku:  false,
		VideoAscending:   false,
		AudioAscending:   false,
		AllowPCDN:        false,
		ForceHTTP:        true,
		ForceReplaceHost: true,
		FilePattern:      "<videoTitle>",
		MultiFilePattern: "<videoTitle>/[P<pageNumberWithZero>]<pageTitle>",
		SelectPage:       "",
		Language:         "",
		WorkDir:          "",
		UserAgent:        "",
		Cookie:           "",
		AccessToken:      "",
		Aria2cArgs:       "",
		FFmpegPath:       "",
		Mp4boxPath:       "",
		Aria2cPath:       "",
		UposHost:         "",
		DelayPerPage:     "0",
		Host:             "api.bilibili.com",
		EpHost:           "api.bilibili.com",
		TvHost:           "api.snm0516.aisee.tv",
		Area:             "",
	}
}

// QualityMap 质量映射表
var QualityMap = map[string]string{
	"127": "8K 超高清",
	"126": "杜比视界",
	"125": "HDR 真彩",
	"120": "4K 超清",
	"116": "1080P 高帧率",
	"112": "1080P 高码率",
	"100": "智能修复",
	"80":  "1080P 高清",
	"74":  "720P 高帧率",
	"64":  "720P 高清",
	"48":  "720P 高清",
	"32":  "480P 清晰",
	"16":  "360P 流畅",
	"5":   "144P 流畅",
	"6":   "240P 流畅",
}

// EncodingMap 编码映射表
var EncodingMap = map[string]string{
	"hevc": "HEVC",
	"av1":  "AV1",
	"avc":  "AVC",
}
