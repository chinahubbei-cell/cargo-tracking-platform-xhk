package services

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"trackcard-server/config"
)

type KuaihuoyunService struct {
	cid       string
	secretKey string
	baseURL   string

	// Secondary credentials
	cid2       string
	secretKey2 string

	// Devices using secondary credentials
	specialDevices map[string]bool
}

type DeviceInfo struct {
	Device       string   `json:"device"`
	Status       int      `json:"status"`
	PowerRate    int      `json:"powerRate"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Speed        float64  `json:"speed"`
	Direction    float64  `json:"direction"`
	LocateType   int      `json:"locateType"`
	LocateTime   int64    `json:"locateTime"`
	Mode         int      `json:"mode"`         // 0=静止模式 1=运动模式
	ReportedRate int      `json:"reportedRate"` // 上报周期（分钟）
	Temperature  *float64 `json:"temperature"`
	Humidity     *float64 `json:"humidity"`
}

type TrackData struct {
	Device      string   `json:"device"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Speed       float64  `json:"speed"`
	Direction   float64  `json:"direction"`
	LocateType  int      `json:"locateType"`
	LocateTime  int64    `json:"locateTime"`
	RunStatus   int      `json:"runStatus"`
	Temperature *float64 `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
}

var Kuaihuoyun *KuaihuoyunService

func InitKuaihuoyun() {
	cfg := config.AppConfig
	Kuaihuoyun = &KuaihuoyunService{
		cid:       cfg.KuaihuoyunCID,
		secretKey: cfg.KuaihuoyunSecretKey,
		baseURL:   cfg.KuaihuoyunBaseURL,

		// Hardcoded secondary config as per request
		cid2:       "1069",
		secretKey2: "ZINRIe5ohGyWSBFy",

		specialDevices: map[string]bool{
			"868120343599970": true,
			"868120343595788": true,
		},
	}
}

func (k *KuaihuoyunService) IsConfigured() bool {
	return k.cid != "" && k.secretKey != ""
}

// getCredentials 根据设备ID获取配置
func (k *KuaihuoyunService) getCredentials(deviceID string) (string, string) {
	if k.specialDevices[deviceID] {
		return k.cid2, k.secretKey2
	}
	return k.cid, k.secretKey
}

// generateToken 根据快货运API规则生成token
// token = md5(md5(cid + access_secret_key) + salt)
// salt格式为yyyyMMddHH (如: 2018090617)
// 注意: 第一步MD5和最终token都需要大写
func (k *KuaihuoyunService) generateToken(cid, secretKey string) string {
	// 第一步: md5(cid + secret_key) - 大写
	step1 := cid + secretKey
	hash1 := md5.Sum([]byte(step1))
	md5Step1 := strings.ToUpper(hex.EncodeToString(hash1[:])) // 必须大写

	// 第二步: 生成salt (yyyyMMddHH格式)
	salt := time.Now().Format("2006010215") // Go的时间格式: 2006=年, 01=月, 02=日, 15=时

	// 第三步: md5(md5结果 + salt) - 最终转大写
	step2 := md5Step1 + salt
	hash2 := md5.Sum([]byte(step2))
	token := strings.ToUpper(hex.EncodeToString(hash2[:]))

	log.Printf("[Kuaihuoyun] Token generation: cid=%s, salt=%s, step1=%s, token=%s", cid, salt, md5Step1, token)
	return token
}

func (k *KuaihuoyunService) request(endpoint string, params map[string]string, deviceID string) ([]byte, error) {
	// 获取对应设备的凭证
	cid, secret := k.getCredentials(deviceID)

	// 添加公共参数: cid 和 token
	params["cid"] = cid
	params["token"] = k.generateToken(cid, secret)

	// 构建URL
	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}

	reqURL := fmt.Sprintf("%s%s?%s", k.baseURL, endpoint, values.Encode())

	// 创建自定义HTTP客户端，禁用keep-alive防止EOF错误
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 15 * time.Second,
			IdleConnTimeout:       30 * time.Second,
		},
	}

	// 重试逻辑
	var lastErr error
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			time.Sleep(time.Duration(retry) * time.Second)
			log.Printf("[Kuaihuoyun] 重试请求 #%d: %s", retry, reqURL)
		}

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "TrackCard-Server/1.0")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Connection", "close")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[Kuaihuoyun] 请求失败 (重试 %d): %v", retry, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		return body, nil
	}

	return nil, fmt.Errorf("请求失败，已重试3次: %v", lastErr)
}

func (k *KuaihuoyunService) GetDeviceInfo(deviceID string) (*DeviceInfo, error) {
	if !k.IsConfigured() {
		return nil, fmt.Errorf("快货运API未配置")
	}

	params := map[string]string{
		"device": deviceID,
	}

	data, err := k.request("/get_device_info", params, deviceID)
	if err != nil {
		return nil, err
	}

	log.Printf("[Kuaihuoyun] GetDeviceInfo response: %s", string(data))

	var result struct {
		Code int        `json:"code"`
		Msg  string     `json:"msg"`
		Data DeviceInfo `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 原始数据: %s", err, string(data))
	}

	// 快货运API返回200表示成功
	if result.Code != 200 {
		return nil, fmt.Errorf("API错误[%d]: %s", result.Code, result.Msg)
	}

	return &result.Data, nil
}

