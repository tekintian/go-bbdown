package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tekintian/go-bbdown/core"
)

var (
	cfgFile          string
	useTVApi         bool
	useAppApi        bool
	useIntlApi       bool
	useMP4Box        bool
	encoding         string
	quality          string
	onlyInfo         bool
	showAll          bool
	useAria2c        bool
	interactive      bool
	hideStreams      bool
	multiThread      bool
	simplyMux        bool
	videoOnly        bool
	audioOnly        bool
	danmakuOnly      bool
	coverOnly        bool
	subOnly          bool
	debug            bool
	skipMux          bool
	skipSub          bool
	skipCover        bool
	forceHTTP        bool
	downloadDanmaku  bool
	skipAI           bool
	videoAsc         bool
	audioAsc         bool
	allowPCDN        bool
	forceReplaceHost bool
	filePattern      string
	multiFilePattern string
	selectPage       string
	language         string
	userAgent        string
	cookie           string
	accessToken      string
	aria2cArgs       string
	workDir          string
	ffmpegPath       string
	mp4boxPath       string
	aria2cPath       string
	uposHost         string
	delayPerPage     string
	host             string
	epHost           string
	tvHost           string
	area             string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-bbdown [url]",
	Short: "BBDown是一个免费且便捷高效的哔哩哔哩下载/解析软件",
	Long: `BBDown是一个免费且便捷高效的哔哩哔哩下载/解析软件。

支持以下链接格式：
- BV号: BV1xx411c7mD
- AV号: av12345678
- EP号: ep123456
- SS号: ss123456
- 完整链接: https://www.bilibili.com/video/BV1xx411c7mD
- 番剧链接: https://www.bilibili.com/bangumi/play/ss123456
- 合集链接: https://space.bilibili.com/89320896/lists/5348941?type=season
- 媒体列表: https://www.bilibili.com/medialist/detail/ml123456`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var url string
		if len(args) > 0 {
			url = args[0]
		}

		config := &core.Config{
			UseTVApi:         useTVApi,
			UseAppApi:        useAppApi,
			UseIntlApi:       useIntlApi,
			UseMP4Box:        useMP4Box,
			EncodingPriority: strings.Split(encoding, ","),
			QualityPriority:  strings.Split(quality, ","),
			OnlyShowInfo:     onlyInfo,
			ShowAll:          showAll,
			UseAria2c:        useAria2c,
			Interactive:      interactive,
			HideStreams:      hideStreams,
			MultiThread:      multiThread,
			SimplyMux:        simplyMux,
			VideoOnly:        videoOnly,
			AudioOnly:        audioOnly,
			DanmakuOnly:      danmakuOnly,
			CoverOnly:        coverOnly,
			SubOnly:          subOnly,
			Debug:            debug,
			SkipMux:          skipMux,
			SkipSubtitle:     skipSub,
			SkipCover:        skipCover,
			ForceHTTP:        forceHTTP,
			DownloadDanmaku:  downloadDanmaku,
			SkipAI:           skipAI,
			VideoAscending:   videoAsc,
			AudioAscending:   audioAsc,
			AllowPCDN:        allowPCDN,
			ForceReplaceHost: forceReplaceHost,
			FilePattern:      filePattern,
			MultiFilePattern: multiFilePattern,
			SelectPage:       selectPage,
			Language:         language,
			UserAgent:        userAgent,
			Cookie:           cookie,
			AccessToken:      accessToken,
			Aria2cArgs:       aria2cArgs,
			WorkDir:          workDir,
			FFmpegPath:       ffmpegPath,
			Mp4boxPath:       mp4boxPath,
			Aria2cPath:       aria2cPath,
			UposHost:         uposHost,
			DelayPerPage:     delayPerPage,
			Host:             host,
			EpHost:           epHost,
			TvHost:           tvHost,
			Area:             area,
		}

		if err := core.Download(url, config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-bbdown.yaml)")

	// 下载选项flags
	rootCmd.Flags().BoolVarP(&useTVApi, "use-tv-api", "", false, "使用TV端解析模式")
	rootCmd.Flags().BoolVarP(&useAppApi, "use-app-api", "", false, "使用APP端解析模式")
	rootCmd.Flags().BoolVarP(&useIntlApi, "use-intl-api", "", false, "使用国际版(东南亚视频)解析模式")
	rootCmd.Flags().BoolVar(&useMP4Box, "use-mp4box", false, "使用MP4Box来混流")
	rootCmd.Flags().StringVarP(&encoding, "encoding-priority", "e", "", "视频编码的选择优先级,用逗号分割 例: \"hevc,av1,avc\"")
	rootCmd.Flags().StringVarP(&quality, "dfn-priority", "q", "", "画质优先级,用逗号分隔 例: \"8K 超高清,1080P 高码率,HDR 真彩,杜比视界\"")
	rootCmd.Flags().BoolVarP(&onlyInfo, "info", "i", false, "只展示信息, 不下载")
	rootCmd.Flags().BoolVar(&showAll, "show-all", false, "展示所有信息")
	rootCmd.Flags().BoolVar(&useAria2c, "use-aria2c", false, "使用aria2c进行下载")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "", false, "交互式选择下载清晰度和编码")
	rootCmd.Flags().BoolVar(&hideStreams, "hide-streams", false, "隐藏所有可用流信息")
	rootCmd.Flags().BoolVar(&multiThread, "multi-thread", true, "是否使用多线程下载")
	rootCmd.Flags().BoolVar(&simplyMux, "simply-mux", false, "简单混流(不合并音视频, 仅混合格式)")
	rootCmd.Flags().BoolVar(&videoOnly, "video-only", false, "只下载视频")
	rootCmd.Flags().BoolVar(&audioOnly, "audio-only", false, "只下载音频")
	rootCmd.Flags().BoolVar(&danmakuOnly, "danmaku-only", false, "只下载弹幕")
	rootCmd.Flags().BoolVar(&coverOnly, "cover-only", false, "只下载封面")
	rootCmd.Flags().BoolVar(&subOnly, "sub-only", false, "只下载字幕")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "启用调试模式")
	rootCmd.Flags().BoolVar(&skipMux, "skip-mux", false, "跳过混流")
	rootCmd.Flags().BoolVar(&skipSub, "skip-subtitle", false, "跳过字幕下载")
	rootCmd.Flags().BoolVar(&skipCover, "skip-cover", false, "跳过封面下载")
	rootCmd.Flags().BoolVar(&forceHTTP, "force-http", true, "强制使用HTTP协议")
	rootCmd.Flags().BoolVar(&downloadDanmaku, "download-danmaku", false, "下载弹幕")
	rootCmd.Flags().BoolVar(&skipAI, "skip-ai", true, "跳过AI生成内容")
	rootCmd.Flags().BoolVar(&videoAsc, "video-ascending", false, "视频编码按序升序")
	rootCmd.Flags().BoolVar(&audioAsc, "audio-ascending", false, "音频编码按序升序")
	rootCmd.Flags().BoolVar(&allowPCDN, "allow-pcdn", false, "允许PCDN")
	rootCmd.Flags().BoolVar(&forceReplaceHost, "force-replace-host", true, "强制替换主机")

	// 文件和路径相关
	rootCmd.Flags().StringVar(&filePattern, "file-pattern", "", "单文件保存路径模板")
	rootCmd.Flags().StringVar(&multiFilePattern, "multi-file-pattern", "", "多文件保存路径模板")
	rootCmd.Flags().StringVar(&selectPage, "select-page", "", "选择指定分P")
	rootCmd.Flags().StringVar(&language, "language", "", "选择音轨语言")
	rootCmd.Flags().StringVar(&workDir, "work-dir", "", "工作目录")

	// 网络和认证相关
	rootCmd.Flags().StringVar(&userAgent, "user-agent", "", "自定义User-Agent")
	rootCmd.Flags().StringVar(&cookie, "cookie", "", "自定义Cookie")
	rootCmd.Flags().StringVar(&accessToken, "access-token", "", "访问令牌")
	rootCmd.Flags().StringVar(&uposHost, "upos-host", "", "UPOS主机")
	rootCmd.Flags().StringVar(&delayPerPage, "delay-per-page", "0", "每页延迟时间")
	rootCmd.Flags().StringVar(&host, "host", "api.bilibili.com", "API主机")
	rootCmd.Flags().StringVar(&epHost, "ep-host", "api.bilibili.com", "EP API主机")
	rootCmd.Flags().StringVar(&tvHost, "tv-host", "api.snm0516.aisee.tv", "TV API主机")
	rootCmd.Flags().StringVar(&area, "area", "", "地区")

	// 外部工具相关
	rootCmd.Flags().StringVar(&aria2cArgs, "aria2c-args", "", "aria2c参数")
	rootCmd.Flags().StringVar(&ffmpegPath, "ffmpeg-path", "", "FFmpeg路径")
	rootCmd.Flags().StringVar(&mp4boxPath, "mp4box-path", "", "MP4Box路径")
	rootCmd.Flags().StringVar(&aria2cPath, "aria2c-path", "", "aria2c路径")

	viper.BindPFlag("use-tv-api", rootCmd.Flags().Lookup("use-tv-api"))
	viper.BindPFlag("use-app-api", rootCmd.Flags().Lookup("use-app-api"))
	viper.BindPFlag("use-intl-api", rootCmd.Flags().Lookup("use-intl-api"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".go-bbdown")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
