package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trackcard-server/models"
)

var (
	chineseAdminPattern  = regexp.MustCompile(`[\p{Han}]{2,12}?(自治州|地区|盟|市|县|区|旗)`)
	chineseCityPattern   = regexp.MustCompile(`[\p{Han}]{2,16}?(自治州|地区|盟|市)`)
	englishAdminKeywords = []string{"city", "county", "district", "prefecture", "region", "state", "province"}
	englishNoiseTokens   = map[string]struct{}{
		"town": {}, "township": {}, "village": {}, "road": {}, "street": {}, "avenue": {},
		"highway": {}, "expressway": {}, "bridge": {}, "service": {}, "area": {}, "parking": {},
	}
	chineseNoiseSuffixes = []string{"小区", "园区", "社区", "校区", "景区", "开发区", "服务区", "园", "站", "高速"}
)

func main() {
	shipmentID := flag.String("shipment", "", "仅回填指定运单")
	dryRun := flag.Bool("dry-run", false, "仅预览不写库")
	force := flag.Bool("force", false, "强制重建所有命中运单（默认仅处理缓存<=1条）")
	flag.Parse()

	dbPath, err := resolveDBPath()
	if err != nil {
		log.Fatalf("定位数据库失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	shipmentIDs, err := loadTargetShipmentIDs(db, strings.TrimSpace(*shipmentID), *force)
	if err != nil {
		log.Fatalf("查询目标运单失败: %v", err)
	}
	log.Printf("待处理运单: %d (shipment=%s, force=%v, dryRun=%v)", len(shipmentIDs), strings.TrimSpace(*shipmentID), *force, *dryRun)

	var rebuilt, skipped, failed int
	for _, id := range shipmentIDs {
		records, err := buildTransitCitiesFromStops(db, id)
		if err != nil {
			failed++
			log.Printf("❌ 构建失败 shipment=%s err=%v", id, err)
			continue
		}
		if len(records) == 0 {
			skipped++
			continue
		}

		log.Printf("🔧 回填运单 shipment=%s cities=%d", id, len(records))

		if !*dryRun {
			err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Where("shipment_id = ?", id).Delete(&models.TransitCityRecord{}).Error; err != nil {
					return err
				}
				return tx.Create(&records).Error
			})
			if err != nil {
				failed++
				log.Printf("❌ 写库失败 shipment=%s err=%v", id, err)
				continue
			}
		}

		rebuilt++
	}

	log.Printf("完成: rebuilt=%d skipped=%d failed=%d", rebuilt, skipped, failed)
}

