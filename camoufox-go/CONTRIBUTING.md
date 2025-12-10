# Contributing to camoufox-go

Thank you for your interest in contributing to camoufox-go! 

## How to Contribute

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/my-feature`
3. **Make your changes**
4. **Run tests**: `go test ./...`
5. **Submit a pull request**

## Development Setup

```bash
# Clone the repo
git clone https://github.com/uzzalhcse/camoufox-go.git
cd camoufox-go

# Install dependencies
go mod download

# Install Python dependencies
pip install browserforge camoufox

# Install Camoufox browser
python -c "import camoufox; camoufox.install()"

# Run tests
go test ./... -v
```

## Code Style

- Follow standard Go formatting (`gofmt`)
- Add comments for exported functions
- Write tests for new features

## Reporting Issues

When reporting issues, please include:
- Go version (`go version`)
- Camoufox version
- Python version
- Operating system
- Steps to reproduce

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
