# Property API

A full-featured RESTful API for a real estate property management platform built with Go, PostgreSQL, and JWT authentication. Supports multiple user roles, property management, inquiries, reviews, favorites, scheduling, and payments.

## Features

- Multi-role authentication (User, Agent, Admin)
- Property CRUD operations with media uploads
- Advanced property search and filtering
- Reviews system with moderation
- Inquiry and viewing schedule management
- Favorite properties and statistics
- Featured listings with payments
- Agent dashboard and analytics
- Admin dashboard and platform statistics
- Rate limiting, CORS support, and TLS support

## Tech Stack

- Go 1.x
- PostgreSQL
- JWT Authentication
- httprouter
- SMTP for emails
- Token bucket rate limiting
- TLS support for HTTPS

## Setup & Configuration

### Environment Variables
```bash
PORT=4000
ENV=development
DB_DSN=postgres://user:pass@localhost/propertydb?sslmode=disable
JWT_SECRET=your-super-secret-jwt-key-change-this
LIMITER_RPS=2
LIMITER_BURST=4
SMTP_HOST=sandbox.smtp.mailtrap.io
SMTP_PORT=2525
SMTP_USERNAME=your-username
SMTP_PASSWORD=your-password
SMTP_SENDER=Property API <noreply@propertyapi.com>
CORS_TRUSTED_ORIGINS=http://localhost:3000 http://localhost:4000
TLS_ENABLED=true
TLS_CERT_FILE=path/to/cert.pem
TLS_KEY_FILE=path/to/key.pem
