import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:http/http.dart' as http;
import '../types.dart';
import 'headers.dart';

/// iLink HTTP API 客户端
class ProtocolClient {
  String _baseUrl;
  final http.Client _httpClient = http.Client();

  ProtocolClient({String? baseUrl})
    : _baseUrl = baseUrl ?? WeChatConstants.defaultBaseURL;

  /// 更新基础URL
  void updateBaseUrl(String baseUrl) {
    _baseUrl = baseUrl;
  }

  /// 关闭客户端
  void close() {
    _httpClient.close();
  }

  // ==================== 登录相关 API ====================

  /// 获取登录二维码
  Future<QRCodeResponse> getBotQRCode() async {
    final url =
        '$_baseUrl/ilink/bot/get_bot_qrcode?bot_type=${WeChatConstants.botType}';
    final response = await _get(url, headers: Headers.getHeaders());
    return QRCodeResponse.fromJson(response);
  }

  /// 轮询二维码状态
  Future<QRCodeStatusResponse> getQRCodeStatus(String qrcode) async {
    final encodedQrcode = Uri.encodeQueryComponent(qrcode);
    final url = '$_baseUrl/ilink/bot/get_qrcode_status?qrcode=$encodedQrcode';
    final response = await _get(
      url,
      headers: Headers.getHeaders(),
      timeout: const Duration(seconds: 35),
    );
    return QRCodeStatusResponse.fromJson(response);
  }

  // ==================== 消息相关 API ====================

  /// 获取更新（长轮询收消息）
  Future<GetUpdatesResponse> getUpdates({
    required String token,
    String cursor = '',
  }) async {
    final url = '$_baseUrl/ilink/bot/getupdates';
    final body = {'get_updates_buf': cursor, 'base_info': BaseInfo().toJson()};
    final response = await _post(
      url,
      body: body,
      headers: Headers.authHeaders(token),
      timeout: WeChatConstants.getUpdatesTimeout,
    );
    return GetUpdatesResponse.fromJson(response);
  }

  /// 发送消息
  Future<void> sendMessage({
    required String token,
    required WireMessage message,
  }) async {
    final url = '$_baseUrl/ilink/bot/sendmessage';
    final body = {'msg': message.toJson(), 'base_info': BaseInfo().toJson()};
    await _post(url, body: body, headers: Headers.authHeaders(token));
  }

  /// 获取配置（获取 typing_ticket）
  Future<GetConfigResponse> getConfig({
    required String token,
    required String ilinkUserId,
    required String contextToken,
  }) async {
    final url = '$_baseUrl/ilink/bot/getconfig';
    final body = {
      'ilink_user_id': ilinkUserId,
      'context_token': contextToken,
      'base_info': BaseInfo().toJson(),
    };
    final response = await _post(
      url,
      body: body,
      headers: Headers.authHeaders(token),
    );
    return GetConfigResponse.fromJson(response);
  }

  /// 发送输入状态
  Future<void> sendTyping({
    required String token,
    required String ilinkUserId,
    required String typingTicket,
    required int status, // 1=开始输入, 2=停止输入
  }) async {
    final url = '$_baseUrl/ilink/bot/sendtyping';
    final body = {
      'ilink_user_id': ilinkUserId,
      'typing_ticket': typingTicket,
      'status': status,
      'base_info': BaseInfo().toJson(),
    };
    await _post(
      url,
      body: body,
      headers: Headers.authHeaders(token),
      timeout: const Duration(seconds: 10),
    );
  }

  // ==================== CDN 上传相关 API ====================

  /// 获取上传URL
  Future<GetUploadURLResponse> getUploadURL({
    required String token,
    required String fileKey,
    required MediaType mediaType,
    required String toUserId,
    required int rawSize,
    required String rawFileMd5,
    required int fileSize,
    required String aesKey,
  }) async {
    final url = '$_baseUrl/ilink/bot/getuploadurl';
    final body = {
      'filekey': fileKey,
      'media_type': mediaType.value,
      'to_user_id': toUserId,
      'rawsize': rawSize,
      'rawfilemd5': rawFileMd5,
      'filesize': fileSize,
      'no_need_thumb': true,
      'aeskey': aesKey,
      'base_info': BaseInfo().toJson(),
    };
    final response = await _post(
      url,
      body: body,
      headers: Headers.authHeaders(token),
    );
    return GetUploadURLResponse.fromJson(response);
  }

  /// 上传文件到CDN
  Future<String> uploadToCDN({
    required String uploadUrl,
    required Uint8List encryptedData,
    required Map<String, String> headers,
  }) async {
    final uri = Uri.parse(uploadUrl);
    final requestHeaders = {
      'Content-Type': 'application/octet-stream',
      ...headers,
    };

    final response = await _httpClient.post(
      uri,
      headers: requestHeaders,
      body: encryptedData,
    );

    if (response.statusCode != 200) {
      throw HttpException(
        'CDN upload failed: ${response.statusCode}, ${response.body}',
      );
    }

    // 从响应头中获取 x-encrypted-param
    final encryptedParam = response.headers['x-encrypted-param'];
    if (encryptedParam == null || encryptedParam.isEmpty) {
      throw HttpException('CDN upload response missing x-encrypted-param');
    }

    return encryptedParam;
  }

  /// 从CDN下载文件
  Future<Uint8List> downloadFromCDN(String downloadUrl) async {
    final uri = Uri.parse(downloadUrl);
    final response = await _httpClient.get(uri);

    if (response.statusCode != 200) {
      throw HttpException('CDN download failed: ${response.statusCode}');
    }

    return response.bodyBytes;
  }

  // ==================== HTTP 基础方法 ====================

  /// GET 请求
  Future<Map<String, dynamic>> _get(
    String url, {
    required Map<String, String> headers,
    Duration? timeout,
  }) async {
    final uri = Uri.parse(url);

    final response = await _httpClient
        .get(uri, headers: headers)
        .timeout(timeout ?? const Duration(seconds: 15));

    if (response.statusCode != 200) {
      throw HttpException(
        'HTTP ${response.statusCode}: ${_decodeResponse(response)}',
      );
    }

    // 使用 UTF-8 解码响应体
    final bodyString = _decodeResponse(response);
    return jsonDecode(bodyString) as Map<String, dynamic>;
  }

  /// POST 请求
  Future<Map<String, dynamic>> _post(
    String url, {
    required Map<String, dynamic> body,
    required Map<String, String> headers,
    Duration? timeout,
  }) async {
    final uri = Uri.parse(url);
    final bodyJson = jsonEncode(body);

    final response = await _httpClient
        .post(uri, headers: headers, body: bodyJson)
        .timeout(timeout ?? const Duration(seconds: 15));

    // 使用 UTF-8 解码响应体
    final bodyString = _decodeResponse(response);

    if (response.statusCode != 200) {
      throw HttpException('HTTP ${response.statusCode}: $bodyString');
    }

    return jsonDecode(bodyString) as Map<String, dynamic>;
  }

  /// 解码 HTTP 响应体，确保正确处理 UTF-8 编码
  String _decodeResponse(http.Response response) {
    // 首先尝试从 Content-Type header 获取编码
    final contentType = response.headers['content-type'];
    if (contentType != null &&
        contentType.toLowerCase().contains('charset=utf-8')) {
      return utf8.decode(response.bodyBytes);
    }

    // 默认使用 UTF-8 解码
    try {
      return utf8.decode(response.bodyBytes);
    } catch (e) {
      // 如果 UTF-8 解码失败，回退到默认的 body（Latin-1）
      return response.body;
    }
  }
}
