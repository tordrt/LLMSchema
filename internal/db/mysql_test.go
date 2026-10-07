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

func TestCleanMySQLExpression(t *testing.T) {
	tests := []struct {
		expression string
		want       string
	}{
		{"(`quantity` * 2)", "(`quantity` * 2)"},
		{`concat(` + "`email`" + `,_latin1\'!\')`, "concat(`email`,'!')"},
		{`substring_index(` + "`email`" + `,_utf8mb4\'@\',-(1))`, "substring_index(`email`,'@',-(1))"},
		{`concat(` + "`a`" + `,_latin1\'it\\\'s\',_latin1\'back\\\\slash\')`, "concat(`a`,'it\\'s','back\\\\slash')"},
		{"convert(`a` using utf8mb4)", "convert(`a` using utf8mb4)"},
	}
	for _, tt := range tests {
		if got := cleanMySQLExpression(tt.expression); got != tt.want {
			t.Errorf("cleanMySQLExpression(%q) = %q, want %q", tt.expression, got, tt.want)
		}
	}
}
