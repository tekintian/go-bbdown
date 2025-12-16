package core

import (
	"time"
)

// Page 视频页面信息
type Page struct {
	Index     int         `json:"page"`
	Aid       int64       `json:"aid"`
	Cid       int64       `json:"cid"`
	Epid      string      `json:"epid"`
	Title     string      `json:"title"`
	Dur       int         `json:"duration"`
	Res       string      `json:"resolution"`
	PubTime   int64       `json:"pubTime"`
	Cover     string      `json:"cover,omitempty"`
	Desc      string      `json:"desc,omitempty"`
	OwnerName string      `json:"ownerName,omitempty"`
	OwnerMid  string      `json:"ownerMid,omitempty"`
	Bvid      string      `json:"bvid,omitempty"`
	Points    []ViewPoint `json:"points,omitempty"`
	Part      string      `json:"part"` // 分P标题
}

// ViewPoint 视频节点（用于互动视频）
type ViewPoint struct {
	Type  int    `json:"type"`
	Title string `json:"title"`
	Dur   int    `json:"duration"`
}

// VInfo 视频信息
type VInfo struct {
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	Pic         string `json:"pic"`
	Owner       Owner  `json:"owner"`
	PubTime     int64  `json:"pubtime"`
	Pages       []Page `json:"pages"`
	IsBangumi   bool   `json:"isBangumi"`
	IsCheese    bool   `json:"isCheese"`
	IsSteinGate bool   `json:"isSteinGate"`
	SeasonID    string `json:"seasonId,omitempty"`
	SeasonName  string `json:"seasonName,omitempty"`
	Stat        struct {
		View     int64 `json:"view"`
		Danmaku  int64 `json:"danmaku"`
		Reply    int64 `json:"reply"`
		Like     int64 `json:"like"`
		Coin     int64 `json:"coin"`
		Favorite int64 `json:"favorite"`
	} `json:"stat"`
	Aid int64 `json:"aid"`
}

// Owner UP主信息
type Owner struct {
	Mid  int64  `json:"mid"`
	Name string `json:"name"`
}

// Track 音视频轨道
type Track struct {
	ID          int      `json:"id"`
	Codecid     int      `json:"codecid"`
	Quality     int      `json:"quality"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	MD5         string   `json:"md5"`
	Size        int64    `json:"size"`
	Bandwidth   int      `json:"bandwidth"`
	FPS         int      `json:"fps"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Format      string   `json:"format"`
	Codec       string   `json:"codec"`
	FrameType   string   `json:"frameType"`            // "video" or "audio"
	BackupURLs  []string `json:"backupUrls,omitempty"` // 备份URL
}

// ParsedResult 解析结果
type ParsedResult struct {
	Tracks  []Track `json:"tracks"`
	Code    int     `json:"code"`
	Message string  `json:"message"`
}

// DownloadTask 下载任务
type DownloadTask struct {
	URL       string
	FilePath  string
	TempPath  string
	Size      int64
	Progress  float64
	Speed     int64
	StartTime time.Time
	Track     Track
}

// ApiResponse API响应基础结构
type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PlayUrlData 播放链接数据
type PlayUrlData struct {
	Duration int     `json:"duration"`
	Video    []Track `json:"video"`
	Audio    []Track `json:"audio"`
	Flash    []Track `json:"flash"`
	DASH     struct {
		Video []Track `json:"video"`
		Audio []Track `json:"audio"`
	} `json:"dash"`
}

// VideoInfoData 视频信息数据
type VideoInfoData struct {
	Bvid    string `json:"bvid"`
	Aid     string `json:"aid"`
	Title   string `json:"title"`
	Desc    string `json:"desc"`
	Pic     string `json:"pic"`
	Owner   Owner  `json:"owner"`
	Pubdate int64  `json:"pubdate"`
	Pages   []Page `json:"pages"`
	Rights  Rights `json:"rights"`
}

// Rights 视频权限信息
type Rights struct {
	IsSteinGate int `json:"is_stein_gate"`
}

// BangumiInfo 番剧信息
type BangumiInfo struct {
	SeasonID   string `json:"season_id"`
	SeasonName string `json:"season_name"`
	EpID       string `json:"ep_id"`
	EpTitle    string `json:"ep_title"`
	IsFinish   bool   `json:"is_finish"`
	TotalCount int    `json:"total_count"`
}

// CheeseInfo 课程信息
type CheeseInfo struct {
	SeasonID   string `json:"season_id"`
	SeasonName string `json:"season_name"`
	EpID       string `json:"ep_id"`
	EpTitle    string `json:"ep_title"`
}

// SeasonInfo 合集信息
type SeasonInfo struct {
	SeasonID    string        `json:"season_id"`
	SeasonName  string        `json:"season_name"`
	Description string        `json:"description"`
	TotalCount  int           `json:"total_count"`
	Videos      []SeasonVideo `json:"videos"`
}

// SeasonVideo 合集中的视频
type SeasonVideo struct {
	Aid      int64  `json:"aid"`
	Bvid     string `json:"bvid"`
	Cid      int64  `json:"cid"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Cover    string `json:"cover"`
	Index    int    `json:"index"`
	Part     string `json:"part"`
}

// MediaListInfo 媒体列表信息
type MediaListInfo struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	TotalCount  int          `json:"total_count"`
	Videos      []MediaVideo `json:"videos"`
}

// MediaVideo 媒体列表中的视频
type MediaVideo struct {
	Aid      int64  `json:"aid"`
	Bvid     string `json:"bvid"`
	Cid      int64  `json:"cid"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Cover    string `json:"cover"`
	Index    int    `json:"index"`
	Part     string `json:"part"`
}

// FavoriteInfo 收藏夹信息
type FavoriteInfo struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	TotalCount  int            `json:"total_count"`
	Videos      []FavoriteVideo `json:"videos"`
}

// FavoriteVideo 收藏夹中的视频
type FavoriteVideo struct {
	Aid      int64  `json:"aid"`
	Bvid     string `json:"bvid"`
	Cid      int64  `json:"cid"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Cover    string `json:"cover"`
	Part     string `json:"part"`
}
