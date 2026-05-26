import 'dart:convert';
import 'package:donk/app/conf/config.dart' as app_config;
import '../client/http_client.dart';

/// Creative 服务类
/// 封装 Creative 多 Agent 循环的 HTTP API 调用
class CreativeService {
  static final String _baseUrl = app_config.apiBaseUrl;
  static final HttpClientSingleton _http = HttpClientSingleton.instance;

  /// 获取 Creative 运行状态
  /// 返回包含 running（是否运行中）、db_status（数据库状态）、session_id（当前会话ID）的信息
  static Future<Map<String, dynamic>> getStatus() async {
    final body = await _http.get('$_baseUrl/creative/status');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 启动 Creative 多 Agent 循环
  static Future<Map<String, dynamic>> startSession() async {
    final body = await _http.post('$_baseUrl/creative/start');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 停止 Creative 多 Agent 循环
  static Future<Map<String, dynamic>> stopSession() async {
    final body = await _http.post('$_baseUrl/creative/stop');
    return jsonDecode(body) as Map<String, dynamic>;
  }
}
