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
func RequirePrivilege(db *sql.DB, requiredRole string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok {
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}

			var actualRole string
			err := db.QueryRowContext(r.Context(),
				"SELECT role FROM users WHERE id = $1",
				userID,
			).Scan(&actualRole)

			if err == sql.ErrNoRows {
				log.Warn("User not found during privilege check", zap.String("user_id", userID))
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}
			if err != nil {
				log.Error("Database error during privilege check", zap.Error(err))
				http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
				return
			}

			if actualRole != requiredRole {
				log.Warn("Insufficient privileges",
					zap.String("user_id", userID),
					zap.String("required_role", requiredRole),
					zap.String("actual_role", actualRole),
				)
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}

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
