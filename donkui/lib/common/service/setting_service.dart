import 'dart:convert';
import 'package:donk/app/conf/config.dart' as app_config;
import '../client/http_client.dart';

/// 设置服务类
/// 封装配置管理相关的 HTTP API 调用
class SettingService {
  static final String _baseUrl = app_config.apiBaseUrl;
  static final HttpClientSingleton _http = HttpClientSingleton.instance;

  /// 获取完整配置
  static Future<Map<String, dynamic>> getConfig() async {
    final body = await _http.get('$_baseUrl/config');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 更新完整配置（支持部分更新）
  static Future<Map<String, dynamic>> updateConfig(
    Map<String, dynamic> config,
  ) async {
    final body = await _http.put('$_baseUrl/config', body: config);
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取 LLM 配置
  static Future<Map<String, dynamic>> getLLMConfig() async {
    final body = await _http.get('$_baseUrl/config/llm');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 更新 LLM 配置
  static Future<Map<String, dynamic>> updateLLMConfig({
    required String provider,
    required String model,
    String? apiKey,
    String? baseUrl,
    double? temperature,
    int? maxTokens,
  }) async {
    final body = await _http.put(
      '$_baseUrl/config/llm',
      body: {
        'provider': provider,
        'model': model,
        if (apiKey != null) 'api_key': apiKey,
        if (baseUrl != null) 'base_url': baseUrl,
        if (temperature != null) 'temperature': temperature,
        if (maxTokens != null) 'max_tokens': maxTokens,
      },
    );
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取 Embedding 配置
  static Future<Map<String, dynamic>> getEmbeddingConfig() async {
    final body = await _http.get('$_baseUrl/config/embedding');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 更新 Embedding 配置
  static Future<Map<String, dynamic>> updateEmbeddingConfig({
    required String provider,
    required String model,
    String? apiKey,
    String? baseUrl,
    int? dimension,
  }) async {
    final body = await _http.put(
      '$_baseUrl/config/embedding',
      body: {
        'provider': provider,
        'model': model,
        if (apiKey != null) 'api_key': apiKey,
        if (baseUrl != null) 'base_url': baseUrl,
        if (dimension != null) 'dimension': dimension,
      },
    );
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取 Agent 配置
  static Future<Map<String, dynamic>> getAgentConfig() async {
    final body = await _http.get('$_baseUrl/config/agent');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 更新 Agent 配置
  static Future<Map<String, dynamic>> updateAgentConfig({
    required String name,
    int? maxLoop,
    int? convergeAfter,
    int? timeout,
    int? dailyTokenLimit,
    int? historyMaxEntries,
    int? historyMaxDays,
  }) async {
    final body = await _http.put(
      '$_baseUrl/config/agent',
      body: {
        'name': name,
        if (maxLoop != null) 'max_loop': maxLoop,
        if (convergeAfter != null) 'converge_after': convergeAfter,
        if (timeout != null) 'timeout': timeout,
        if (dailyTokenLimit != null) 'daily_token_limit': dailyTokenLimit,
        if (historyMaxEntries != null) 'history_max_entries': historyMaxEntries,
        if (historyMaxDays != null) 'history_max_days': historyMaxDays,
      },
    );
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取睡眠管理状态
  static Future<Map<String, dynamic>> getSleepStatus() async {
    final body = await _http.get('$_baseUrl/system/sleep');
    final response = jsonDecode(body) as Map<String, dynamic>;
    return response['data'] as Map<String, dynamic>;
  }

  /// 阻止系统睡眠
  static Future<Map<String, dynamic>> preventSleep() async {
    final body = await _http.post(
      '$_baseUrl/system/sleep/prevent',
      body: {'keep_display': false},
    );
    final response = jsonDecode(body) as Map<String, dynamic>;
    return response['data'] as Map<String, dynamic>;
  }

  /// 允许系统睡眠
  static Future<Map<String, dynamic>> allowSleep() async {
    final body = await _http.post('$_baseUrl/system/sleep/allow');
    final response = jsonDecode(body) as Map<String, dynamic>;
    return response['data'] as Map<String, dynamic>;
  }

  /// 获取 Token 预算状态（今日使用量）
  static Future<Map<String, dynamic>> getTokenBudget() async {
    final body = await _http.get('$_baseUrl/tokens/budget');
    final response = jsonDecode(body) as Map<String, dynamic>;
    return response['data'] as Map<String, dynamic>;
  }

  /// 获取知识库配置
  static Future<Map<String, dynamic>> getKnowledgeConfig() async {
    final body = await _http.get('$_baseUrl/config/knowledge');
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 更新知识库配置
  static Future<Map<String, dynamic>> updateKnowledgeConfig({
    required bool enabled,
  }) async {
    final body = await _http.put(
      '$_baseUrl/config/knowledge',
      body: {'enabled': enabled},
    );
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取知识库状态
  static Future<Map<String, dynamic>> getKnowledgeStatus() async {
    final body = await _http.get('$_baseUrl/knowledge/status');
    return jsonDecode(body) as Map<String, dynamic>;
  }
}
