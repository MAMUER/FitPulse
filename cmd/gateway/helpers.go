package main

import (
	"net/http"
	"strings"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
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

// grpcToHTTPStatus maps gRPC error codes to HTTP status codes.
// Returns the mapped HTTP status code and a user-friendly message in Russian.
func grpcToHTTPStatus(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, "Внутренняя ошибка сервера"
	}
	msg := st.Message()
	// Переводим технические сообщения на русский
	switch st.Code() {
	case codes.OK:
		return http.StatusOK, ""
	case codes.Canceled:
		return http.StatusRequestTimeout, "Запрос отменён"
	case codes.InvalidArgument:
		return http.StatusBadRequest, translateError(msg)
	case codes.NotFound:
		return http.StatusNotFound, "Не найдено"
	case codes.AlreadyExists:
		return http.StatusConflict, translateError(msg)
	case codes.PermissionDenied:
		// Требование #4: Никогда не возвращаем 403 — заменяем на 404
		return http.StatusNotFound, "Не найдено"
	case codes.Unauthenticated:
		return http.StatusUnauthorized, "Неверные учётные данные"
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests, "Превышен лимит запросов"
	case codes.FailedPrecondition:
		return http.StatusBadRequest, translateError(msg)
	case codes.Aborted:
		return http.StatusConflict, "Операция прервана"
	case codes.OutOfRange:
		return http.StatusBadRequest, translateError(msg)
	case codes.Unimplemented:
		return http.StatusNotImplemented, "Функция не реализована"
	case codes.Internal:
		return http.StatusInternalServerError, "Внутренняя ошибка сервера"
	case codes.Unavailable:
		return http.StatusServiceUnavailable, "Сервис временно недоступен"
	case codes.DataLoss:
		return http.StatusInternalServerError, "Потеря данных"
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout, "Превышено время ожидания"
	case codes.Unknown:
		return http.StatusInternalServerError, translateError(msg)
	default:
		return http.StatusInternalServerError, translateError(msg)
	}
}

// translateError converts technical error messages to user-friendly Russian
func translateError(msg string) string {
	translations := map[string]string{ // #nosec G101
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

// extractFeatures извлекает фичи из биометрических данных для ML-классификации
func extractMLPayload(bioResp *biometricpb.BiometricRecord) map[string]interface{} {
	// Дефолтные значения при отсутствии данных
	heartRate := 70.0
	hrv := 50.0
	spo2 := 98.0
	temp := 36.6
	bpSystolic := 120.0
	bpDiastolic := 80.0
	sleepHours := 7.0

	if bioResp != nil {
		switch bioResp.MetricType {
		case "heart_rate":
			heartRate = bioResp.Value
		case "hrv":
			hrv = bioResp.Value
		case "spo2":
			spo2 = bioResp.Value
		case "temperature":
			temp = bioResp.Value
		case "systolic_pressure":
			bpSystolic = bioResp.Value
		case "diastolic_pressure":
			bpDiastolic = bioResp.Value
		case "sleep_hours":
			sleepHours = bioResp.Value
		}
	}

	return map[string]interface{}{
		"physiological_data": map[string]float64{
			"heart_rate":               heartRate,
			"heart_rate_variability":   hrv,
			"spo2":                     spo2,
			"temperature":              temp,
			"blood_pressure_systolic":  bpSystolic,
			"blood_pressure_diastolic": bpDiastolic,
			"sleep_hours":              sleepHours,
		},
	}
}
