package db

import (
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestMySQLDSN(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"mysql://user:pass@tcp(localhost:3306)/db", "user:pass@tcp(localhost:3306)/db"},
		{"mysql://user:pass@tcp(localhost:3306)/db?parseTime=true", "user:pass@tcp(localhost:3306)/db?parseTime=true"},
		{"mysql://user:pass@localhost:3306/db", "user:pass@tcp(localhost:3306)/db"},
		{"mysql://user:pass@localhost/db", "user:pass@tcp(localhost)/db"},
		{"mysql://user:pass@localhost:3306/db?parseTime=true", "user:pass@tcp(localhost:3306)/db?parseTime=true"},
		{"mysql://user:p%40ss%2Fword@db.internal:3307/app", "user:p@ss/word@tcp(db.internal:3307)/app"},
		{"mysql://user@[::1]:3306/db", "user@tcp([::1]:3306)/db"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := MySQLDSN(tt.url); got != tt.want {
				t.Errorf("MySQLDSN(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestMySQLDSNPreservesSpecialCharactersInPassword(t *testing.T) {
	cfg, err := mysql.ParseDSN(MySQLDSN("mysql://user:p%40ss%2Fword@db.internal/app"))
	if err != nil {
		t.Fatalf("ParseDSN() failed: %v", err)
	}
	if cfg.User != "user" || cfg.Passwd != "p@ss/word" || cfg.Addr != "db.internal:3306" || cfg.DBName != "app" {
		t.Fatalf("ParseDSN() = user %q, password %q, addr %q, db %q", cfg.User, cfg.Passwd, cfg.Addr, cfg.DBName)
	}
}
