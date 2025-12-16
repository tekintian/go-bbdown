package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tekintian/go-bbdown/util"
)

// Parser 解析器
type Parser struct {
	HttpClient *HTTPClient // 修复类型名称为HTTPClient
	Config     *Config
}

// NewParser 创建解析器
func NewParser(config *Config) *Parser {
	return &Parser{
		HttpClient: NewHTTPClient(), // 直接创建HTTPClient实例
		Config:     config,
	}
}

// ExtractTracks 提取音视频轨道
func (p *Parser) ExtractTracks(encoding, aidOri, aid, cid, epId string, tvApi, intlApi, appApi bool, qn string) ([]*Track, error) {
	if qn == "" {
		qn = "0"
	}

	var playJson string
	var err error

	if intlApi {
		playJson, err = p.getIntlPlayJson(aid, cid, epId, qn, "0")
	} else if appApi {
		playJson, err = p.getAppPlayJson(aid, cid, epId, qn, false, encoding, "")
	} else {
		playJson, err = p.getPlayJson(encoding, aidOri, aid, cid, epId, tvApi, false, appApi, qn)
	}

	if err != nil {
		return nil, err
	}

	// 解析播放数据
	tracks, err := p.parsePlayData(playJson, encoding)
	if err != nil {
		return nil, err
	}

	return tracks, nil
}

