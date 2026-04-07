# Livotech Warehouse Management System - Backend API

![Go](https://img.shields.io/badge/Go-1.25.5-00ADD8?logo=go)
![Fiber](https://img.shields.io/badge/Fiber-3.x-E7ECEF?logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14+-316192?logo=postgresql)
![Status](https://img.shields.io/badge/Status-Active-success)

A production-ready Backend REST API service for the Livotech Warehouse Management System. Built with modern Go technologies, this application provides a comprehensive solution for managing warehouse operations with role-based access control, real-time tracking, and integrated face recognition capabilities.

## 📋 Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Application](#running-the-application)
- [Project Structure](#project-structure)
- [API Documentation](#api-documentation)
- [Key Features](#key-features)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## ✨ Features

### Core Functionality

- **User Management** - Role-based user administration with RBAC (Role-Based Access Control)
- **Store Management** - Multi-store inventory system with centralized control
- **Channel Management** - Multiple sales channel integration and management
- **Product Management** - Complete product catalog with SKU tracking
- **Expedition Management** - Logistics and shipment planning
- **Order Processing** - Full order lifecycle management (creation, fulfillment, tracking)
- **Return Management** - Streamlined product return processing
- **Complaint Handling** - Customer complaint tracking and resolution
- **QC Management** - Quality control workflows (online & ribbon-based)
- **Picking & Packing** - Optimized order fulfillment operations
- **Location Management** - Warehouse location and zone management
- **Box Management** - Packaging and shipping container tracking
- **Lost & Found** - Management of misplaced items
- **Comprehensive Reporting** - Advanced analytics and business intelligence

### Advanced Features

- **Face Recognition Integration** - DeepFace integration for attendance verification
- **Real-time Attendance Tracking** - Mobile-enabled attendance system
- **Mobile APIs** - Dedicated endpoints for mobile applications
- **PASETO Token Security** - Secure token-based authentication
- **RBAC Middleware** - Fine-grained permission control
- **CSRF Protection** - Security middleware for CSRF prevention
- **Swagger Documentation** - Auto-generated API documentation

## 🛠️ Tech Stack

- **Language**: Go 1.25.5
- **Framework**: Fiber 3.x (Express.js-like web framework for Go)
- **ORM**: GORM 1.31.x
- **Database**: PostgreSQL 14+
- **Authentication**: PASETO v2 (Plaintext Agnostic Signed Encryption Tokens)
- **UUID**: Google UUID library
- **Security**: Bcrypt for password hashing, CORS & CSRF middleware
- **External Services**: DeepFace (Face Recognition)
- **Documentation**: Swagger/OpenAPI

## 📦 Prerequisites

Before you begin, ensure you have the following installed:

- **Go** - Version 1.25.5 or higher ([Download](https://golang.org/dl/))
- **PostgreSQL** - Version 14 or higher ([Download](https://www.postgresql.org/download/))
- **Git** - Version 2.0 or higher
- **Make** (optional) - For running common commands
- **Postman** or **Thunder Client** - For API testing (optional)

### Optional but Recommended

- **Docker** - For containerized PostgreSQL setup
- **DeepFace Service** - For face recognition features

## ⚙️ Installation

### 1. Clone the Repository

```bash
git clone https://github.com/ekamauln/livo-fiber-backend.git
cd livo-fiber-backend
```

### 2. Install Go Dependencies

```bash
go mod tidy
```

This command downloads all required dependencies specified in `go.mod`.

### 3. Generate PASETO Key

Generate a secure 32-byte symmetric key for token management:

```bash
go run ./cmd/generate_key.go
```

Copy the generated key for use in environment configuration.

### 4. Database Setup

#### Using PostgreSQL Directly

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE livo_fiber_db;

# Create user (if needed)
CREATE USER livo_user WITH PASSWORD 'your_secure_password';
ALTER ROLE livo_user CREATEDB;

# Grant privileges
GRANT ALL PRIVILEGES ON DATABASE livo_fiber_db TO livo_user;
```

#### Using Docker (Recommended)

```bash
docker run --name livo-postgres \
  -e POSTGRES_USER=livo_user \
  -e POSTGRES_PASSWORD=your_secure_password \
  -e POSTGRES_DB=livo_fiber_db \
  -p 5432:5432 \
  -d postgres:14-alpine
```

## 🔐 Configuration

Create a `.env` file in the root directory with the following variables:

### Database Configuration

```env
# PostgreSQL Connection
DB_HOST=localhost
DB_PORT=5432
DB_USER=livo_user
DB_PASSWORD=your_secure_password
DB_NAME=livo_fiber_db
DB_SSLMODE=disable
DB_TZ=Asia/Jakarta
```

### Application Configuration

```env
# Server Settings
ENV=development           # development or production
PORT=8000
APP_URL=http://localhost:8000
APP_NAME=Livotech Warehouse Management System API
LOG_LEVEL=info           # debug, info, warn, error
```

### Token Security

```env
# PASETO Token Configuration (generate using cmd/generate_key.go)
PASETO_SYMMETRIC_KEY=your_32_byte_symmetric_key_here
ACCESS_TOKEN_TTL=60      # in minutes (default: 60)
REFRESH_TOKEN_TTL=7      # in days (default: 7)
```

### CORS Configuration

```env
# Comma-separated list of allowed origins
CORS_ORIGINS=http://localhost:3000,http://localhost:3001,https://yourdomain.com
```

### External Services

```env
# DeepFace Service (for face recognition)
DEEPFACE_URL=http://localhost:5000
```

### Example `.env` File

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=livo_user
DB_PASSWORD=secure_password_here
DB_NAME=livo_fiber_db
DB_SSLMODE=disable
DB_TZ=Asia/Jakarta

# Server
ENV=development
PORT=8000
APP_URL=http://localhost:8000
APP_NAME=Livotech Warehouse Management System API
LOG_LEVEL=info

# Security & Tokens
PASETO_SYMMETRIC_KEY=your_32_byte_key_generated_by_generate_key.go
ACCESS_TOKEN_TTL=60
REFRESH_TOKEN_TTL=7

# CORS
CORS_ORIGINS=http://localhost:3000,http://localhost:3001

# External Services
DEEPFACE_URL=http://localhost:5000
```

## 🚀 Running the Application

### Development Mode

```bash
# Run directly
go run main.go

# With automatic reload (requires air: go install github.com/cosmtrek/air@latest)
air
```

The API will be available at `http://localhost:8000` (or your configured PORT)

### Production Build

```bash
# Build executable
go build -o livotech-app main.go

# Run the application
./livotech-app

# Or on Windows
livotech-app.exe
```

### Docker Deployment (Optional)

```bash
# Build Docker image
docker build -t livo-fiber-backend:latest .

# Run container
docker run -p 8000:8000 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=livo_user \
  -e DB_PASSWORD=your_password \
  -e DB_NAME=livo_fiber_db \
  -e PASETO_SYMMETRIC_KEY=your_key \
  livo-fiber-backend:latest
```

## 📁 Project Structure

```
livo-fiber-backend/
├── cmd/                              # Command-line utilities
│   └── generate_key.go              # PASETO key generator
├── config/                           # Configuration management
│   └── config.go                    # Config struct & loader
├── controllers/                      # HTTP request handlers
│   ├── auth_controller.go           # Authentication endpoints
│   ├── user_controller.go           # User management
│   ├── store_controller.go          # Store operations
│   ├── order_controller.go          # Order processing
│   ├── product_controller.go        # Product management
│   ├── returned_order_controller.go # Return management
│   ├── expedition_controller.go     # Logistics management
│   ├── qc_*.go                      # Quality control
│   ├── mobile_*.go                  # Mobile-specific endpoints
│   └── ...                          # Additional controllers
├── middleware/                       # HTTP middleware
│   ├── auth.go                      # Authentication middleware
│   ├── rbac.go                      # Role-based access control
│   └── csrf.go                      # CSRF protection
├── models/                           # Data models (GORM structs)
│   ├── user.go                      # User model
│   ├── order.go                     # Order model
│   ├── product.go                   # Product model
│   ├── store.go                     # Store model
│   └── ...                          # Additional models
├── database/                         # Database connection & setup
│   └── database.go                  # DB initialization
├── routes/                           # Route definitions
│   └── routes.go                    # All API routes
├── utils/                            # Utility functions
│   ├── token.go                     # PASETO token handling
│   ├── password.go                  # Password hashing
│   ├── permission.go                # Permission checking
│   ├── deepface_client.go           # Face recognition integration
│   ├── response.go                  # JSON response formatting
│   ├── gps.go                       # GPS utilities
│   └── ...                          # Additional utilities
├── docs/                             # API documentation
│   ├── swagger.json                 # OpenAPI spec (JSON)
│   ├── swagger.yaml                 # OpenAPI spec (YAML)
│   └── status-list.txt              # Status definitions
├── database/                         # Database utilities
├── main.go                           # Application entry point
├── go.mod                            # Go module definition
├── go.sum                            # Dependencies checksums
├── .env                              # Environment variables (not in repo)
└── env.example                       # Example env file
```

## 📖 API Documentation

### Swagger Documentation

Once the application is running, access the interactive API documentation:

**Swagger UI**: `http://localhost:8000/swagger/index.html`

The Swagger documentation includes:

- All available endpoints
- Request/response models
- Authentication requirements
- Example requests and responses

### API Endpoints Overview

#### Authentication

- `POST /auth/login` - User login
- `POST /auth/refresh` - Refresh access token
- `POST /auth/logout` - User logout

#### User Management

- `GET /users` - List all users
- `GET /users/:id` - Get user details
- `POST /users` - Create new user
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Delete user

#### Store Management

- `GET /stores` - List stores
- `GET /stores/:id` - Get store details
- `POST /stores` - Create store
- `PUT /stores/:id` - Update store
- `DELETE /stores/:id` - Delete store

#### Order Management

- `GET /orders` - List orders
- `GET /orders/:id` - Get order details
- `POST /orders` - Create order
- `PUT /orders/:id` - Update order
- `DELETE /orders/:id` - Cancel order
- `POST /orders/:id/pack` - Pack order
- `POST /orders/:id/ship` - Ship order

#### Product Management

- `GET /products` - List products
- `GET /products/:id` - Get product details
- `POST /products` - Create product
- `PUT /products/:id` - Update product
- `DELETE /products/:id` - Delete product

#### Additional Endpoints

Refer to the Swagger documentation for comprehensive endpoint listings.

## 🔑 Key Features Details

### Role-Based Access Control (RBAC)

The system implements fine-grained RBAC through:

- Role definitions in the database
- Middleware-enforced permission checks
- Resource-level access control

### Authentication & Security

- **PASETO Tokens**: Secure token-based authentication
- **Password Hashing**: Bcrypt with configurable cost
- **CSRF Protection**: Middleware to prevent CSRF attacks
- **CORS Support**: Configurable cross-origin requests

### Face Recognition

Integrated DeepFace service for:

- Employee attendance verification
- Security and identity validation

### Mobile Support

Dedicated mobile API endpoints at `/mobile/*` for:

- Optimized mobile operations
- Attendance tracking
- Order and return management

## 👨‍💻 Development

### Prerequisites for Development

```bash
# Install air for hot reload (optional)
go install github.com/cosmtrek/air@latest

# Install swag for Swagger generation (optional)
go install github.com/swaggo/swag/cmd/swag@latest
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./controllers -v
```

### Code Generation

#### Generate New PASETO Key

```bash
go run ./cmd/generate_key.go
```

#### Update Swagger Documentation

```bash
swag init
swag fmt
```

### Adding New Features

1. **Create Model** in `models/your_feature.go`
2. **Create Controller** in `controllers/your_feature_controller.go`
3. **Add Routes** to `routes/routes.go`
4. **Add Middleware** if needed to `middleware/`
5. **Update Tests** and documentation

### Code Style Guidelines

- Follow Go code conventions ([Effective Go](https://golang.org/doc/effective_go))
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and testable
- Use error handling best practices

## 🐛 Troubleshooting

### Database Connection Issues

**Error**: `could not connect to database`

```bash
# Verify PostgreSQL is running
# Check credentials in .env file
# Ensure database exists
psql -U livo_user -d livo_fiber_db -c "SELECT 1;"
```

### PASETO Key Issues

**Error**: `invalid paseto key`

```bash
# Regenerate key
go run ./cmd/generate_key.go

# Ensure key is correctly set in .env (32 bytes)
```

### Port Already in Use

```bash
# Change PORT in .env or use:
PORT=8001 go run main.go

# Linux/Mac: Find and kill process on port 8000
lsof -ti:8000 | xargs kill -9

# Windows: Find and kill process on port 8000
netstat -ano | findstr :8000
taskkill /PID <PID> /F
```

### CORS Errors

Ensure your frontend URL is in `CORS_ORIGINS` environment variable:

```env
CORS_ORIGINS=http://localhost:3000,http://your-frontend-domain.com
```

### DeepFace Connection Issues

Verify DeepFace service is running:

```bash
curl http://localhost:5000/health
```

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is proprietary software. All rights reserved © 2025 Livotech.

## 📧 Support & Contact

For issues, questions, or support:

- Create an issue on the GitHub repository
- Contact the development team through official channels

---

**Happy Coding! 🚀**

https://discord.gg/xzHzq2cY
