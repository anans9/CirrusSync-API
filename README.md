# CirrusSync API

A secure, end-to-end encrypted cloud storage API built with Go, featuring advanced authentication, file sharing, and multi-tenant storage management.

## 🚀 Features

### Core Functionality
- **Secure File Storage**: End-to-end encrypted file storage with AWS S3 backend
- **Drive Volumes**: Multi-tenant storage volumes with configurable size limits
- **File Sharing**: Advanced sharing capabilities with permission management
- **File Versioning**: Complete revision history and rollback capabilities
- **Thumbnail Generation**: Automatic thumbnail generation for media files

### Authentication & Security
- **SRP Authentication**: Secure Remote Password protocol implementation
- **Multi-Factor Authentication**: TOTP, Email, and SMS-based 2FA
- **JWT Tokens**: RSA-signed access and refresh tokens
- **CSRF Protection**: Built-in Cross-Site Request Forgery protection
- **Security Events**: Comprehensive audit logging and monitoring
- **Device Management**: Track and manage user devices

### User Management
- **User Profiles**: Complete user account management
- **Session Management**: Secure session handling with Redis
- **Recovery Kits**: Account recovery mechanisms
- **Billing Integration**: Stripe integration for subscriptions
- **Notification System**: Email and SMS notifications

### Infrastructure
- **Graceful Shutdown**: Proper cleanup and shutdown procedures
- **Health Monitoring**: Sentry integration for error tracking
- **Redis Caching**: High-performance caching layer
- **Database Migrations**: Automated schema management
- **CORS Support**: Cross-origin resource sharing configuration

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │────│   CirrusSync    │────│   PostgreSQL    │
│   Applications  │    │   API Server    │    │   Database      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ├─────────────────┐
                              │                 │
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Redis Cache   │    │   AWS S3        │
                       │   & Sessions    │    │   File Storage  │
                       └─────────────────┘    └─────────────────┘
```

## 🛠️ Technology Stack

- **Language**: Go 1.24.2
- **Web Framework**: Gin
- **Database**: PostgreSQL with GORM ORM
- **Cache**: Redis
- **File Storage**: AWS S3
- **Authentication**: JWT + SRP
- **Monitoring**: Sentry
- **Development**: Air (hot reload)

## 📦 Installation

### Prerequisites

- Go 1.24.2 or higher
- PostgreSQL 12+
- Redis 6+
- AWS S3 bucket (or S3-compatible storage)

### Clone Repository

```bash
git clone https://github.com/anans9/cirrussync-api.git
cd cirrussync-api
```

### Install Dependencies

```bash
go mod download
```

### Generate RSA Keys

```bash
mkdir -p keys
# Generate private key
openssl genrsa -out keys/private.pem 2048
# Generate public key
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

## ⚙️ Configuration

Create environment configuration files:

### `.env` (Base Configuration)

```env
# Server Configuration
PORT=8000
HOST=localhost
ENVIRONMENT=development
REQUEST_TIMEOUT=30
SHUTDOWN_TIMEOUT=10

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=cirrussync
DB_USER=postgres
DB_PASSWORD=your_password
DB_SSL_MODE=disable
DB_TIMEZONE=UTC
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=300
MIGRATE_ON_BOOT=true

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10

# AWS S3 Configuration
S3_REGION=us-east-1
S3_BUCKET_NAME=your-bucket-name
S3_ACCESS_KEY_ID=your-access-key
S3_SECRET_ACCESS_KEY=your-secret-key
S3_ENDPOINT=  # Optional: for S3-compatible services

# CSRF Protection
CSRF_SECRET=your-32-char-secret-key-here
CSRF_SECURE=false

# Mail Configuration (SMTP)
MAIL_SMTP_HOST=smtp.gmail.com
MAIL_SMTP_PORT=587
MAIL_SMTP_USERNAME=your-email@gmail.com
MAIL_SMTP_PASSWORD=your-app-password
MAIL_FROM_ADDRESS=noreply@cirrussync.com
MAIL_FROM_NAME=CirrusSync

# TOTP Configuration
TOTP_ISSUER=CirrusSync
TOTP_ACCOUNT_NAME=CirrusSync Account

# Monitoring (Optional)
SENTRY_DSN=your-sentry-dsn
APP_VERSION=1.0.0
```

### Database Setup

```bash
# Create database
createdb cirrussync

# The application will automatically run migrations on startup
# when MIGRATE_ON_BOOT=true
```

## 🚀 Running the Application

### Development Mode (with hot reload)

```bash
# Install Air for hot reloading
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### Production Mode

```bash
# Build the application
go build -o cirrussync-api cmd/main.go

