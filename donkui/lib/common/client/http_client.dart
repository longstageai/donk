import 'dart:convert';
import 'dart:io';

/// HTTP 客户端单例
/// 用于封装 HttpClient 的配置和复用
/// 采用单例模式确保整个应用只创建一个 HttpClient 实例
class HttpClientSingleton {
  static HttpClientSingleton? _instance;
  late HttpClient _client;

  HttpClientSingleton._internal() {
    _client = HttpClient();
    _client.connectionTimeout = const Duration(seconds: 30);
    _client.idleTimeout = const Duration(seconds: 60);
  }

  /// 获取单例实例
  /// 第一次调用时创建实例，之后返回已存在的实例
  static HttpClientSingleton get instance {
    _instance ??= HttpClientSingleton._internal();
    return _instance!;
  }

  /// 获取底层的 HttpClient 实例
  HttpClient get client => _client;

  /// 发送 GET 请求
  /// [url] 请求的完整 URL 地址
  /// 返回响应体的字符串内容
  Future<String> get(String url) async {
    final request = await _client.getUrl(Uri.parse(url));
    final response = await request.close();
    return _handleResponse(response);
  }

  /// 发送 POST 请求
  /// [url] 请求的完整 URL 地址
  /// [body] 可选的请求体，会被序列化为 JSON
  /// 返回响应体的字符串内容
  Future<String> post(String url, {Map<String, dynamic>? body}) async {
    final request = await _client.postUrl(Uri.parse(url));
    request.headers.set('Content-Type', 'application/json');
    if (body != null) {
      request.write(jsonEncode(body));
    }
    final response = await request.close();
    return _handleResponse(response);
  }

  /// 发送 PUT 请求
  /// [url] 请求的完整 URL 地址
  /// [body] 可选的请求体，会被序列化为 JSON
  /// 返回响应体的字符串内容
  Future<String> put(String url, {Map<String, dynamic>? body}) async {
    final request = await _client.putUrl(Uri.parse(url));
    request.headers.set('Content-Type', 'application/json');
    if (body != null) {
      request.write(jsonEncode(body));
    }
    final response = await request.close();
    return _handleResponse(response);
  }

  /// 发送 DELETE 请求
  /// [url] 请求的完整 URL 地址
  /// 返回响应体的字符串内容
  Future<String> delete(String url) async {
    final request = await _client.deleteUrl(Uri.parse(url));
    final response = await request.close();
    return _handleResponse(response);
  }

  /// 处理 HTTP 响应
  /// 将响应体转换为字符串
  Future<String> _handleResponse(HttpClientResponse response) async {
    final body = await response.transform(utf8.decoder).join();
    return body;
  }
}
