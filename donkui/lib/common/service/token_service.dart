import 'dart:convert';
import 'package:donk/app/conf/config.dart' as app_config;
import '../client/http_client.dart';

/// Token 统计服务类
/// 封装 Token 用量统计相关的 HTTP API 调用
class TokenService {
  static final String _baseUrl = app_config.apiBaseUrl;
  static final HttpClientSingleton _http = HttpClientSingleton.instance;

  /// 获取 Token 使用记录列表
  /// [page] 页码，从1开始，默认1
  /// [pageSize] 每页条数，默认20，最大100
  static Future<Map<String, dynamic>> getTokenUsage({
    int page = 1,
    int pageSize = 20,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      'page_size': pageSize.toString(),
    };

    final uri = Uri.parse(
      '$_baseUrl/tokens/usage',
    ).replace(queryParameters: queryParams);
    final body = await _http.get(uri.toString());
    return jsonDecode(body) as Map<String, dynamic>;
  }

  /// 获取今日 Token 预算状态
  static Future<Map<String, dynamic>> getTokenBudget() async {
    final body = await _http.get('$_baseUrl/tokens/budget');
    return jsonDecode(body) as Map<String, dynamic>;
  }
}
