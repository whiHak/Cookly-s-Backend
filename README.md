# Food Recipe Backend

A powerful backend service for a food recipe sharing platform built with Go, Hasura, and PostgreSQL.

## Features

- User authentication with JWT
- Recipe management (create, read, update, delete)
- Recipe interactions (like, bookmark, comment, rate)
- File upload for recipe images
- Payment integration with Chapa
- GraphQL API with Hasura
- PostgreSQL database with advanced features
- Docker support

## Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose
- PostgreSQL 15
- Hasura GraphQL Engine
- Chapa account for payments

## Getting Started

1. Clone the repository:
```bash
git clone https://github.com/whiHak/food-recipe-backend.git
cd food-recipe-backend
```

2. Copy the example environment file and update the values:
```bash
cp .env.example .env
```

3. Start the services using Docker Compose:
```bash
docker-compose up -d
```

4. Install Go dependencies:
```bash
go mod download
```

5. Run the migrations:
```bash
hasura migrate apply
```

6. Start the server:
```bash
go run cmd/main.go
```

The server will start on port 5000 (or the port specified in your .env file).

## API Documentation

### Authentication

#### Register
```http
POST /api/auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securepassword",
  "full_name": "John Doe"
}
```

#### Login
```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword"
}
```

### Recipes

#### Create Recipe
```http
POST /api/recipes
Authorization: Bearer <token>
Content-Type: multipart/form-data

{
  "title": "Delicious Pasta",
  "description": "A simple and tasty pasta recipe",
  "preparation_time": 30,
  "category_id": "uuid",
  "featured_image": <file>,
  "images": [<file1>, <file2>],
  "steps": [
    {
      "step_number": 1,
      "description": "Boil the pasta",
      "image_url": "optional-image-url"
    }
  ],
  "ingredients": [
    {
      "ingredient_id": "uuid",
      "quantity": "200",
      "unit": "g"
    }
  ],
  "price": 10.99
}
```

#### Get Recipe
```http
GET /api/recipes/:id
Authorization: Bearer <token>
```

#### Update Recipe
```http
PUT /api/recipes/:id
Authorization: Bearer <token>
Content-Type: multipart/form-data

{
  // Same structure as create recipe
}
```

#### Delete Recipe
```http
DELETE /api/recipes/:id
Authorization: Bearer <token>
```

### Recipe Interactions

#### Like Recipe
```http
POST /api/recipes/:id/like
Authorization: Bearer <token>
```

#### Unlike Recipe
```http
DELETE /api/recipes/:id/like
Authorization: Bearer <token>
```

#### Bookmark Recipe
```http
POST /api/recipes/:id/bookmark
Authorization: Bearer <token>
```

#### Unbookmark Recipe
```http
DELETE /api/recipes/:id/bookmark
Authorization: Bearer <token>
```

#### Rate Recipe
```http
POST /api/recipes/:id/rate
Authorization: Bearer <token>
Content-Type: application/json

{
  "rating": 5
}
```

#### Comment on Recipe
```http
POST /api/recipes/:id/comment
Authorization: Bearer <token>
Content-Type: application/json

{
  "content": "Great recipe!"
}
```

## Database Schema

The application uses PostgreSQL with the following main tables:
- users
- recipes
- categories
- recipe_steps
- ingredients
- recipe_ingredients
- recipe_images
- recipe_likes
- recipe_bookmarks
- recipe_comments
- recipe_ratings
- recipe_purchases

## Hasura Setup

1. Access the Hasura Console at http://localhost:8080/console
2. Apply the metadata:
```bash
hasura metadata apply
```
3. Check the API Explorer to test the GraphQL API

## File Upload

- Supported file types: .jpg, .jpeg, .png, .gif
- Maximum file size: 5MB
- Files are stored in the `uploads` directory
- Each file gets a unique UUID-based filename

## Payment Integration

The application uses Chapa for payments. To test payments:

1. Get your API keys from Chapa
2. Update the `CHAPA_SECRET_KEY` in your .env file
3. Test payments using Chapa's test cards

## Development

### Project Structure
```
.
├── cmd/
│   └── main.go
├── pkg/
│   ├── auth/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── payment/
│   └── recipe/
├── migrations/
├── uploads/
├── .env
├── .gitignore
├── docker-compose.yml
├── go.mod
└── README.md
```

### Adding New Features

1. Create necessary database migrations
2. Update the GraphQL schema in Hasura
3. Add new models in `pkg/models`
4. Implement the service logic
5. Add new handlers
6. Update the routes in `cmd/main.go`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Deployment Guide

### Prerequisites
- PostgreSQL database
- Hasura Cloud account (or self-hosted Hasura instance)
- A platform to deploy the Go API (Railway, Render, or Heroku)

### Environment Variables
Make sure to set these environment variables in your deployment platform:

```env
# Database
DATABASE_URL=your_postgres_connection_string

# JWT
JWT_SECRET=your_jwt_secret_key
HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"your_jwt_secret_key"}

# Hasura
HASURA_ENDPOINT=your_hasura_endpoint
HASURA_ADMIN_SECRET=your_hasura_admin_secret

# Server
PORT=5000
ENVIRONMENT=production

# File Upload
MAX_FILE_SIZE=5242880
UPLOAD_DIR=./uploads
ALLOWED_FILE_TYPES=.jpg,.jpeg,.png,.gif

# Chapa Payment (if using payment features)
CHAPA_SECRET_KEY=your_chapa_secret_key
CHAPA_CALLBACK_URL=your_api_url/api/payment/callback
CHAPA_RETURN_URL=your_frontend_url/payment/success
```

### Deployment Steps

1. **Deploy Hasura**:
   - Create a Hasura Cloud account at https://hasura.io/cloud/
   - Create a new project
   - Note down your Hasura endpoint URL
   - Set up your admin secret
   - Connect your PostgreSQL database

2. **Deploy the Go API**:

   #### Using Railway:
   ```bash
   # Install Railway CLI
   npm i -g @railway/cli

   # Login to Railway
   railway login

   # Create a new project
   railway init

   # Deploy
   railway up
   ```

   #### Using Render:
   - Connect your GitHub repository
   - Create a new Web Service
   - Select Go
   - Add your environment variables
   - Deploy

   #### Using Heroku:
   ```bash
   # Install Heroku CLI
   # Login to Heroku
   heroku login

   # Create a new app
   heroku create your-app-name

   # Set environment variables
   heroku config:set JWT_SECRET=your_jwt_secret
   heroku config:set HASURA_ENDPOINT=your_hasura_endpoint
   heroku config:set HASURA_ADMIN_SECRET=your_hasura_admin_secret
   # ... set other environment variables

   # Deploy
   git push heroku main
   ```

3. **Update Frontend Configuration**:
   - Update your frontend API endpoint to point to your deployed API
   - Example:
   ```javascript
   const API_URL = 'https://your-deployed-api.com';
   ```

### Testing the Deployment

1. Test registration:
```bash
curl -X POST https://your-api.com/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"password123","fullName":"Test User"}'
```

2. Test login:
```bash
curl -X POST https://your-api.com/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

3. Test protected routes:
```bash
curl -X GET https://your-api.com/api/recipes \
  -H "Authorization: Bearer your_token"
```

### Important Notes

1. Make sure your Hasura instance is properly configured with the correct permissions
2. Ensure your PostgreSQL database is accessible from your deployed API
3. Set up proper CORS configuration if your frontend is hosted on a different domain
4. Consider setting up a CDN for file uploads
5. Implement proper SSL/TLS for security
6. Set up monitoring and logging
7. Configure automatic backups for your database 