// getPlayJson 获取播放数据JSON
func (p *Parser) getPlayJson(encoding, aidOri, aid, cid, epId string, tvApi, intlApi, appApi bool, qn string) (string, error) {
	isCheese := strings.HasPrefix(aidOri, "cheese:")
	isBangumi := isCheese || strings.HasPrefix(aidOri, "ep:")

	var prefix string
	if tvApi {
		if isBangumi {
			prefix = fmt.Sprintf("https://%s/pgc/player/api/playurltv?", p.Config.TvHost)
		} else {
			prefix = fmt.Sprintf("https://%s/x/tv/playurl?", p.Config.TvHost)
		}
	} else {
		if isBangumi {
			prefix = fmt.Sprintf("https://%s/pgc/player/web/v2/playurl?", p.Config.Host)
		} else {
			prefix = "https://api.bilibili.com/x/player/wbi/playurl?"
		}
	}

	var api string
	if tvApi {
		params := make(map[string]string)
		if p.Config.AccessToken != "" {
			params["access_key"] = p.Config.AccessToken
		}
		params["appkey"] = "4409e2ce8ffd12b8"
		params["build"] = "106500"
		params["cid"] = cid
		params["device"] = "android"
		if isBangumi {
			params["ep_id"] = epId
			params["expire"] = "0"
		}
		params["fnval"] = "4048"
		params["fnver"] = "0"
		params["fourk"] = "1"
		params["mid"] = "0"
		params["mobi_app"] = "android_tv_yst"
		params["object_id"] = aid
		params["platform"] = "android"
		params["playurl_type"] = "1"
		params["qn"] = qn
		params["ts"] = strconv.FormatInt(time.Now().Unix(), 10)

		queryString := buildQueryStringHTTP(params)
		sign := util.Sign(queryString, "4409e2ce8ffd12b8", "59b43e04ad6965f34319062b478f83dd")
		api = fmt.Sprintf("%s%s&sign=%s", prefix, queryString, sign)
	} else {
		params := make(map[string]string)
		params["support_multi_audio"] = "true"
		params["from_client"] = "BROWSER"
		params["avid"] = aid
		params["cid"] = cid
		params["fnval"] = "4048"
		params["fnver"] = "0"
		params["fourk"] = "1"
		if p.Config.Area != "" {
			params["access_key"] = p.Config.AccessToken
			params["area"] = p.Config.Area
		}
		params["otype"] = "json"
		params["qn"] = qn
		if isBangumi {
			params["module"] = "bangumi"
			params["ep_id"] = epId
			params["session"] = ""
		}
		if p.Config.Cookie == "" {
			params["try_look"] = "1"
		}
		params["wts"] = strconv.FormatInt(time.Now().Unix(), 10)

		if isBangumi {
			api = fmt.Sprintf("%s%s", prefix, buildQueryStringHTTP(params))
		} else {
			// WBI签名
			wbiKey, err := GetWBIKey(p.HttpClient)
			if err != nil {
				return "", fmt.Errorf("获取WBI密钥失败: %w", err)
			}
			queryString := buildQueryStringHTTP(params)
			api = fmt.Sprintf("%s%s&w_rid=%s", prefix, queryString, util.WBISign(queryString, wbiKey))
		}
	}

	// 课程接口特殊处理
	if isCheese {
		api = strings.Replace(api, "/pgc/", "/pugv/", 1)
	}

	// 发送请求
	resp, err := p.HttpClient.GetWebSource(api, p.Config.UserAgent)
	if err != nil {
		return "", err
	}

	// 检查是否需要从网页源码解析
	if strings.Contains(resp, "大会员专享限制") {
		// 从网页源码尝试解析
		webUrl := fmt.Sprintf("https://www.bilibili.com/bangumi/play/ep%s", epId)
		webSource, err := p.HttpClient.GetWebSource(webUrl, p.Config.UserAgent)
		if err != nil {
			return "", err
		}

		// 正则匹配playerJson
		playerJsonRegex := regexp.MustCompile(`window.__playinfo__=({.*?});`)
		matches := playerJsonRegex.FindStringSubmatch(webSource)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return resp, nil
}

// getIntlPlayJson 获取国际版播放数据JSON
func (p *Parser) getIntlPlayJson(aid, cid, epId, qn, code string) (string, error) {
	isBiliPlus := p.Config.Host != "api.bilibili.com"
	var apiPrefix string
	if isBiliPlus {
		apiPrefix = fmt.Sprintf("https://%s/intl/gateway/v2/ogv/playurl?", p.Config.Host)
	} else {
		apiPrefix = "https://api.biliintl.com/intl/gateway/v2/ogv/playurl?"
	}

	params := make(map[string]string)
	if p.Config.AccessToken != "" {
		params["access_key"] = p.Config.AccessToken
	}
	params["aid"] = aid
	if isBiliPlus {
		params["appkey"] = "7d089525d3611b1c"
		area := p.Config.Area
		if area == "" {
			area = "th"
		}
		params["area"] = area
	}
	params["cid"] = cid
	params["ep_id"] = epId
	params["platform"] = "android"
	params["prefer_code_type"] = code
	params["qn"] = qn
	if isBiliPlus {
		params["ts"] = strconv.FormatInt(time.Now().Unix(), 10)
	}
	params["s_locale"] = "zh_SG"

	queryString := buildQueryStringHTTP(params)
	var api string
	if isBiliPlus {
		sign := util.Sign(queryString, "7d089525d3611b1c", "a2ffa5973e38609c2e0465a978e78c10")
		api = fmt.Sprintf("%s%s&sign=%s", apiPrefix, queryString, sign)
	} else {
		api = fmt.Sprintf("%s%s", apiPrefix, queryString)
	}

	// 发送请求
	resp, err := p.HttpClient.GetWebSource(api, p.Config.UserAgent)
	if err != nil {
		return "", err
	}

	return resp, nil
}

// getAppPlayJson 获取APP版播放数据JSON
func (p *Parser) getAppPlayJson(aid, cid, epId, qn string, bangumi bool, encoding string, token string) (string, error) {
	// APP API暂未实现
	return "", fmt.Errorf("APP API暂未实现")
}

// parsePlayData 解析播放数据
func (p *Parser) parsePlayData(playJson string, encoding string) ([]*Track, error) {
	// 解析JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(playJson), &data); err != nil {
		return nil, err
	}

	var tracks []*Track

	// 检查是否是国际版接口 (stream_list)
	if p.hasStreamList(data) {
		intlTracks, err := p.parseIntlPlayData(data)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, intlTracks...)
		return tracks, nil
	}

	// 获取根节点
	root := p.getRootNode(data)
	if root == nil {
		return nil, fmt.Errorf("无法解析播放数据根节点")
	}

	// 解析DASH格式
	if dash, ok := root["dash"].(map[string]interface{}); ok {
		dashTracks, err := p.parseDashData(dash)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, dashTracks...)
	}

	// 解析FLV格式 (durl)
	if durl, ok := root["durl"].([]interface{}); ok {
		flvTracks, err := p.parseFlvData(root, durl)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, flvTracks...)
	}

	return tracks, nil
}

