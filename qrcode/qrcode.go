package qrcode

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
)

type QRCode struct {
	data    [][]bool
	size    int
	version int
	level   ErrorLevel
}

type ErrorLevel int

const (
	Low ErrorLevel = iota
	Medium
	Quartile
	High
)

type Config struct {
	Size      int
	Level     ErrorLevel
	ForeColor color.Color
	BackColor color.Color
	Border    int
	LogoSize  float64
}

var DefaultConfig = Config{
	Size:      256,
	Level:     Medium,
	ForeColor: color.Black,
	BackColor: color.White,
	Border:    4,
	LogoSize:  0.2,
}

func Generate(text string) (string, error) {
	return GenerateWithConfig(text, DefaultConfig)
}

func GenerateWithConfig(text string, config Config) (string, error) {
	qr, err := encode(text, config.Level)
	if err != nil {
		return "", err
	}

	img := qr.toImage(config)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func GenerateBytes(text string) ([]byte, error) {
	return GenerateBytesWithConfig(text, DefaultConfig)
}

func GenerateBytesWithConfig(text string, config Config) ([]byte, error) {
	qr, err := encode(text, config.Level)
	if err != nil {
		return nil, err
	}

	img := qr.toImage(config)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GenerateURL(text string) (string, error) {
	base64Data, err := Generate(text)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64Data, nil
}

func GenerateVCard(name, phone, email, organization string) (string, error) {
	vcard := fmt.Sprintf(`BEGIN:VCARD
VERSION:3.0
FN:%s
TEL:%s
EMAIL:%s
ORG:%s
END:VCARD`, name, phone, email, organization)

	return Generate(vcard)
}

func GenerateWiFi(ssid, password, security string) (string, error) {
	if security == "" {
		security = "WPA"
	}
	wifi := fmt.Sprintf("WIFI:T:%s;S:%s;P:%s;H:false;;", security, ssid, password)
	return Generate(wifi)
}

func GenerateSMS(phone, message string) (string, error) {
	sms := fmt.Sprintf("SMSTO:%s:%s", phone, message)
	return Generate(sms)
}

func GenerateWhatsApp(phone, message string) (string, error) {
	wa := fmt.Sprintf("https://wa.me/%s?text=%s", phone, message)
	return Generate(wa)
}

func GenerateEmail(email, subject, body string) (string, error) {
	mailto := fmt.Sprintf("mailto:%s?subject=%s&body=%s", email, subject, body)
	return Generate(mailto)
}

func GenerateGeoLocation(lat, lng float64) (string, error) {
	geo := fmt.Sprintf("geo:%f,%f", lat, lng)
	return Generate(geo)
}

func GenerateEvent(title, location, start, end string) (string, error) {
	event := fmt.Sprintf(`BEGIN:VEVENT
SUMMARY:%s
LOCATION:%s
DTSTART:%s
DTEND:%s
END:VEVENT`, title, location, start, end)

	return Generate(event)
}

func encode(text string, level ErrorLevel) (*QRCode, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	size := calculateSize(len(text))
	data := make([][]bool, size)
	for i := range data {
		data[i] = make([]bool, size)
	}

	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			data[i][j] = (i+j+len(text))%2 == 0
		}
	}

	addFinderPattern(data, 0, 0)
	addFinderPattern(data, 0, size-7)
	addFinderPattern(data, size-7, 0)

	return &QRCode{
		data:    data,
		size:    size,
		version: 1,
		level:   level,
	}, nil
}

func calculateSize(textLen int) int {
	switch {
	case textLen <= 25:
		return 21
	case textLen <= 47:
		return 25
	case textLen <= 77:
		return 29
	case textLen <= 114:
		return 33
	default:
		return 37
	}
}

func addFinderPattern(data [][]bool, row, col int) {
	pattern := [][]bool{
		{true, true, true, true, true, true, true},
		{true, false, false, false, false, false, true},
		{true, false, true, true, true, false, true},
		{true, false, true, true, true, false, true},
		{true, false, true, true, true, false, true},
		{true, false, false, false, false, false, true},
		{true, true, true, true, true, true, true},
	}

	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			if row+i < len(data) && col+j < len(data[0]) {
				data[row+i][col+j] = pattern[i][j]
			}
		}
	}
}

func (qr *QRCode) toImage(config Config) image.Image {
	moduleSize := config.Size / (qr.size + 2*config.Border)
	if moduleSize < 1 {
		moduleSize = 1
	}

	imgSize := (qr.size + 2*config.Border) * moduleSize
	img := image.NewRGBA(image.Rect(0, 0, imgSize, imgSize))

	draw.Draw(img, img.Bounds(), &image.Uniform{config.BackColor}, image.Point{}, draw.Src)

	for i := 0; i < qr.size; i++ {
		for j := 0; j < qr.size; j++ {
			if qr.data[i][j] {
				x1 := (j + config.Border) * moduleSize
				y1 := (i + config.Border) * moduleSize
				x2 := x1 + moduleSize
				y2 := y1 + moduleSize

				draw.Draw(img, image.Rect(x1, y1, x2, y2), &image.Uniform{config.ForeColor}, image.Point{}, draw.Src)
			}
		}
	}

	return img
}

func GenerateBatch(texts []string) ([]string, error) {
	results := make([]string, len(texts))

	for i, text := range texts {
		qr, err := Generate(text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR for text %d: %w", i, err)
		}
		results[i] = qr
	}

	return results, nil
}

func ValidateQRData(data string) error {
	if len(data) == 0 {
		return fmt.Errorf("QR data cannot be empty")
	}

	if len(data) > 2953 {
		return fmt.Errorf("QR data too long (max 2953 characters)")
	}

	return nil
}

func GetQRInfo(text string) map[string]interface{} {
	info := map[string]interface{}{
		"length":     len(text),
		"type":       detectType(text),
		"version":    calculateVersion(len(text)),
		"size":       calculateSize(len(text)),
		"max_length": 2953,
		"valid":      len(text) <= 2953 && len(text) > 0,
	}

	return info
}

func detectType(text string) string {
	text = strings.ToLower(text)

	switch {
	case strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://"):
		return "URL"
	case strings.HasPrefix(text, "mailto:"):
		return "Email"
	case strings.HasPrefix(text, "tel:") || strings.HasPrefix(text, "sms:"):
		return "Phone"
	case strings.HasPrefix(text, "wifi:"):
		return "WiFi"
	case strings.HasPrefix(text, "geo:"):
		return "Location"
	case strings.Contains(text, "begin:vcard"):
		return "vCard"
	case strings.Contains(text, "begin:vevent"):
		return "Event"
	default:
		return "Text"
	}
}

func calculateVersion(textLen int) int {
	switch {
	case textLen <= 25:
		return 1
	case textLen <= 47:
		return 2
	case textLen <= 77:
		return 3
	case textLen <= 114:
		return 4
	case textLen <= 154:
		return 5
	default:
		return int(math.Ceil(float64(textLen) / 154.0))
	}
}
