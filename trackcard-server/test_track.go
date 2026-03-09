package main

import (
    "fmt"
    "log"
    "time"
    "crypto/md5"
    "encoding/hex"
    "strings"
    "net/http"
    "net/url"
    "io"
    "encoding/json"
)

func generateToken(cid, secretKey string) string {
    step1 := cid + secretKey
    hash1 := md5.Sum([]byte(step1))
    md5Step1 := strings.ToUpper(hex.EncodeToString(hash1[:]))
    salt := time.Now().Format("2006010215")
    step2 := md5Step1 + salt
    hash2 := md5.Sum([]byte(step2))
    token := strings.ToUpper(hex.EncodeToString(hash2[:]))
    return token
}

func request(baseURL, endpoint string, params map[string]string, cid, secret string) ([]byte, error) {
    params["cid"] = cid
    params["token"] = generateToken(cid, secret)
    values := url.Values{}
    for key, value := range params {
        values.Add(key, value)
    }
    reqURL := fmt.Sprintf("%s%s?%s", baseURL, endpoint, values.Encode())
    log.Printf("Request URL: %s", reqURL)
    client := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            DisableKeepAlives:     true,
            ResponseHeaderTimeout: 15 * time.Second,
            IdleConnTimeout:       30 * time.Second,
        },
    }
    var lastErr error
    for retry := 0; retry < 3; retry++ {
        if retry > 0 {
            time.Sleep(time.Duration(retry) * time.Second)
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

func main() {
    baseURL := "http://bapi.kuaihuoyun.com/api/v1"
    cid2 := "1069"
    secret2 := "ZINRIe5ohGyWSBFy"
    
    devices := []string{"868120343599970", "868120343595788"}
    
    for _, device := range devices {
        log.Printf("=== 测试设备 %s 轨迹 ===", device)
        // 查询最近24小时
        endTime := time.Now()
        startTime := endTime.Add(-24 * time.Hour)
        startStr := startTime.Format("2006-01-02 15:04:05")
        endStr := endTime.Format("2006-01-02 15:04:05")
        
        params := map[string]string{
            "device":    device,
            "startTime": startStr,
            "endTime":   endStr,
        }
        
        data, err := request(baseURL, "/get_track", params, cid2, secret2)
        if err != nil {
            log.Printf("设备 %s 轨迹请求失败: %v", device, err)
            continue
        }
        
        // 解析响应
        var result struct {
            Code    int                    `json:"code"`
            Msg     string                 `json:"msg"`
            Message string                 `json:"message"`
            Data    []interface{}          `json:"data"`
        }
        if err := json.Unmarshal(data, &result); err != nil {
            log.Printf("解析响应失败: %v, 原始数据: %s", err, string(data))
            continue
        }
        
        log.Printf("设备 %s 轨迹响应: code=%d, msg=%s, 数据点数=%d", device, result.Code, result.Msg, len(result.Data))
        if result.Code != 200 {
            log.Printf("错误: %s", result.Message)
        }
    }
}