// hasStreamList 检查是否包含stream_list (国际版接口)
func (p *Parser) hasStreamList(data map[string]interface{}) bool {
	if dataObj, ok := data["data"].(map[string]interface{}); ok {
		if _, ok := dataObj["video_info"]; ok {
			if videoInfo, ok := dataObj["video_info"].(map[string]interface{}); ok {
				if _, ok := videoInfo["stream_list"]; ok {
					return true
				}
			}
		}
	}
	return false
}

// parseIntlPlayData 解析国际版播放数据
func (p *Parser) parseIntlPlayData(data map[string]interface{}) ([]*Track, error) {
	var tracks []*Track

	dataObj := data["data"].(map[string]interface{})
	videoInfo := dataObj["video_info"].(map[string]interface{})
	_ = p.getIntValue(videoInfo, "timelength") / 1000 // duration用于国际版接口

	// 处理视频流
	if streamList, ok := videoInfo["stream_list"].([]interface{}); ok {
		for _, stream := range streamList {
			if streamMap, ok := stream.(map[string]interface{}); ok {
				if dashVideo, ok := streamMap["dash_video"].(map[string]interface{}); ok {
					if baseURL, ok := dashVideo["base_url"].(string); ok && baseURL != "" {
						track := &Track{
							ID:          p.getIntValue(streamMap["stream_info"], "quality"),
							Description: getQualityDesc(p.getIntValue(streamMap["stream_info"], "quality")),
							URL:         baseURL,
							Bandwidth:   p.getIntValue(dashVideo, "bandwidth") / 1000,
							FrameType:   "video",
							Codec:       p.getVideoCodec(p.getIntValue(dashVideo, "codecid")),
							Size:        int64(p.getIntValue(dashVideo, "size")),
						}
						tracks = append(tracks, track)
					}
				}
			}
		}
	}

	// 处理音频流
	if dashAudio, ok := videoInfo["dash_audio"].([]interface{}); ok {
		for _, audio := range dashAudio {
			if audioMap, ok := audio.(map[string]interface{}); ok {
				track := &Track{
					ID:          p.getIntValue(audioMap, "id"),
					Description: fmt.Sprintf("%d", p.getIntValue(audioMap, "id")),
					URL:         p.getStringValue(audioMap, "base_url"),
					Bandwidth:   p.getIntValue(audioMap, "bandwidth") / 1000,
					FrameType:   "audio",
					Codec:       "M4A",
				}
				tracks = append(tracks, track)
			}
		}
	}

	return tracks, nil
}

// getRootNode 获取根节点
func (p *Parser) getRootNode(data map[string]interface{}) map[string]interface{} {
	// 检查result节点
	if result, ok := data["result"].(map[string]interface{}); ok {
		// v2接口有video_info节点
		if videoInfo, ok := result["video_info"].(map[string]interface{}); ok {
			return videoInfo
		}
		return result
	}

	// 检查data节点
	if dataObj, ok := data["data"].(map[string]interface{}); ok {
		return dataObj
	}

	return data
}

