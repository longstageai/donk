const String name = "核动驴";
const String name2 = "Donk";
const String logo = "assets/img/app.ico";
const String logo2 = "assets/img/app2.ico";
const String user = "assets/user.png";

const double circular = 10;
const String version = "1.0.0";

// 服务器基础配置
const String serverHost = "localhost";
const int serverPort = 65434;

// HTTP API 基础地址
const String apiBaseUrl = "http://$serverHost:$serverPort/api/v1";

// WebSocket 配置
const String wsUrl = "ws://$serverHost:$serverPort/ws/events";

// SSE 配置
const String sseUrl = "http://$serverHost:$serverPort/api/v1/chat";
