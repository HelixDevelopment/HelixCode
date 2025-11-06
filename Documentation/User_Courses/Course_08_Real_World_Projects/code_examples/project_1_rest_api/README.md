# Project 1: REST API with Authentication

Complete REST API built with HelixCode featuring user authentication, database integration, and comprehensive testing.

## Project Overview

This project demonstrates building a production-ready REST API using HelixCode to accelerate development. You'll create a user management system with:

- User registration and login
- JWT-based authentication
- Password hashing with bcrypt
- PostgreSQL database integration
- Input validation and error handling
- Rate limiting for API security
- Comprehensive test suite
- API documentation

## Technologies Used

- **Framework:** Flask with Flask-RESTful
- **Database:** PostgreSQL with SQLAlchemy ORM
- **Authentication:** JWT tokens via Flask-JWT-Extended
- **Validation:** Marshmallow schemas
- **Testing:** pytest with Flask test client
- **Production Server:** Gunicorn

## Project Structure

```
project_1_rest_api/
├── src/
│   ├── __init__.py
│   ├── app.py              # Application factory
│   ├── config.py           # Configuration management
│   ├── models/
│   │   ├── __init__.py
│   │   └── user.py         # User model
│   ├── resources/
│   │   ├── __init__.py
│   │   ├── auth.py         # Auth endpoints
│   │   └── users.py        # User CRUD endpoints
│   ├── schemas/
│   │   ├── __init__.py
│   │   └── user.py         # Validation schemas
│   └── utils/
│       ├── __init__.py
│       ├── validators.py   # Custom validators
│       └── decorators.py   # Auth decorators
├── tests/
│   ├── __init__.py
│   ├── conftest.py         # Test fixtures
│   ├── test_auth.py
│   └── test_users.py
├── migrations/              # Alembic migrations
├── .env.example
├── requirements.txt
└── README.md
```

## Building with HelixCode

### Phase 1: Project Setup (Chapter 2)

Start HelixCode and create the project structure:

```bash
mkdir project_1_rest_api && cd project_1_rest_api
helixcode --model anthropic/claude-3-opus
```

**Prompts to use:**
1. "Create a Flask REST API project structure with authentication"
2. "Add requirements.txt with Flask, SQLAlchemy, JWT, and testing dependencies"
3. "Create a configuration module with development and production settings"
4. "Set up database models for user management with email and password"

### Phase 2: Authentication Implementation (Chapter 3)

**Prompts to use:**
1. "Implement user registration endpoint with email validation and password hashing"
2. "Create login endpoint that returns JWT tokens"
3. "Add JWT token verification decorator for protected routes"
4. "Implement token refresh functionality"
5. "Add rate limiting to auth endpoints (5 requests per minute)"

### Phase 3: CRUD Operations (Chapter 3)

**Prompts to use:**
1. "Create RESTful endpoints for user CRUD operations"
2. "Add authorization - users can only modify their own data"
3. "Implement pagination for user list endpoint"
4. "Add filtering and sorting to user queries"
5. "Create admin endpoints for user management"

### Phase 4: Testing (Chapter 4)

**Prompts to use:**
1. "Create pytest fixtures for test database and client"
2. "Write tests for user registration with valid and invalid data"
3. "Add tests for login flow and JWT token validation"
4. "Create tests for protected endpoints without authentication"
5. "Add integration tests for complete user workflows"
6. "Aim for 90%+ code coverage"

### Phase 5: Documentation and Deployment (Chapter 4)

**Prompts to use:**
1. "Generate API documentation using OpenAPI/Swagger"
2. "Create Docker configuration for containerized deployment"
3. "Add GitHub Actions CI/CD pipeline"
4. "Create deployment guide for Heroku/AWS"
5. "Add comprehensive README with setup instructions"

## Learning Outcomes

By completing this project, you'll learn:

1. **HelixCode Workflows**
   - Breaking complex projects into manageable prompts
   - Iterating on generated code
   - Using HelixCode for testing
   - Reviewing and committing changes

2. **REST API Development**
   - Designing RESTful endpoints
   - Implementing authentication and authorization
   - Database modeling and migrations
   - Input validation and error handling

3. **Best Practices**
   - Security considerations (password hashing, rate limiting)
   - Testing strategies (unit, integration, fixtures)
   - Configuration management
   - Deployment preparation

## Key Features Demonstrated

### User Registration
```python
POST /api/auth/register
{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "username": "johndoe"
}

Response: 201 Created
{
  "message": "User registered successfully",
  "user_id": 1
}
```

### User Login
```python
POST /api/auth/login
{
  "email": "user@example.com",
  "password": "SecurePass123!"
}

Response: 200 OK
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "johndoe"
  }
}
```

### Protected Endpoint
```python
GET /api/users/me
Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGc...

Response: 200 OK
{
  "id": 1,
  "email": "user@example.com",
  "username": "johndoe",
  "created_at": "2025-11-06T10:30:00Z"
}
```

## Running the Project

### Setup

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Set up environment variables
cp .env.example .env
# Edit .env with your database credentials

# Initialize database
flask db upgrade

# Run development server
flask run
```

### Testing

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=src --cov-report=html

# Run specific test file
pytest tests/test_auth.py -v
```

### Production Deployment

```bash
# Using Gunicorn
gunicorn -w 4 -b 0.0.0.0:8000 "src.app:create_app()"

# Using Docker
docker build -t rest-api .
docker run -p 8000:8000 rest-api
```

## Challenges and Extensions

After completing the basic project, try these extensions:

1. **Email Verification**
   - Add email verification flow
   - Send verification emails
   - Implement token-based verification

2. **Password Reset**
   - Create password reset flow
   - Generate secure reset tokens
   - Send reset emails

3. **OAuth Integration**
   - Add Google OAuth login
   - Support multiple auth providers
   - Link accounts

4. **API Versioning**
   - Implement API versioning (/api/v1/)
   - Support multiple versions
   - Deprecation strategy

5. **Advanced Features**
   - Add role-based access control (RBAC)
   - Implement audit logging
   - Add WebSocket support for real-time updates
   - Create admin dashboard

## HelixCode Tips for This Project

1. **Context Management**
   - Keep relevant files in context
   - Drop test files when implementing features
   - Add test files when writing tests

2. **Incremental Development**
   - Build one endpoint at a time
   - Test after each feature
   - Commit frequently

3. **Using HelixCode Effectively**
   - Be specific in prompts
   - Reference existing code: "Similar to the registration endpoint..."
   - Ask for explanations: "Explain the security implications of..."

4. **Review Generated Code**
   - Always review security-critical code (auth, passwords)
   - Verify SQL queries for injection vulnerabilities
   - Check error handling is comprehensive

## Resources

- Flask Documentation: https://flask.palletsprojects.com/
- Flask-RESTful: https://flask-restful.readthedocs.io/
- JWT Best Practices: https://tools.ietf.org/html/rfc8725
- SQLAlchemy ORM: https://www.sqlalchemy.org/
- pytest Documentation: https://docs.pytest.org/

## Next Project

After completing this REST API, move on to **Project 2: React Dashboard** where you'll build a frontend to consume this API!
