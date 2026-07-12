package main

import (
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ========== Helper Functions ==========

func ptrInt32(v int32) *int32       { return &v }
func ptrString(v string) *string    { return &v }
func ptrFloat64(v float64) *float64 { return &v }
func ptrFloat32(v float32) *float32 { return &v }

// safeIntToInt32 safely converts int to int32 with overflow check
func safeIntToInt32(v int) int32 {
	if v > 2147483647 {
		return 2147483647
	}
	if v < -2147483648 {
		return -2147483648
	}
	return int32(v)
}

// isValidServiceURL validates that a URL points to an allowed internal service
func isValidServiceURL(url string, allowedPrefixes ...string) bool {
	// Must start with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}
	// Check against allowed prefixes
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func grpcToHTTPStatus(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, "Внутренняя ошибка сервера"
	}
	return mapGRPCToHTTP(st)
}

type grpcHTTPMapping struct {
	status    int
	msg       string
	translate bool
}

var grpcCodeToHTTP = map[codes.Code]grpcHTTPMapping{
	codes.OK:                 {http.StatusOK, "", false},
	codes.Canceled:           {http.StatusRequestTimeout, "Запрос отменён", false},
	codes.InvalidArgument:    {http.StatusBadRequest, "", true},
	codes.NotFound:           {http.StatusNotFound, "Не найдено", false},
	codes.AlreadyExists:      {http.StatusConflict, "", true},
	codes.PermissionDenied:   {http.StatusNotFound, "Не найдено", false},
	codes.Unauthenticated:    {http.StatusUnauthorized, "Неверные учётные данные", false},
	codes.ResourceExhausted:  {http.StatusTooManyRequests, "Превышен лимит запросов", false},
	codes.FailedPrecondition: {http.StatusBadRequest, "", true},
	codes.Aborted:            {http.StatusConflict, "Операция прервана", false},
	codes.OutOfRange:         {http.StatusBadRequest, "", true},
	codes.Unimplemented:      {http.StatusNotImplemented, "Функция не реализована", false},
	codes.Internal:           {http.StatusInternalServerError, "Внутренняя ошибка сервера", false},
	codes.Unavailable:        {http.StatusServiceUnavailable, "Сервис временно недоступен", false},
	codes.DataLoss:           {http.StatusInternalServerError, "Потеря данных", false},
	codes.DeadlineExceeded:   {http.StatusGatewayTimeout, "Превышено время ожидания", false},
	codes.Unknown:            {http.StatusInternalServerError, "", true},
}

func mapGRPCToHTTP(st *status.Status) (int, string) {
	msg := st.Message()
	if m, ok := grpcCodeToHTTP[st.Code()]; ok {
		if m.translate {
			return m.status, translateError(msg)
		}
		return m.status, m.msg
	}
	return http.StatusInternalServerError, translateError(msg)
}

// translateError converts technical error messages to user-friendly Russian
func translateError(msg string) string {
	translations := map[string]string{
		"email is required":                   "Укажите email",
		"password is required":                "Укажите пароль",
		"full name is required":               "Укажите имя",
		"invalid role":                        "Недопустимая роль",
		"invalid email format":                "Некорректный формат email",
		"password must be at least":           "Пароль должен быть не менее 8 символов",
		"user_id is required":                 "Необходима авторизация",
		"age must be between":                 "Возраст должен быть от 0 до 150",
		"height_cm must be between":           "Рост должен быть от 50 до 300 см",
		"weight_kg must be between":           "Вес должен быть от 1 до 500 кг",
		"fitness_level must be":               "Выберите уровень подготовки",
		"user not found":                      "Пользователь не найден",
		"email already exists":                "Этот email уже зарегистрирован",
		"invalid credentials":                 "Неверный email или пароль",
		"user already exists":                 "Этот email уже зарегистрирован",
		"value cannot be negative":            "Значение не может быть отрицательным",
		"metric_type is required":             "Укажите тип метрики",
		"invalid metric data":                 "Некорректные данные метрики",
		"heart_rate out of valid range":       "Пульс вне допустимого диапазона (30–220)",
		"spo2 out of valid range":             "SpO₂ вне допустимого диапазона (70–100)",
		"metric_type not found":               "Тип метрики не найден",
		"user already has an active plan":     "У вас уже есть активная программа тренировок",
		"user already has a connected device": "У вас уже есть подключённое устройство",
	}
	for pattern, translated := range translations {
		if containsIgnoreCase(msg, pattern) {
			return translated
		}
	}
	// Если не нашли перевод — возвращаем как есть
	return msg
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				containsSubstringIgnoreCase(s, substr))
}

func containsSubstringIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
