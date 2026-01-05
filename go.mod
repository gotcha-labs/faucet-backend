module faucet-backend

go 1.21

require (
	github.com/ethereum/go-ethereum v1.13.8
	github.com/gofiber/fiber/v2 v2.52.0
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.4.0
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
)
```

---

### `.gitignore`
```
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
faucet-backend

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out

# Environment variables
.env
.env.local

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Railway
.railway/
