# diploma project - Oiyn-Shak
## Go Microservices Monorepo

This monorepo contains a set of Go-based microservices for a backend system. It includes core services like:

- **SSO Service** – authentication, authorization, JWT-based access and refresh tokens.
- **Profile Service** – user profile management.
- **Mailer Integration** – email verification and password reset using Mailtrap.
- **PostgreSQL** – primary storage.
- **gRPC + grpc-gateway** – for high-performance service communication with REST compatibility.

---

## 📁 Structure
```
/cmd
└── service_name/ # Entry points for microservices (migrator as well)
/internal
├── app/ # Application wire-up logic
├── services/ # Service logic
├── config/ # Environment/config loading
├── http/ # gRPC-Gateway HTTP server wrapper
├── mailer/ # Mailtrap email sender
├── storage/ # DB interactions using pgx/pgxpool
└── ...
```


---

## 🚀 Quick Start

### 1. Clone and setup

```bash
git clone https://github.com/m4rk1sov/oiyn-shak.git
cd cloned-directory
```

## 2. Environment setup

### PostgreSQL

```shell
psql -U postgres
#password for postgres admin
create user auth with password 'password';
create database 'oiyn-shak-sso' owner auth;
create database 'oiyn-shak-profile' owner auth;
```

### Configuration of .env file

```dotenv
CONFIG_PATH="./config/local_config.yaml"

DSN_STRING_SSO="postgres://auth:password@localhost:5432/oiyn-shak-sso"
DSN_STRING_PROFILE="postgres://auth:password@localhost:5432/oiyn-shak-profile"
MIGRATE_STRING_SSO="auth:password@localhost:5432/oiyn-shak-sso"
MIGRATE_STRING_PROFILE="auth:password@localhost:5432/oiyn-shak-profile"
MIGRATE_PATH="./migrations/"
SSL_MODE="disable"
```

### Mailtrap API

```dotenv
MAILTRAP_API="YOUR_API_KEY"
```

## 3. Launch locally
```shell
go run ./cmd/migrator --cmd up

go run ./cmd/service-name
```