// parseDashData 解析DASH数据
func (p *Parser) parseDashData(dash map[string]interface{}) ([]*Track, error) {
	var tracks []*Track

	// 处理视频流
	if videoData, ok := dash["video"].([]interface{}); ok {
		for _, v := range videoData {
			if video, ok := v.(map[string]interface{}); ok {
				// 尝试获取最佳URL，优先使用backup_url
				baseURL := p.getStringValue(video, "base_url")
				if backupUrls, ok := video["backup_url"].([]interface{}); ok && len(backupUrls) > 0 {
					// 使用第一个备用URL
					if backupURL, ok := backupUrls[0].(string); ok && backupURL != "" {
						baseURL = backupURL
					}
				}

				track := &Track{
					ID:          p.getIntValue(video, "id"),
					Description: getQualityDesc(p.getIntValue(video, "id")),
					URL:         baseURL,
					Bandwidth:   p.getIntValue(video, "bandwidth") / 1000,
					FrameType:   "video",
					Codec:       p.getVideoCodec(p.getIntValue(video, "codecid")),
					Size:        int64(p.getIntValue(video, "size")),
					Width:       p.getIntValue(video, "width"),
					Height:      p.getIntValue(video, "height"),
					FPS:         p.getIntValue(video, "frame_rate"),
				}
				tracks = append(tracks, track)
			}
		}
	}

	// 处理音频流
	if audioData, ok := dash["audio"].([]interface{}); ok {
		for _, a := range audioData {
			if audio, ok := a.(map[string]interface{}); ok {
				codecs := p.getStringValue(audio, "codecs")
				// 转换编码格式
				switch codecs {
				case "mp4a.40.2", "mp4a.40.5":
					codecs = "M4A"
				case "ec-3":
					codecs = "E-AC-3"
				case "fLaC":
					codecs = "FLAC"
				}

				// 尝试获取最佳URL，优先使用backup_url
				baseURL := p.getStringValue(audio, "base_url")
				if backupUrls, ok := audio["backup_url"].([]interface{}); ok && len(backupUrls) > 0 {
					// 使用第一个备用URL
					if backupURL, ok := backupUrls[0].(string); ok && backupURL != "" {
						baseURL = backupURL
					}
				}

				track := &Track{
					ID:          p.getIntValue(audio, "id"),
					Description: p.getStringValue(audio, "id"),
					URL:         baseURL,
					Bandwidth:   p.getIntValue(audio, "bandwidth") / 1000,
					FrameType:   "audio",
					Codec:       codecs,
				}
				tracks = append(tracks, track)
			}
		}
	}

	// 处理杜比音频
	if dolby, ok := dash["dolby"].(map[string]interface{}); ok {
		if dolbyAudio, ok := dolby["audio"].([]interface{}); ok {
			for _, a := range dolbyAudio {
				if audio, ok := a.(map[string]interface{}); ok {
					// 尝试获取最佳URL，优先使用backup_url
					baseURL := p.getStringValue(audio, "base_url")
					if backupUrls, ok := audio["backup_url"].([]interface{}); ok && len(backupUrls) > 0 {
						// 使用第一个备用URL
						if backupURL, ok := backupUrls[0].(string); ok && backupURL != "" {
							baseURL = backupURL
						}
					}

					track := &Track{
						ID:          p.getIntValue(audio, "id"),
						Description: p.getStringValue(audio, "id"),
						URL:         baseURL,
						Bandwidth:   p.getIntValue(audio, "bandwidth") / 1000,
						FrameType:   "audio",
						Codec:       "E-AC-3",
					}
					tracks = append(tracks, track)
				}
			}
		}
	}

	// 处理Hi-Res无损音频
	if flac, ok := dash["flac"].(map[string]interface{}); ok {
		if flacAudio, ok := flac["audio"].(map[string]interface{}); ok {
			if flacAudio != nil {
				track := &Track{
					ID:          p.getIntValue(flacAudio, "id"),
					Description: p.getStringValue(flacAudio, "id"),
					URL:         p.getStringValue(flacAudio, "base_url"),
					Bandwidth:   p.getIntValue(flacAudio, "bandwidth") / 1000,
					FrameType:   "audio",
					Codec:       "FLAC",
				}
				tracks = append(tracks, track)
			}
		}
	}

	return tracks, nil
}

// parseFlvData 解析FLV数据
func (p *Parser) parseFlvData(root map[string]interface{}, durl []interface{}) ([]*Track, error) {
	var tracks []*Track

	quality := p.getIntValue(root, "quality")
	videoCodecid := p.getStringValue(root, "video_codecid")

	var totalSize int64
	var totalLength int64

	// 计算总大小和时长
	for _, node := range durl {
		if clip, ok := node.(map[string]interface{}); ok {
			totalSize += p.getInt64Value(clip, "size")
			totalLength += p.getInt64Value(clip, "length")
		}
	}

	// 创建FLV轨道
	track := &Track{
		ID:          quality,
		Description: getQualityDesc(quality),
		FrameType:   "video",
		Codec:       p.getVideoCodecString(videoCodecid),
		Size:        totalSize,
	}

	tracks = append(tracks, track)
	return tracks, nil
}

