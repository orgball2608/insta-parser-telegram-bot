# 🤖 Instagram Parser Telegram Bot

<!-- <p align="center">
  <img src="https://i.imgur.com/your-logo-image-url.png" alt="Bot Logo" width="150"/>
</p> -->

<p align="center">
  A robust, high-performance Golang Telegram bot for downloading Instagram content, including Stories, Highlights, Posts, and Reels. Built with a focus on reliability, scalability, and a great user experience.
</p>

<p align="center">
  <!-- Badges -->
  <a href="https://github.com/orgball2608/insta-parser-telegram-bot/actions/workflows/test.yml">
    <img src="https://github.com/orgball2608/insta-parser-telegram-bot/actions/workflows/test.yml/badge.svg" alt="Tests">
  </a>
  <a href="https://github.com/orgball2608/insta-parser-telegram-bot/actions/workflows/build_and_push.yml">
    <img src="https://github.com/orgball2608/insta-parser-telegram-bot/actions/workflows/build_and_push.yml/badge.svg" alt="Build and Push">
  </a>
  <a href="https://goreportcard.com/report/github.com/orgball2608/insta-parser-telegram-bot">
    <img src="https://goreportcard.com/badge/github.com/orgball2608/insta-parser-telegram-bot" alt="Go Report Card">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT">
  </a>
</p>

---

## ✨ Features

-   **Automatic Story Subscription**: Subscribe to users and automatically receive their new stories.
-   **On-Demand Downloads**: Fetch content instantly with simple commands.
    -   `/story <username>`: Get all current stories.
    -   `/highlights <username>`: Get all highlight albums.
    -   `/post <url>`: Download a single post or an album.
    -   `/reel <url>`: Download a Reel video.
-   **High Performance**: Utilizes a worker pool (`ants`) to handle multiple scraping jobs concurrently.
-   **Reliable & Resilient**:
    -   Smart retry mechanism with backoff for network or scraper failures.
    -   User-friendly feedback with real-time status updates (e.g., "Fetching...", "Retrying...").
-   **Clean Architecture**:
    -   Well-structured project layout (`cmd`, `internal`, `pkg`).
    -   Dependency Injection with `uber/fx` for a modular and testable codebase.
-   **Containerized**: Fully containerized with Docker and Docker Compose for easy deployment.
-   **CI/CD Pipeline**: Automated testing, security scanning (`Trivy`, `Dockle`), and image publishing via GitHub Actions.

## 🚀 Getting Started

### Prerequisites

-   Go (version 1.21 or higher)
-   Docker & Docker Compose (for containerized deployment)
-   A Telegram Bot Token from [@BotFather](https://t.me/BotFather)

### 1. Local Development

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/orgball2608/insta-parser-telegram-bot.git
    cd insta-parser-telegram-bot
    ```

2.  **Set up environment variables:**
    Create a `.env` file from the example:
    ```bash
    cp .env.example .env
    ```
    Then, fill in your details in the `.env` file, especially `TELEGRAM_BOT_TOKEN`.

3.  **Run the application:**
    The project uses `air` for live-reloading. The `make run` command will handle everything.
    ```bash
    make run
    ```
    If you don't have `air`, install it with: `go install github.com/cosmtrek/air@latest`.

### 2. Docker Deployment

The easiest way to get the bot and its database up and running.

1.  **Set up environment variables:**
    Create and configure your `.env` file as described above.

2.  **Build and run with Docker Compose:**
    ```bash
    docker-compose up --build -d
    ```
    The `-d` flag runs the containers in detached mode.

3.  **To view logs:**
    ```bash
    docker-compose logs -f app
    ```

4.  **To stop:**
    ```bash
    docker-compose down
    ```

## 🛠️ Project Structure

The project follows a standard Go project layout to maintain a clean and scalable architecture.
```
.
├── cmd/                # Main application entrypoint
├── internal/           # Private application code (not for export)
│   ├── app/            # Application setup, dependency injection (FX)
│   ├── command/        # Telegram command handlers
│   ├── domain/         # Core business entities (Story, Post, etc.)
│   ├── instagram/      # Instagram client logic (scraping adapter)
│   ├── parser/         # Scheduled jobs and processing logic
│   ├── repositories/   # Data access layer (PostgreSQL)
│   │   ├── currentstory/ # Current stories repository
│   │   ├── highlights/   # Highlights repository
│   │   ├── story/        # Stories repository
│   │   ├── subscription/ # Subscriptions repository
│   │   └── fx/           # Repository dependency injection
│   └── telegram/       # Telegram client implementation
├── pkg/                # Public libraries safe to use by other projects
│   ├── config/         # Configuration handling
│   ├── errors/         # Error handling utilities
│   ├── logger/         # Logging utilities
│   ├── middleware/     # HTTP middleware
│   ├── pgx/            # PostgreSQL connection utilities
│   └── retry/          # Retry mechanism for operations
├── migrations/         # Database migrations (Goose)
└── tools/              # Supporting tools for development
    └── migrate/        # Migration tool
```


## 🤖 Bot Commands

-   `/start`, `/help` - Shows the help message.
-   `/subscribe <username>` - Subscribe to new stories from a user.
-   `/unsubscribe <username>` - Unsubscribe from a user.
-   `/listsubscriptions` - Show your current subscriptions.
-   `/story <username>` - Fetch current stories.
-   `/highlights <username>` - Fetch all highlight albums.
-   `/post <url>` - Download a post or album.
-   `/reel <url>` - Download a Reel.

## 🧰 Development

This project comes with a handy `Makefile` for common development tasks.

-   `make run`: Run the app locally with live-reloading.
-   `make build`: Build the application binary.
-   `make test`: Run all tests with coverage.
-   `make lint`: Run the GolangCI-Lint linter.
-   `make mock`: Generate mocks using Mockery.
-   `make migrate-up`: Apply all pending database migrations.
-   `make migrate-down`: Rollback the last database migration.
-   `make create-migration name=<name>`: Create a new SQL migration file.

## 🤝 Contributing

Contributions are welcome! If you have ideas for new features or improvements, feel free to open an issue or submit a pull request.

1.  Fork the repository.
2.  Create your feature branch (`git checkout -b feature/AmazingFeature`).
3.  Commit your changes (`git commit -m 'Add some AmazingFeature'`).
4.  Push to the branch (`git push origin feature/AmazingFeature`).
5.  Open a Pull Request.

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.