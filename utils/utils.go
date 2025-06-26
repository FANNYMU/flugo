package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}

func RandomInt(min, max int) int {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	n := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if n < 0 {
		n = -n
	}
	return min + n%(max-min+1)
}

func UUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:])
}

func MD5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func SHA256(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ContainsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func UniqueInts(slice []int) []int {
	seen := make(map[int]bool)
	result := []int{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func Truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

func TruncateWords(s string, words int) string {
	wordSlice := strings.Fields(s)
	if len(wordSlice) <= words {
		return s
	}
	return strings.Join(wordSlice[:words], " ") + "..."
}

func Slug(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func CamelCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsDigit(c)
	})

	result := ""
	for i, word := range words {
		if i == 0 {
			result += strings.ToLower(word)
		} else {
			result += strings.Title(strings.ToLower(word))
		}
	}
	return result
}

func PascalCase(s string) string {
	words := strings.FieldsFunc(s, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsDigit(c)
	})

	result := ""
	for _, word := range words {
		result += strings.Title(strings.ToLower(word))
	}
	return result
}

func SnakeCase(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

func KebabCase(s string) string {
	return Slug(s)
}

func IsEmail(email string) bool {
	regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return regex.MatchString(email)
}

func IsURL(url string) bool {
	regex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return regex.MatchString(url)
}

func IsPhone(phone string) bool {
	regex := regexp.MustCompile(`^[\+]?[1-9][\d]{0,15}$`)
	return regex.MatchString(strings.ReplaceAll(phone, " ", ""))
}

func IsAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func ToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func ToFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func ToString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func ToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func FromJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

func Round(f float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(f*multiplier) / multiplier
}

func Ceil(f float64) int {
	return int(math.Ceil(f))
}

func Floor(f float64) int {
	return int(math.Floor(f))
}

func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func AbsFloat(f float64) float64 {
	return math.Abs(f)
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinFloat(a, b float64) float64 {
	return math.Min(a, b)
}

func MaxFloat(a, b float64) float64 {
	return math.Max(a, b)
}

func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func ClampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func MapStrings(slice []string, fn func(string) string) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

func FilterStrings(slice []string, fn func(string) bool) []string {
	result := []string{}
	for _, v := range slice {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

func ReduceStrings(slice []string, fn func(string, string) string, initial string) string {
	result := initial
	for _, v := range slice {
		result = fn(result, v)
	}
	return result
}

func Chunk(slice []string, size int) [][]string {
	if size <= 0 {
		return nil
	}

	var chunks [][]string
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

func Paginate(slice []string, page, perPage int) ([]string, int, int) {
	total := len(slice)
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage

	if start > total {
		return []string{}, page, totalPages
	}
	if end > total {
		end = total
	}

	return slice[start:end], page, totalPages
}

func Struct2Map(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return result
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // Skip unexported fields
			continue
		}

		key := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				key = parts[0]
			}
		}

		result[key] = v.Field(i).Interface()
	}

	return result
}

func Map2Struct(m map[string]interface{}, obj interface{}) error {
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, obj)
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

func HumanizeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	}
	if diff < 30*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}
	if diff < 365*24*time.Hour {
		return fmt.Sprintf("%d months ago", int(diff.Hours()/(24*30)))
	}
	return fmt.Sprintf("%d years ago", int(diff.Hours()/(24*365)))
}

func ParseDate(dateStr, layout string) (time.Time, error) {
	return time.Parse(layout, dateStr)
}

func FormatDate(t time.Time, layout string) string {
	return t.Format(layout)
}

func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

func AddMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

func AddYears(t time.Time, years int) time.Time {
	return t.AddDate(years, 0, 0)
}

func GetDaysDiff(t1, t2 time.Time) int {
	return int(t2.Sub(t1).Hours() / 24)
}

func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func If(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func IfString(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

func IfInt(condition bool, trueVal, falseVal int) int {
	if condition {
		return trueVal
	}
	return falseVal
}

func Coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func CoalesceInt(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

func IsEmpty(v interface{}) bool {
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Slice, reflect.Array, reflect.Map:
		return val.Len() == 0
	case reflect.Ptr:
		return val.IsNil()
	default:
		return false
	}
}

func IsNotEmpty(v interface{}) bool {
	return !IsEmpty(v)
}
