# vuDataSim Cluster Manager

A comprehensive load testing and simulation management system with real-time dashboard and backend API.

## 🚀 Features

- **Real-time Dashboard**: Live monitoring of cluster performance with animated charts
- **Load Profile Management**: Configurable load testing profiles (Low, Medium, High, Custom)
- **Interactive Controls**: Start/stop simulation with real-time parameter adjustment
- **Node Status Monitoring**: Visual indicators for active/inactive nodes
- **Real-time Metrics**: Live updates of EPS, Kafka load, ClickHouse load, CPU, and memory usage
- **Log Management**: Filtered log viewing with real-time updates
- **Responsive Design**: Dark/light theme support with modern UI
- **WebSocket Integration**: Real-time bidirectional communication
- **RESTful API**: Complete backend API for integration

## 📁 Project Structure

```
Load-Testing-Tool/
├── src/
│   └── main.go              # Go backend server
├── static/
│   ├── index.html           # Main dashboard interface
│   ├── styles.css           # Custom styles and animations
│   └── script.js            # Frontend JavaScript functionality
├── go.mod                   # Go module dependencies
└── README.md               # This file
```

## 🛠️ Technologies Used

- **Backend**: Go with Gorilla Mux, WebSocket support
- **Frontend**: HTML5, CSS3, Vanilla JavaScript
- **Styling**: Tailwind CSS with custom animations
- **Icons**: Material Symbols
- **Real-time**: WebSocket for live updates

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Modern web browser

### Installation & Setup

1. **Clone or navigate to the project directory:**
   ```bash
   cd Load-Testing-Tool
   ```

2. **Install Go dependencies:**
   ```bash
   go mod tidy
   ```

3. **Start the server:**
   ```bash
   go run src/main.go
   ```

4. **Open your browser:**
   Navigate to `http://localhost:8080`

## 📖 Usage Guide

### Dashboard Overview

The main interface consists of:

- **Sidebar Controls** (Left):
  - Load Profile Selector (dropdown)
  - Target EPS configuration
  - Target Kafka Load configuration
  - Target ClickHouse Load configuration
  - Action buttons (Start/Stop/Sync)

- **Main Dashboard** (Right):
  - Node status indicators
  - Cluster overview table
  - Live charts and metrics
  - Profile summary
  - Real-time logs with filtering

### Load Testing Profiles

- **Low**: Minimal load for testing basic functionality
- **Medium**: Balanced load for standard testing (default)
- **High**: Maximum load for stress testing
- **Custom**: User-defined parameters

### Real-time Features

- **Live Metrics**: Updates every 2 seconds during simulation
- **Animated Charts**: SVG-based charts with smooth animations
- **Node Status**: Visual indicators with pulsing animations
- **Log Streaming**: Real-time log updates with filtering

## 🔌 API Endpoints

### Simulation Control
- `POST /api/simulation/start` - Start load testing simulation
- `POST /api/simulation/stop` - Stop current simulation
- `POST /api/config/sync` - Sync configuration settings

### Data Retrieval
- `GET /api/dashboard` - Get current dashboard data
- `GET /api/logs` - Get filtered log entries
- `GET /api/health` - Health check endpoint

### Real-time Communication
- `WebSocket /ws` - Real-time bidirectional updates

### Node Management
- `PUT /api/nodes/{nodeId}/metrics` - Update node metrics

## 🔧 Configuration

### Environment Variables

```bash
PORT=:8080              # Server port
STATIC_DIR=./static     # Static files directory
```

### API Configuration

The backend API is configured to accept requests from any origin (CORS enabled). For production deployment, update the CORS settings in `main.go`.

## 🎨 Customization

### Styling

Custom styles are located in `static/styles.css`:
- Material Design color scheme
- Custom animations and transitions
- Responsive design breakpoints
- Dark/light theme support

### JavaScript Functionality

Frontend logic in `static/script.js`:
- Real-time data management
- Form validation
- WebSocket communication
- Chart animations
- Interactive controls

## 🚦 Development

### Running in Development Mode

```bash
# Terminal 1 - Start the backend server
go run src/main.go

# Terminal 2 - Watch for frontend changes (optional)
# You can add file watchers or live reload tools
```

### Building for Production

```bash
# Build the Go binary
go build -o vuDataSim src/main.go

# Run the compiled binary
./vuDataSim
```

## 🔍 Monitoring

### Health Checks

- **Endpoint**: `GET /api/health`
- **Response**: Server status, version, uptime

### Logs

Server logs are output to stdout with timestamps and request details.

## 🔒 Security Considerations

- CORS is currently configured for development (allows all origins)
- Update CORS settings for production deployment
- Consider adding authentication/authorization
- Validate and sanitize all input data

## 🐛 Troubleshooting

### Common Issues

1. **Port already in use**: Change the port in `main.go` or stop the conflicting service
2. **Static files not loading**: Ensure the `static/` directory exists and contains the required files
3. **WebSocket connection failed**: Check that the server is running and accessible

### Debug Mode

Enable verbose logging by modifying the log level in `main.go`.

## 📈 Performance

- **Concurrent Connections**: Supports multiple WebSocket clients
- **Real-time Updates**: Optimized for 2-second update intervals
- **Memory Usage**: Efficient state management with mutex locking
- **Responsive UI**: Smooth animations with hardware acceleration

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## 🆘 Support

For issues, questions, or contributions, please:
- Check existing documentation
- Review open issues
- Create new issues with detailed information
- Submit pull requests with clear descriptions

---

**Built with ❤️ for load testing and performance monitoring**