func resolveDBPath() (string, error) {
	candidates := []string{
		"trackcard.db",
		"../trackcard-server/trackcard.db",
		"/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk/trackcard-server/trackcard.db",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

func loadTargetShipmentIDs(db *gorm.DB, shipmentID string, force bool) ([]string, error) {
	if shipmentID != "" {
		return []string{shipmentID}, nil
	}

	var rows []struct {
		ShipmentID string
		CityCount  int
	}

	query := db.Table("shipments s").
		Select("s.id AS shipment_id, COUNT(t.id) AS city_count").
		Joins("LEFT JOIN transit_city_records t ON t.shipment_id = s.id AND t.deleted_at IS NULL").
		Where("s.deleted_at IS NULL").
		Group("s.id")

	if !force {
		query = query.Having("COUNT(t.id) <= 1")
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ShipmentID) == "" {
			continue
		}
		ids = append(ids, row.ShipmentID)
	}
	sort.Strings(ids)
	return ids, nil
}

func buildTransitCitiesFromStops(db *gorm.DB, shipmentID string) ([]models.TransitCityRecord, error) {
	var stops []models.DeviceStopRecord
	if err := db.
		Where("shipment_id = ?", shipmentID).
		Order("start_time ASC").
		Find(&stops).Error; err != nil {
		return nil, err
	}

	visited := make(map[string]struct{}, 16)
	records := make([]models.TransitCityRecord, 0, 16)

	for _, stop := range stops {
		if stop.Latitude == nil || stop.Longitude == nil {
			continue
		}

		country, city := parseCountryCityFromStopAddress(stop.Address, *stop.Latitude, *stop.Longitude)
		if country == "" {
			country = inferTransitCountryFromCoordinate(*stop.Latitude, *stop.Longitude)
		}
		country = normalizeTransitCountry(country)
		if country == "" {
			country = "国际区域"
		}
		city = normalizeTransitCity(city)
		if city == "" {
			city = fallbackTransitCityFromCoordinate(*stop.Latitude, *stop.Longitude)
		}
		if country == "" || city == "" {
			continue
		}

		key := strings.ToLower(country) + "|" + strings.ToLower(city)
		if _, ok := visited[key]; ok {
			continue
		}
		visited[key] = struct{}{}

		records = append(records, models.TransitCityRecord{
			ShipmentID: shipmentID,
			DeviceID:   stop.DeviceID,
			Country:    country,
			City:       city,
			Latitude:   *stop.Latitude,
			Longitude:  *stop.Longitude,
			EnteredAt:  stop.StartTime,
			IsOversea:  !isChinaCountry(country),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
	}

	return records, nil
}

func fallbackTransitCityFromCoordinate(lat, lng float64) string {
	const step = 0.5
	latGrid := math.Round(lat/step) * step
	lngGrid := math.Round(lng/step) * step
	return fmt.Sprintf("坐标区域(%.1f,%.1f)", latGrid, lngGrid)
}

func parseCountryCityFromStopAddress(address string, lat, lng float64) (string, string) {
	text := strings.TrimSpace(address)
	if text == "" {
		return inferTransitCountryFromCoordinate(lat, lng), ""
	}

	parts := strings.Split(text, "/")
	zhPart := strings.TrimSpace(parts[0])
	enPart := ""
	if len(parts) > 1 {
		enPart = strings.TrimSpace(parts[1])
	}

	country := inferTransitCountryFromText(strings.Join([]string{zhPart, enPart}, " "))
	if country == "" {
		country = inferTransitCountryFromCoordinate(lat, lng)
	}

	city := extractTransitCityFromChinese(zhPart)
	if candidate := extractTransitCityFromEnglish(enPart); candidate != "" {
		city = pickBetterTransitCityCandidate(city, candidate)
	}
	if candidate := extractTransitCityFromEnglish(zhPart); candidate != "" {
		city = pickBetterTransitCityCandidate(city, candidate)
	}

	return country, city
}

func pickBetterTransitCityCandidate(current, candidate string) string {
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	currentDistrictLike := isTransitDistrictLike(current)
	candidateDistrictLike := isTransitDistrictLike(candidate)
	if currentDistrictLike && !candidateDistrictLike {
		return candidate
	}
	if !hasChineseCharacters(current) && hasChineseCharacters(candidate) {
		return candidate
	}
	return current
}

func inferTransitCountryFromText(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return ""
	}

	aliases := map[string]string{
		"china": "中国", "prc": "中国", "中国": "中国", "中华人民共和国": "中国",
		"hong kong": "中国", "hongkong": "中国", "香港": "中国",
		"macau": "中国", "macao": "中国", "澳门": "中国",
		"taiwan": "中国", "台湾": "中国",
		"russia": "俄罗斯", "俄罗斯": "俄罗斯",
		"mongolia": "蒙古", "蒙古": "蒙古",
		"kazakhstan": "哈萨克斯坦", "哈萨克斯坦": "哈萨克斯坦",
		"united states": "美国", "usa": "美国", "america": "美国", "美国": "美国",
		"canada": "加拿大", "加拿大": "加拿大",
		"germany": "德国", "德国": "德国",
		"france": "法国", "法国": "法国",
		"spain": "西班牙", "西班牙": "西班牙",
		"italy": "意大利", "意大利": "意大利",
		"netherlands": "荷兰", "荷兰": "荷兰",
		"belgium": "比利时", "比利时": "比利时",
		"japan": "日本", "日本": "日本",
		"korea": "韩国", "south korea": "韩国", "韩国": "韩国",
		"thailand": "泰国", "泰国": "泰国",
		"vietnam": "越南", "越南": "越南",
		"singapore": "新加坡", "新加坡": "新加坡",
		"malaysia": "马来西亚", "马来西亚": "马来西亚",
		"indonesia": "印度尼西亚", "印尼": "印度尼西亚", "印度尼西亚": "印度尼西亚",
	}

	for key, value := range aliases {
		if strings.Contains(lower, key) {
			return value
		}
	}
	return ""
}

func inferTransitCountryFromCoordinate(lat, lng float64) string {
	if lat >= 18.0 && lat <= 54.0 && lng >= 73.0 && lng <= 136.0 {
		return "中国"
	}
	return ""
}

func extractTransitCityFromChinese(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	matches := chineseAdminPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}

	cityLevel := make([]string, 0, len(matches))
	districtLevel := make([]string, 0, len(matches))

	for _, match := range matches {
		candidate := strings.TrimSpace(match)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, "中华人民共和国") || candidate == "中国" {
			continue
		}

		skip := false
		for _, suffix := range chineseNoiseSuffixes {
			if strings.HasSuffix(candidate, suffix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		switch {
		case strings.HasSuffix(candidate, "市"),
			strings.HasSuffix(candidate, "自治州"),
			strings.HasSuffix(candidate, "地区"),
			strings.HasSuffix(candidate, "盟"):
			cityLevel = append(cityLevel, candidate)
		case strings.HasSuffix(candidate, "县"),
			strings.HasSuffix(candidate, "区"),
			strings.HasSuffix(candidate, "旗"):
			districtLevel = append(districtLevel, candidate)
		}
	}

	if len(cityLevel) > 0 {
		return cityLevel[0]
	}
	if len(districtLevel) > 0 {
		return districtLevel[0]
	}
	return ""
}

func extractTransitCityFromEnglish(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	segments := splitEnglishAddressSegments(text)
	for _, keyword := range englishAdminKeywords {
		for _, segment := range segments {
			if candidate := extractEnglishAdminByKeyword(segment, keyword); candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func splitEnglishAddressSegments(text string) []string {
	replacer := strings.NewReplacer("，", ",", ";", ",", "；", ",", "/", ",", "(", " ", ")", " ", "（", " ", "）", " ")
	parts := strings.Split(replacer.Replace(text), ",")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	if len(segments) == 0 {
		return []string{text}
	}
	return segments
}

func extractEnglishAdminByKeyword(segment string, keyword string) string {
	words := strings.FieldsFunc(segment, func(r rune) bool {
		return !(unicode.IsLetter(r) || r == '\'' || r == '-')
	})
	for i, word := range words {
		if !strings.EqualFold(word, keyword) || i == 0 || i != len(words)-1 {
			continue
		}

		start := i - 1
		for lookback := i - 2; lookback >= 0 && i-lookback <= 3; lookback-- {
			if _, noise := englishNoiseTokens[strings.ToLower(words[lookback])]; noise {
				break
			}
			start = lookback
		}

		candidate := strings.Join(words[start:i+1], " ")
		return strings.TrimSpace(candidate)
	}
	return ""
}

func normalizeTransitCountry(country string) string {
	country = strings.TrimSpace(country)
	switch strings.ToLower(country) {
	case "中华人民共和国":
		return "中国"
	case "china", "people's republic of china", "prc":
		return "中国"
	default:
		return country
	}
}

func normalizeTransitCity(city string) string {
	city = strings.TrimSpace(city)
	city = strings.Trim(city, ",;")
	switch strings.ToLower(city) {
	case "", "市辖区":
		return ""
	}
	if strings.HasPrefix(city, "坐标区域(") {
		return city
	}

	if hasChineseCharacters(city) {
		normalized := strings.Join(strings.Fields(city), "")
		if normalized == "" {
			return ""
		}
		if preferred := extractChineseTransitCity(normalized); preferred != "" {
			return preferred
		}
		if hasTransitCitySuffix(normalized) {
			return normalized
		}
		return normalized + "市"
	}

	cleaned := strings.NewReplacer("，", " ", ",", " ", ";", " ", "；", " ").Replace(city)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return ""
	}

	return toTransitTitleCase(cleaned)
}

func hasTransitCitySuffix(city string) bool {
	for _, suffix := range []string{"自治州", "地区", "盟", "市", "县", "区", "旗", "省", "特别行政区"} {
		if strings.HasSuffix(city, suffix) {
			return true
		}
	}
	return false
}

func extractChineseTransitCity(text string) string {
	matches := chineseCityPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}
	for i := len(matches) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(matches[i])
		if candidate == "" || candidate == "市辖区" {
			continue
		}
		return candidate
	}
	return ""
}

func isTransitDistrictLike(city string) bool {
	value := strings.TrimSpace(city)
	if value == "" || strings.HasPrefix(value, "坐标区域(") {
		return false
	}
	if hasChineseCharacters(value) {
		return strings.HasSuffix(value, "区") || strings.HasSuffix(value, "县") || strings.HasSuffix(value, "旗")
	}
	lower := strings.ToLower(value)
	return strings.HasSuffix(lower, " district") || strings.HasSuffix(lower, " county")
}

func hasChineseCharacters(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func toTransitTitleCase(text string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func isChinaCountry(country string) bool {
	normalized := strings.ToLower(strings.TrimSpace(country))
	switch normalized {
	case "中国", "中华人民共和国", "china", "people's republic of china", "prc", "cn":
		return true
	default:
		return false
	}
}