func (k *KuaihuoyunService) GetDeviceInfoList(deviceIDs []string) ([]DeviceInfo, error) {
	if !k.IsConfigured() {
		return nil, fmt.Errorf("快货运API未配置")
	}

	// 批量接口不可用，使用单设备查询循环
	var results []DeviceInfo
	for _, deviceID := range deviceIDs {
		info, err := k.GetDeviceInfo(deviceID)
		if err != nil {
			log.Printf("[Kuaihuoyun] Failed to get device %s: %v", deviceID, err)
			continue
		}
		if info != nil {
			results = append(results, *info)
		}
	}

	log.Printf("[Kuaihuoyun] GetDeviceInfoList fetched %d/%d devices", len(results), len(deviceIDs))
	return results, nil
}

// GetTrack 获取设备轨迹数据
// 快货运API限制: 轨迹查询时间间隔不能超过24小时
// 此函数会自动将长时间范围分割成多个24小时以内的请求，并并发执行以提高速度
func (k *KuaihuoyunService) GetTrack(deviceID, startTime, endTime string) ([]TrackData, error) {
	if !k.IsConfigured() {
		return nil, fmt.Errorf("快货运API未配置")
	}

	// 解析时间
	layout := "2006-01-02 15:04:05"
	start, err := time.Parse(layout, startTime)
	if err != nil {
		return nil, fmt.Errorf("解析开始时间失败: %v", err)
	}
	end, err := time.Parse(layout, endTime)
	if err != nil {
		return nil, fmt.Errorf("解析结束时间失败: %v", err)
	}

	// 如果时间范围超过24小时，分割成多个请求
	// 使用并发请求提高速度
	const maxDuration = 23 * time.Hour // 使用23小时留一些余量
	type timeRange struct {
		start string
		end   string
	}
	var ranges []timeRange

	currentStart := start
	for currentStart.Before(end) {
		currentEnd := currentStart.Add(maxDuration)
		if currentEnd.After(end) {
			currentEnd = end
		}
		ranges = append(ranges, timeRange{
			start: currentStart.Format(layout),
			end:   currentEnd.Format(layout),
		})
		currentStart = currentEnd
	}

	// 设置并发限制 (例如最多5个并发请求)
	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)
	resultsChan := make(chan []TrackData, len(ranges))
	errChan := make(chan error, len(ranges))

	for _, r := range ranges {
		sem <- struct{}{} // 获取信号量
		go func(r timeRange) {
			defer func() { <-sem }() // 释放信号量
			tracks, err := k.getTrackSingle(deviceID, r.start, r.end)
			if err != nil {
				log.Printf("[Kuaihuoyun] GetTrack chunk failed (%s to %s): %v", r.start, r.end, err)
				// 不中断整体流程，只记录错误
				errChan <- err
				return
			}
			resultsChan <- tracks
		}(r)
	}

	// 等待所有goroutine完成 (这里简单使用循环等待结果)
	// 更好的方式是使用 sync.WaitGroup，但这里为了收集结果，直接计数也可以
	var allTracks []TrackData
	for i := 0; i < len(ranges); i++ {
		select {
		case tracks := <-resultsChan:
			allTracks = append(allTracks, tracks...)
		case <-errChan:
			// 错误已记录，忽略
		}
	}

	log.Printf("[Kuaihuoyun] GetTrack total: %d points for device %s (%s to %s)",
		len(allTracks), deviceID, startTime, endTime)
	return allTracks, nil
}

// getTrackSingle 执行单次轨迹查询（时间范围必须在24小时以内）
func (k *KuaihuoyunService) getTrackSingle(deviceID, startTime, endTime string) ([]TrackData, error) {
	params := map[string]string{
		"device":    deviceID,
		"startTime": startTime,
		"endTime":   endTime,
	}

	data, err := k.request("/get_track", params, deviceID)
	if err != nil {
		return nil, err
	}

	// 限制日志输出长度
	logData := string(data)
	if len(logData) > 500 {
		logData = logData[:500] + "... (truncated)"
	}
	log.Printf("[Kuaihuoyun] GetTrack response (%s to %s): %s", startTime, endTime, logData)

	var result struct {
		Code    int         `json:"code"`
		Msg     string      `json:"msg"`
		Message string      `json:"message"` // API有时用message而不是msg
		Data    []TrackData `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 原始数据: %s", err, string(data))
	}

	// 快货运API返回200表示成功
	if result.Code != 200 {
		errMsg := result.Msg
		if errMsg == "" {
			errMsg = result.Message
		}
		return nil, fmt.Errorf("API错误[%d]: %s", result.Code, errMsg)
	}

	return result.Data, nil
}
