# Challenge Go

A Go app with user authentication (email/password and Google OAuth), profile management, and a Mysql database.

## Run with Docker

1. Clone the repository and copy the example env file:

```
cp .env.example .env
```

2. Open `.env` and add your Google OAuth credentials (`GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`). You can get these from https://console.cloud.google.com/apis/credentials

3. Start all services:

```
docker compose up --build
```

4. Open http://localhost:8081 in your browser.