# Run the application
./cirrussync-api
```

### Using Docker (Optional)

```dockerfile
# Dockerfile example
FROM golang:1.24.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o cirrussync-api cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/cirrussync-api .
COPY --from=builder /app/keys ./keys

CMD ["./cirrussync-api"]
```

## 📚 API Documentation

### Base URL
```
http://localhost:8000/api/v1
```

### Authentication

The API uses JWT tokens for authentication. Most endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### Core Endpoints

#### Authentication
- `POST /auth/register` - User registration
- `POST /auth/login/challenge` - SRP login challenge
- `POST /auth/login/verify` - SRP login verification
- `POST /auth/refresh` - Refresh JWT token
- `POST /auth/logout` - User logout
- `GET /auth/me` - Get current user info

#### Users
- `GET /users/profile` - Get user profile
- `PUT /users/profile` - Update user profile
- `DELETE /users/account` - Delete user account

#### Sessions
- `GET /sessions` - List user sessions
- `DELETE /sessions/:id` - Revoke specific session
- `DELETE /sessions/all` - Revoke all sessions

#### Multi-Factor Authentication
- `GET /mfa/methods` - List MFA methods
- `POST /mfa/totp/setup` - Setup TOTP
- `POST /mfa/totp/verify` - Verify TOTP
- `POST /mfa/email/send` - Send email code
- `POST /mfa/sms/send` - Send SMS code

#### Drive & Files
- `GET /drive/volumes` - List drive volumes
- `POST /drive/volumes` - Create new volume
- `GET /drive/volumes/:id/items` - List items in volume
- `POST /drive/upload` - Upload file
- `GET /drive/download/:id` - Download file
- `POST /drive/share` - Share file/folder
- `GET /drive/shared` - List shared items

#### Utility
- `GET /csrf/token` - Get CSRF token

## 🔒 Security Features

### SRP Authentication
The API implements the Secure Remote Password (SRP) protocol for zero-knowledge password authentication:

1. Client requests login challenge
2. Server responds with salt and challenge
3. Client computes proof using password
4. Server verifies proof without knowing password

### Encryption
- All file data is encrypted at rest
- Transport layer security with HTTPS
- JWT tokens signed with RSA keys
- Password hashing with bcrypt

### Security Headers
- CSRF protection enabled
- CORS properly configured
- Security headers set automatically

## 🧪 Development

### Project Structure

```
├── api/v1/              # API handlers
│   ├── auth/           # Authentication endpoints
│   ├── drive/          # File storage endpoints
│   ├── mfa/            # Multi-factor auth endpoints
│   ├── sessions/       # Session management
│   └── users/          # User management
├── cmd/                # Application entry point
├── internal/           # Private application code
│   ├── auth/          # Authentication service
│   ├── drive/         # Drive service
│   ├── jwt/           # JWT service
│   ├── middleware/    # HTTP middleware
│   ├── models/        # Database models
│   └── user/          # User service
├── pkg/               # Public packages
│   ├── config/        # Configuration
│   ├── db/            # Database connection
│   ├── redis/         # Redis connection
│   └── s3/            # S3 client
└── router/            # HTTP router setup
```

## 🐛 Debugging

### Enable Debug Logging

Set environment variable:
```bash
export GIN_MODE=debug
```

### Health Checks

The application provides several health check endpoints:

- Database connectivity
- Redis connectivity
- S3 connectivity

### Monitoring

Integration with Sentry for error monitoring:
- Automatic error collection
- Performance monitoring
- Custom event tracking

## 🚀 Deployment

### Environment Variables

Ensure all required environment variables are set in production:

```bash
# Security
CSRF_SECURE=true
ENVIRONMENT=production

# SSL/TLS
DB_SSL_MODE=require

# Monitoring
SENTRY_DSN=your-production-sentry-dsn
```

### Systemd Service (Linux)

```ini
[Unit]
Description=CirrusSync API Server
After=network.target

[Service]
Type=simple
User=cirrussync
WorkingDirectory=/opt/cirrussync
ExecStart=/opt/cirrussync/cirrussync-api
Restart=always
RestartSec=5
Environment=ENVIRONMENT=production

[Install]
WantedBy=multi-user.target
```

### Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name api.cirrussync.com;

    location / {
        proxy_pass http://localhost:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Contribution Guidelines

- Follow Go conventions and best practices
- Write tests for new features
- Update documentation as needed
- Ensure all tests pass before submitting PR
- Use conventional commit messages

## 🎯 Roadmap

- [ ] Billing support
- [ ] GraphQL API support
- [ ] API rate limiting

---

**CirrusSync API** - Secure, scalable cloud storage for the modern web.
