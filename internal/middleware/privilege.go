// Package middleware provides HTTP middleware for authentication, logging, and security.
package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"go.uber.org/zap"
)

// PrivilegeKey — ключ для хранения подтверждённой роли в контексте
type PrivilegeKey struct{}

// RequirePrivilege проверяет привилегию пользователя непосредственно в БД
// Требование #10: Серверная валидация привилегий
// Требование #8: Middleware выполняется ДО основного обработчика
func RequirePrivilege(db *sql.DB, requiredRole string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok {
				// Требование #4: 404 вместо 403
				http.Error(w, "not found", http.StatusNotFound)
				return // Требование #7
			}

			// Требование #10: Повторная проверка роли в БД на момент выполнения операции
			var actualRole string
			err := db.QueryRowContext(r.Context(),
				"SELECT role FROM users WHERE id = $1",
				userID,
			).Scan(&actualRole)

			if err == sql.ErrNoRows {
				log.Warn("User not found during privilege check", zap.String("user_id", userID))
				http.Error(w, "not found", http.StatusNotFound)
				return // Требование #7
			}
			if err != nil {
				log.Error("Database error during privilege check", zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return // Требование #7
			}

			// Требование #10: Проверяем актуальную роль из БД
			if actualRole != requiredRole {
				log.Warn("Insufficient privileges",
					zap.String("user_id", userID),
					zap.String("required_role", requiredRole),
					zap.String("actual_role", actualRole),
				)
				http.Error(w, "not found", http.StatusNotFound) // Требование #4
				return                                          // Требование #7
			}

			// Сохраняем подтверждённую роль в контекст
			ctx := context.WithValue(r.Context(), PrivilegeKey{}, actualRole)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetConfirmedPrivilege получает подтверждённую роль из контекста
func GetConfirmedPrivilege(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(PrivilegeKey{}).(string)
	return role, ok
}
