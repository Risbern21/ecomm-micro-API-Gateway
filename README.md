# 🚀 API Gateway - Microservices E-Commerce

A robust and scalable Gateway service designed to manage, secure, and route traffic across multiple microservices in an e-commerce ecosystem. This service acts as the central entry point for all client requests, handling authentication, session management, rate limiting, and intelligent load balancing via service discovery.

## 🌟 Overview

The API Gateway is a high-performance Go-based service that streamlines communication between external clients and internal microservices. It integrates seamlessly with Eureka for service discovery and uses Redis for high-speed caching and rate limiting, ensuring the platform remains responsive and secure under load.

## 🛠️ Technology Stack

- **Languge:** Go (1.21+)
- **Router:** [Gorilla Mux](https://github.com/gorilla/mux)
- **Database:** PostgreSQL (via [GORM](https://gorm.io/))
- **Caching & Rate Limiting:** [Redis](https://redis.io/)
- **Service Discovery:** [Netflix Eureka](https://github.com/Netflix/eureka) (via [Fargo](https://github.com/hudl/fargo))
- **Security:** JWT (JSON Web Tokens) with Session Management
- **Infrastructure:** Docker

## ✨ Key Features

- **Centralized Authentication:** OAuth2-style flows with Access and Refresh tokens.
- **Microservices Proxying:** Dynamic routing to Products, Orders, and Chat services.
- **Service Discovery:** Automatic registration and heartbeat monitoring with Eureka.
- **Session Management:** Robust session handling with revocation and renewal capabilities.
- **Performance:** Integrated caching layer and rate limiting to protect downstream services.
- **Observability:** Structured logging and centralized error handling for all proxied requests.

---

## 📡 API Routes

The Gateway provides a unified API surface under the `/api` prefix.

### 🔐 Authentication & Identity
| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/auth/signin` | `POST` | Register a new user account with profile details. |
| `/api/auth/login` | `POST` | Authenticate credentials and receive a JWT Access & Refresh token. |
| `/api/auth/logout` | `POST` | Terminate the current session (requires `id` query parameter). |
| `/api/tokens/renew` | `POST` | Use a valid Refresh Token to generate a new Access Token. |
| `/api/tokens/revoke` | `POST` | Revoke a specific session by its ID (requires `id` query parameter). |

### 🛠️ System Health
| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/health` | `GET` | Health check endpoint to verify the Gateway's operational status. |

### 📦 Microservices Proxy (Authenticated)
All requests to the following paths are automatically proxied to their respective backend services. These routes require a valid JWT token.

#### Products Service
- **Endpoint:** `/api/products/*`
- **Proxy Target:** `PRODUCTS-SERVICE`
- **Supported Methods:** `GET`, `POST`, `PUT`, `DELETE`
- **Features:** Rate limiting, Caching, Authentication

#### Orders Service
- **Endpoint:** `/api/orders/*`
- **Proxy Target:** `ORDERS-SERVICE`
- **Supported Methods:** `GET`, `POST`, `PUT`, `DELETE`
- **Features:** Rate limiting, Caching, Authentication

#### Chat & AI Client Service
- **Endpoint:** `/api/chat/*`
- **Proxy Target:** `CHAT-CLIENT-SERVICE`
- **Supported Methods:** `GET`, `POST`, `PUT`, `DELETE`
- **Features:** Rate limiting, Caching, Authentication

#### AI Generation Service
- **Endpoint:** `/api/generate/*`
- **Proxy Target:** `CHAT-CLIENT-SERVICE` (AI Implementation)
- **Supported Methods:** `GET`, `POST`, `PUT`, `DELETE`
- **Features:** Rate limiting, Caching, Authentication

---

## 🚀 Getting Started

### Prerequisites
- Go installed on your local machine.
- Docker and Docker Compose (recommended for local infrastructure).
- A running Eureka Server for service discovery.

### Running the Gateway
## 1.Clone from source
1. Ensure your database and Redis instances are reachable.
2. Initialize the service:
   ```bash
   go mod download
   ```
3. Run the application:
   ```bash
   go run main.go
   ```
---
## 2. Pull from DockerHub
   ```bash
   docker pull risbernfernandes/micro-ecomm-api-gateway:latest
   ```
---

## 🏗️ Project Structure

```bash
.
├── handler/        # HTTP Handlers & Request/Response Logic
├── internal/       # Core Logic (Database, Cache, Token Maker, Middleware)
├── model/          # GORM Models & Database Operations
├── routes/         # Router Definitions & Proxy Logic
├── util/           # Helper functions (Hashing, etc.)
└── main.go         # Application Entry Point
```

---
*Designed with ❤️ for High-Scale E-Commerce.*
