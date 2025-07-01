# Contributing to RSK Event Listener

Thank you for your interest in contributing! We welcome all kinds of contributions: new features, bug fixes, documentation, and more.

## How to Contribute

1. **Fork the repository**  
   Click the "Fork" button at the top right of this page.

2. **Clone your fork**  
   ```bash
   git clone https://github.com/smartdevs17/rsk-event-listener.git
   cd rsk-event-listener
   ```

3. **Create a feature branch**  
   ```bash
   git checkout -b feature/my-feature
   ```

4. **Make your changes**  
   - Write clear, concise code and comments.
   - Add or update tests as needed.
   - Run `go fmt ./...` and `golangci-lint run` to ensure code quality.

5. **Commit your changes**  
   ```bash
   git add .
   git commit -m "Describe your changes"
   ```

6. **Push to your fork**  
   ```bash
   git push origin feature/my-feature
   ```

7. **Open a Pull Request**  
   - Go to your fork on GitHub and click "Compare & pull request".
   - Fill in the PR template and describe your changes.

## Code Style

- Use `go fmt ./...` for formatting.
- Run `golangci-lint run` for linting.
- Write clear commit messages.
- Add tests for new features and bug fixes.

## Running Tests

```bash
go test ./...
# For integration tests:
go test -tags=integration ./...
```

## Reporting Issues

- Use [GitHub Issues](https://github.com/smartdevs17/rsk-event-listener/issues) to report bugs or request features.
- Please provide as much detail as possible, including logs and steps to reproduce.

## Community

- Join the [RSK Discord](https://discord.gg/rsk) for discussions and support.
- See the [README.md](README.md) for more info.

---

Thank you for helping make RSK Event