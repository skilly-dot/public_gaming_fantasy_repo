## **README.md - Backend**

```markdown
# FanBaze Backend
four months 1 week 3 days and 7 hours 2 minutes to my commit...a journey so hard...but here it is..

A Go-based backend server for the FanBaze fantasy gaming platform. Provides REST API endpoints, WebSocket connections for real-time updates, and comprehensive test coverage for league management, betting systems, and user operations.

## Overview

The backend is built with Go and uses PostgreSQL for primary data storage, Redis for session caching, and WebSocket for real-time client updates. It serves a fantasy gaming platform where users can create leagues, manage teams, place bets on matches, and participate in quick matches and 50-50 propositions.

## Technology Stack

- **Go**: Programming language
- **Chi Router**: HTTP routing framework
- **PostgreSQL**: Primary database
- **Redis**: Session caching
- **WebSocket**: Real-time updates
- **bcrypt**: Password hashing

## Project Structure

```
gaming-fantasy/
├── cmd/
│   └── api/
│       ├── main.go          # Application entry point
│       └── routes.go        # Route definitions and middleware setup
├── internal/
│   ├── config/              # Configuration loading
│   ├── database/            # Database connections and migrations
│   ├── engine/              # League generation and match simulation
│   ├── handlers/            # HTTP request handlers
│   ├── middleware/          # Authentication and security middleware
│   ├── models/              # Data models
│   └── websocket/           # WebSocket hub and client management
└── tests/                   # Test files
```

## Core Features

### League Management

- Create leagues with 5 to 20 teams
- Multiple difficulty levels (Beginner to Grandmaster)
- Automatic fixture generation using round-robin scheduling
- League table calculation with points, goal difference, and standings
- Week-by-week progression
- League forfeit with penalty
- League completion with prize distribution

### Match Simulation

- Team strength-based simulation using Poisson distribution
- Realistic score generation with maximum 5 goals per team
- Goal scorer selection based on player positions
- Match statistics generation (possession, shots, cards, corners)
- Player rating system with position-specific attributes

### Betting Systems

- **League Bets**: Multi-selection accumulator bets on league matches
- **Admin Match Bets**: Special bets created by administrators with custom odds
- **50-50 Bets**: Binary yes/no propositions with configurable odds
- **Winner Bets**: Free bets predicting league champions with prize pools
- **Quick Matches**: Instant match generation with immediate settlement

### Wallet System

- Three currencies: USD, BetPoints, and Blings
- Currency exchange with configurable rates
- Daily bonus claims with streak tracking
- Transaction history
- Automatic wallet updates on bet placement and settlement

### Authentication

- User registration with username and password
- Session-based authentication using UUID tokens
- Session expiry after 7 days
- Account deletion with complete data cleanup
- Admin role separation with access control

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/register | Register new user |
| POST | /api/v1/auth/login | User login |
| POST | /api/v1/auth/logout | User logout |

### Leagues

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/leagues/create | Create new league |
| GET | /api/v1/leagues/my | Get user's leagues |
| GET | /api/v1/leagues/{id} | Get league details |
| GET | /api/v1/leagues/{id}/state | Get league state |
| GET | /api/v1/leagues/{id}/table | Get league table |
| GET | /api/v1/leagues/{id}/teams | Get league teams |
| GET | /api/v1/leagues/{id}/players | Get league players |
| POST | /api/v1/leagues/{id}/start-week | Start week simulation |
| POST | /api/v1/leagues/{id}/next-week | Advance to next week |
| POST | /api/v1/leagues/{id}/winner-bet | Place winner bet |
| POST | /api/v1/leagues/{id}/forfeit | Forfeit league |
| POST | /api/v1/leagues/{id}/finish | Finish completed league |

### Quick Match

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/leagues/{id}/quick-match/generate | Generate random matchup |
| POST | /api/v1/leagues/{id}/quick-match/{matchId}/bet | Place bet on quick match |
| POST | /api/v1/leagues/{id}/quick-match/{matchId}/start | Start match simulation |

### Betting

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/bets/place | Place league bet |
| GET | /api/v1/bets/active | Get active bets |
| GET | /api/v1/bets/history | Get bet history |
| GET | /api/v1/bets/admin-matches | Get admin match bets |
| POST | /api/v1/bets/admin-place | Place admin match bet |
| GET | /api/v1/bets/fifty-fifty-available | Get 50-50 bets |
| POST | /api/v1/bets/fifty-fifty-place | Place 50-50 bet |

### Wallet

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/wallet | Get wallet balance |
| POST | /api/v1/wallet/daily-bonus | Claim daily bonus |
| POST | /api/v1/wallet/exchange | Exchange currency |

### Admin

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/admin/users | Get all users |
| PUT | /api/v1/admin/users/{id} | Update user |
| DELETE | /api/v1/admin/users/{id} | Delete user |
| GET | /api/v1/admin/leagues | Get all leagues |
| PUT | /api/v1/admin/wallets/{id} | Update wallet |
| POST | /api/v1/admin/match-bets/create | Create match bet |
| POST | /api/v1/admin/matches/{id}/settle | Settle match |
| POST | /api/v1/admin/fifty-fifty/create | Create 50-50 bet |
| PUT | /api/v1/admin/fifty-fifty/{id}/settle | Settle 50-50 bet |

## Testing

The project includes unit tests, integration tests, and load tests covering engine logic, API handlers, and security features.

### Unit Tests (Engine)

Tests in `internal/engine/engine_test.go` verify:

- **League Generation**: Tests custom league creation with different team counts and difficulty levels. Verifies correct number of teams, fixtures, and players are generated.

- **Poisson Distribution**: Tests that goal generation using Poisson distribution produces realistic values within expected ranges across 1000 iterations.

- **Odds Calculation**: Verifies that odds calculation produces valid values and that stronger teams have lower odds (higher probability of winning).

- **Match Simulation**: Tests that simulated matches produce valid scores and that league table points are calculated correctly.

- **Player Generation**: Verifies that players are generated with correct positions, ratings, and statistics.

### Integration Tests (Handlers)

Tests in `internal/handlers/` verify:

- **User Lifecycle**: Tests registration, login, and account deletion flows.

- **League Operations**: Tests league creation, forfeit idempotency (forfeiting an already-deleted league returns success without double-charging).

- **Input Validation**: Tests that invalid inputs are rejected with appropriate status codes.

- **Currency Normalization**: Tests that currency name mappings (USD to Kash, BetPoints to Points, Blings to Coins) work correctly.

### Security Tests

Tests in `internal/handlers/security_test.go` verify:

- **Invalid Sessions**: Tests that empty tokens, fake tokens, SQL injection attempts, and XSS payloads are all rejected with 401 Unauthorized.

- **SQL Injection Prevention**: Tests that malicious usernames cannot bypass validation or cause database damage.

- **App Key Validation**: Tests that requests without valid app keys are blocked in production mode.

- **Session Expiry**: Tests that expired sessions are rejected.

- **Admin Access Control**: Tests that non-admin users cannot access admin endpoints.

- **Rate Limiting**: Tests that the rate limiter blocks requests after the specified limit.

### Load Tests

Tests simulate concurrent requests to verify the system handles multiple users simultaneously:

- **Concurrent Bet Placement**: Simulates 100 simultaneous bet placements.

- **Concurrent Wallet Fetch**: Simulates 50 simultaneous wallet retrievals.

## Running Tests

### Run all tests (requires database):

```bash
go test ./... -v
```

### Run unit tests only:

```bash
go test ./internal/engine/ -v
```

### Run integration tests only:

```bash
go test ./internal/handlers/ -v
```

### Run security tests only:

```bash
go test ./internal/handlers/ -v -run "TestInvalid|TestSQL|TestAppKey|TestSession|TestAdmin|TestRate"
```

### Run with race detection:

```bash
go test ./... -race
```

### Run with coverage:

```bash
go test ./... -cover
```

## Security Features

- **Password Hashing**: bcrypt with salt for secure password storage
- **SQL Injection Prevention**: Parameterized queries throughout
- **XSS Protection**: Input sanitization and validation
- **Session Management**: UUID tokens with 7-day expiry
- **Rate Limiting**: Prevents brute force attacks (5 requests/minute for auth)
- **CORS**: Restrictive origin validation
- **App Key**: Required for all API requests
- **Admin Access Control**: Separate middleware for admin routes
- **Input Validation**: All user inputs validated before processing

## WebSocket

The backend provides WebSocket connections for real-time updates:

- Match completion notifications
- Bet settlement results
- Wallet balance updates
- League progression events
- Admin bet creation and settlement
- 50-50 bet updates

## Database Migrations

Migrations run automatically on startup and create:

- Users table with wallet relationships
- Leagues with configurable team counts
- Teams, players, and coaches
- Fixtures and match results
- Bets (league, admin, 50-50, quick match, winner bet)
- Sessions with expiry
- Quick matches
- Notifications
- League tables and top scorers

## Environment Variables

```env
PORT=8080
ENVIRONMENT=production
APP_SECRET=your_secret_key
ALLOWED_ORIGINS=https://your-frontend-domain.com
POSTGRES_DSN=your_postgres_connection_string
REDIS_ADDR=your_redis_connection_string
```

## License

Proprietary software. All rights reserved.