// getIntValue 安全获取int值
func (p *Parser) getIntValue(obj interface{}, key string) int {
	if m, ok := obj.(map[string]interface{}); ok {
		if val, ok := m[key].(float64); ok {
			return int(val)
		}
	}
	return 0
}

// getInt64Value 安全获取int64值
func (p *Parser) getInt64Value(obj interface{}, key string) int64 {
	if m, ok := obj.(map[string]interface{}); ok {
		if val, ok := m[key].(float64); ok {
			return int64(val)
		}
	}
	return 0
}

// getStringValue 安全获取string值
func (p *Parser) getStringValue(obj interface{}, key string) string {
	if m, ok := obj.(map[string]interface{}); ok {
		if val, ok := m[key].(string); ok {
			return val
		}
	}
	return ""
}

// getVideoCodec 获取视频编码
func (p *Parser) getVideoCodec(code int) string {
	switch code {
	case 13:
		return "AV1"
	case 12:
		return "HEVC"
	case 7:
		return "AVC"
	default:
		return "UNKNOWN"
	}
}

// getVideoCodecString 获取视频编码字符串
func (p *Parser) getVideoCodecString(code string) string {
	switch code {
	case "13":
		return "AV1"
	case "12":
		return "HEVC"
	case "7":
		return "AVC"
	default:
		return "UNKNOWN"
	}
}

// convertToTrack 转换为轨道
func (p *Parser) convertToTrack(data map[string]interface{}, trackType, encoding string) *Track {
	// 提取基础信息
	id, _ := data["id"].(float64)
	baseUrl, _ := data["baseUrl"].(string)
	// backupUrl, _ := data["backupUrl"].([]interface{}) // 删除未使用的变量
	mimeType, _ := data["mimeType"].(string)
	codecs, _ := data["codecs"].(string)
	width, _ := data["width"].(float64)
	height, _ := data["height"].(float64)
	frameRate, _ := data["frameRate"].(string)
	// sar, _ := data["sar"].(string)  // 删除未使用的变量
	codecid, _ := data["codecid"].(float64)
	// size, _ := data["size"].(float64)  // 删除未使用的变量
	bandwidth, _ := data["bandwidth"].(float64)
	// avgBitrate, _ := data["avgBitrate"].(float64)  // 删除未使用的变量
	// maxBitrate, _ := data["maxBitrate"].(float64)  // 删除未使用的变量
	// profile, _ := data["profile"].(string)  // 删除未使用的变量
	// duration, _ := data["duration"].(float64)  // 删除未使用的变量
	// fnval, _ := data["fnval"].(float64)  // 删除未使用的变量
	// fnver, _ := data["fnver"].(float64)
	// fourk, _ := data["fourk"].(float64)
	// dashId, _ := data["dashId"].(string)
	// rotation, _ := data["rotation"].(float64)
	frameRateF, _ := strconv.ParseFloat(frameRate, 64)

	// 创建轨道
	track := &Track{
		ID:          int(id),
		Codecid:     int(codecid),
		Quality:     int(id), // 使用id作为quality
		Description: fmt.Sprintf("%dx%d", int(width), int(height)),
		URL:         baseUrl,
		Format:      mimeType,
		Codec:       codecs,
		Width:       int(width),
		Height:      int(height),
		Bandwidth:   int(bandwidth),
		FPS:         int(frameRateF),
		FrameType:   trackType,
	}

	return track
}

// getQualityDesc 获取画质描述
func getQualityDesc(qn int) string {
	switch qn {
	case 120:
		return "4K 超高清"
	case 116:
		return "1080P 高码率"
	case 112:
		return "1080P 高码率"
	case 80:
		return "1080P 高清"
	case 74:
		return "720P 60帧"
	case 72:
		return "720P 高清"
	case 64:
		return "480P 清晰"
	case 32:
		return "360P 流畅"
	case 16:
		return "240P 极速"
	default:
		return fmt.Sprintf("未知画质(%d)", qn)
	}
}

// getWBIKey 获取WBI密钥
func (p *Parser) getWBIKey() (string, error) {
	// WBI密钥获取逻辑暂未实现
	return "", nil
}
