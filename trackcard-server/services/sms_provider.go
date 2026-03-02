package services

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

type SMSProvider interface {
	SendCode(phoneCountryCode, phoneNumber, code, scene string) (bizID string, err error)
	Name() string
}

type MockSMSProvider struct{}

func (p *MockSMSProvider) SendCode(phoneCountryCode, phoneNumber, code, scene string) (string, error) {
	log.Printf("[SMS][MOCK] scene=%s phone=%s%s code=%s", scene, phoneCountryCode, phoneNumber, code)
	return fmt.Sprintf("mock-%d", time.Now().UnixNano()), nil
}

func (p *MockSMSProvider) Name() string { return "mock" }

type AliyunSMSProvider struct{}

func (p *AliyunSMSProvider) SendCode(phoneCountryCode, phoneNumber, code, scene string) (string, error) {
	accessKey := os.Getenv("ALIYUN_SMS_ACCESS_KEY_ID")
	secret := os.Getenv("ALIYUN_SMS_ACCESS_KEY_SECRET")
	sign := os.Getenv("ALIYUN_SMS_SIGN_NAME")
	if accessKey == "" || secret == "" || sign == "" {
		return "", fmt.Errorf("aliyun sms not configured")
	}
	log.Printf("[SMS][ALIYUN] TODO provider called scene=%s phone=%s%s", scene, phoneCountryCode, phoneNumber)
	return fmt.Sprintf("aliyun-%d", time.Now().UnixNano()), nil
}

func (p *AliyunSMSProvider) Name() string { return "aliyun" }

func NewSMSProvider() SMSProvider {
	if os.Getenv("SMS_PROVIDER") == "aliyun" {
		return &AliyunSMSProvider{}
	}
	return &MockSMSProvider{}
}

func Generate6DigitCode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}
