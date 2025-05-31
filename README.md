# Instagram Parser Telegram Bot

A Golang-based Telegram bot that parses and processes Instagram content.

## Features
- Instagram content parsing and processing
- Telegram bot integration
- Clean architecture implementation
- Dependency injection using uber/fx
- Docker support

## Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose (optional)
- Telegram Bot Token
- Instagram credentials

## Getting Started

### Local Development
1. Clone the repository
```bash
git clone https://github.com/your-username/insta-parser-telegram-bot.git
cd insta-parser-telegram-bot
```

2. Install dependencies
```bash
go mod download
```

3. Set up environment variables (create a .env file)
```env
TELEGRAM_BOT_TOKEN=your_bot_token
INSTAGRAM_USERNAME=your_instagram_username
INSTAGRAM_PASSWORD=your_instagram_password
```

4. Run the application
```bash
make run
```

### Docker Deployment
1. Build and run using Docker Compose
```bash
docker-compose up --build
```

## Project Structure
```
.
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── app/              # Application setup and DI
│   ├── domain/           # Business logic and entities
│   ├── telegram/         # Telegram bot implementation
│   ├── instagram/        # Instagram client implementation
│   ├── parser/           # Content parsing logic
│   ├── repositories/     # Data access layer
│   └── command/          # Command handlers
├── pkg/                  # Public libraries
├── migrations/           # Database migrations
└── tools/               # Development tools
```

## Development

### Available Make Commands
```bash
make run          # Run the application
make build        # Build the application
make test         # Run tests
make lint         # Run linters
make mock         # Generate mocks
```

### Contributing
1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License
This project is licensed under the MIT License - see the LICENSE file for details.

