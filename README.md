# Comment Service

A comprehensive comment microservice built with Go and MongoDB, featuring multi-tenant support, moderation workflow, reactions, and real-time notifications.

## Features

### Core Features
- **Comments & Replies**: Nested comments with configurable depth limit
- **Reactions**: Like, dislike, love, haha, wow, sad, angry
- **CRUD Operations**: Create, read, update, soft delete comments
- **Multi-tenant Support**: Isolate comments by tenant (shop, ticket system, blog, etc.)
- **Resource-based**: Comments attached to any resource type/ID

### Moderation
- **Approval Workflow**: Comments can require approval before being visible
- **Bad Words Filter**: Configurable list of blocked words
- **Pin Comments**: Highlight important comments
- **Rejection Reasons**: Track why comments were rejected
- **Bulk Moderation**: Approve/reject multiple comments at once

### Additional Features
- **Anonymous Comments**: Optional anonymous posting
- **Edit History**: Track all edits to comments
- **Search**: Full-text search across comments
- **Statistics**: Get comment counts and metrics
- **Rate Limiting**: Prevent spam with configurable limits
- **Notifications**: Integration with notifier service

## Architecture

```
comment/
├── cmd/
│   └── main.go              # Application entry point
├── config/
│   └── config.go            # Configuration management
├── internal/
│   ├── client/
│   │   └── notifier_client.go  # Notifier service client
│   ├── database/
│   │   └── mongodb.go       # MongoDB connection & indexes
│   ├── handler/
│   │   ├── admin_handler.go     # Admin endpoints
│   │   ├── comment_handler.go   # Comment CRUD endpoints
│   │   ├── health_handler.go    # Health check endpoints
│   │   └── reaction_handler.go  # Reaction endpoints
│   ├── middleware/
│   │   ├── auth.go          # Authentication middleware
│   │   ├── logging.go       # Request logging
│   │   ├── rate_limit.go    # Rate limiting
│   │   └── tenant.go        # Tenant extraction
│   ├── models/
│   │   ├── comment.go       # Domain models
│   │   └── dto.go           # Request/Response DTOs
│   ├── repository/
│   │   ├── comment_repository.go   # Comment data access
│   │   ├── reaction_repository.go  # Reaction data access
│   │   ├── report_repository.go    # Report data access
│   │   └── settings_repository.go  # Settings data access
│   ├── router/
│   │   └── router.go        # HTTP route setup
│   └── usecase/
│       ├── comment_usecase.go   # Comment business logic
│       └── reaction_usecase.go  # Reaction business logic
├── Dockerfile
├── docker-compose.yml
├── docker-compose.dev.yml
├── Makefile
└── go.mod
```

## API Endpoints

### Comments
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/comments` | Create a comment |
| GET | `/api/v1/comments` | List comments |
| GET | `/api/v1/comments/:id` | Get a comment |
| PUT | `/api/v1/comments/:id` | Update a comment |
| DELETE | `/api/v1/comments/:id` | Delete a comment |
| GET | `/api/v1/comments/:id/replies` | Get replies |
| GET | `/api/v1/comments/search` | Search comments |
| GET | `/api/v1/comments/stats` | Get statistics |

### Reactions
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/comments/:id/reactions` | Add/update reaction |
| DELETE | `/api/v1/comments/:id/reactions` | Remove reaction |
| GET | `/api/v1/comments/:id/reactions/me` | Get user's reaction |

### Admin
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/comments/pending` | Get pending comments |
| POST | `/api/v1/admin/comments/:id/moderate` | Approve/reject comment |
| POST | `/api/v1/admin/comments/:id/pin` | Pin/unpin comment |
| DELETE | `/api/v1/admin/comments/:id` | Hard delete comment |
| POST | `/api/v1/admin/comments/bulk-moderate` | Bulk moderation |

### Health
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Full health check |
| GET | `/ready` | Readiness probe |
| GET | `/live` | Liveness probe |

## Configuration

Copy `.env.example` to `.env` and configure:

```env
# Server
SERVER_PORT=5010

# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=minisource_comments

# Auth Service
AUTH_SERVICE_URL=http://localhost:5001
AUTH_CLIENT_ID=comment-service
AUTH_CLIENT_SECRET=comment-service-secret-key

# Moderation
MODERATION_REQUIRE_APPROVAL=true
MODERATION_BAD_WORDS_ENABLED=true
MODERATION_MAX_COMMENT_LENGTH=5000
MODERATION_MAX_REPLY_DEPTH=5
MODERATION_RATE_LIMIT_PER_MINUTE=10
```

## Development

### Prerequisites
- Go 1.24+
- MongoDB 7+
- Docker (optional)

### Running Locally

```bash
# Install dependencies
go mod download

# Run the service
make run
```

### Docker

```bash
# Development
docker-compose -f docker-compose.dev.yml up -d

# Production
docker-compose up -d
```

### Testing

```bash
make test
```

## Multi-tenant Usage

Set the tenant in request headers:

```
X-Tenant-ID: shop-tenant
```

Or as a query parameter:

```
GET /api/v1/comments?tenant_id=shop-tenant&resource_type=product&resource_id=123
```

## Example Requests

### Create Comment

```bash
curl -X POST http://localhost:5010/api/v1/comments \
  -H "Authorization: Bearer <token>" \
  -H "X-Tenant-ID: shop" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "product",
    "resource_id": "product-123",
    "content": "Great product!",
    "rating": 5
  }'
```

### Add Reaction

```bash
curl -X POST http://localhost:5010/api/v1/comments/abc123/reactions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "like"
  }'
```

### Moderate Comment

```bash
curl -X POST http://localhost:5010/api/v1/admin/comments/abc123/moderate \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "approved"
  }'
```

## License

MIT
