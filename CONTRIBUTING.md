# Contributing to 3X-UI

## Local Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/govnoeby/3x-ui.git
   cd 3x-ui
   ```

2. Create a directory named `x-ui` in the project root:
   ```bash
   mkdir x-ui
   ```

3. Copy environment config:
   ```bash
   cp .env.example .env
   ```

4. Build the frontend:
   ```bash
   cd frontend
   npm ci
   npm run build
   cd ..
   ```

5. Run the application:
   ```bash
   go run main.go
   ```

## Project Structure

```
├── main.go              # Entry point
├── config/              # Configuration management
├── database/            # Database models and initialization
├── web/                 # Web server, controllers, services
│   ├── controller/      # HTTP handlers
│   ├── service/         # Business logic
│   ├── middleware/      # HTTP middleware
│   ├── entity/          # Data transfer objects
│   └── dist/            # Built frontend (gitignored)
├── frontend/            # Vue 3 frontend
├── sub/                 # Subscription server
├── xray/                # Xray-core integration
├── util/                # Utility packages
└── docs/                # Swagger documentation
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Code Style

- Go code follows standard `gofmt` formatting
- Frontend code follows ESLint configuration
- Write tests for new functionality where possible

## Questions?

Open an issue or check the [Wiki](https://github.com/govnoeby/3x-ui/wiki).
