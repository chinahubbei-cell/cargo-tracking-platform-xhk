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
)

func generateToken(cid, secretKey string) string {
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

    log.Printf("Token generation: cid=%s, salt=%s, step1=%s, token=%s", cid, salt, md5Step1, token)
    return token
}

func request(baseURL, endpoint string, params map[string]string, cid, secret string) ([]byte, error) {
    // 添加公共参数: cid 和 token
    params["cid"] = cid
    params["token"] = generateToken(cid, secret)

    // 构建URL
    values := url.Values{}
    for key, value := range params {
        values.Add(key, value)
    }

    reqURL := fmt.Sprintf("%s%s?%s", baseURL, endpoint, values.Encode())
    log.Printf("Request URL: %s", reqURL)

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
            log.Printf("重试请求 #%d: %s", retry, reqURL)
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
            log.Printf("请求失败 (重试 %d): %v", retry, err)
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
    
    // 第一组凭证
    cid1 := "1067"
    secret1 := "O0DcRZTWrCwZqL44"
    
    // 第二组凭证
    cid2 := "1069"
    secret2 := "ZINRIe5ohGyWSBFy"
    
    devices := []string{"868120343599970", "868120343595788"}
    
    for _, device := range devices {
        log.Printf("=== 测试设备 %s ===", device)
        // 决定使用哪组凭证
        cid, secret := cid1, secret1
        if device == "868120343599970" || device == "868120343595788" {
            cid, secret = cid2, secret2
            log.Printf("使用第二组凭证 (cid=%s)", cid)
        }
        
        params := map[string]string{
            "device": device,
        }
        
        data, err := request(baseURL, "/get_device_info", params, cid, secret)
        if err != nil {
            log.Printf("设备 %s 请求失败: %v", device, err)
            continue
        }
        
        log.Printf("设备 %s 响应: %s", device, string(data))
    }
}