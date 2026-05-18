import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';
import '../types.dart';

/// HTTP 请求头生成工具类
class Headers {
  /// 生成公共Header（所有请求）
  static Map<String, String> commonHeaders() {
    return {
      'iLink-App-Id': WeChatConstants.iLinkAppID,
      'iLink-App-ClientVersion': WeChatConstants.iLinkClientVer,
    };
  }

  /// 生成认证Header（POST请求）
  static Map<String, String> authHeaders(String token) {
    return {
      ...commonHeaders(),
      'Content-Type': 'application/json',
      'AuthorizationType': 'ilink_bot_token',
      'Authorization': 'Bearer $token',
      'X-WECHAT-UIN': randomWechatUIN(),
    };
  }

  /// 生成GET请求Header（无需认证）
  static Map<String, String> getHeaders() {
    return {...commonHeaders()};
  }

  /// 生成GET请求Header（需要认证）
  static Map<String, String> getAuthHeaders(String token) {
    return {
      ...commonHeaders(),
      'AuthorizationType': 'ilink_bot_token',
      'Authorization': 'Bearer $token',
      'X-WECHAT-UIN': randomWechatUIN(),
    };
  }

  /// X-WECHAT-UIN 生成算法
  /// 生成4个随机字节 -> 读取为无符号32位整数（大端序）-> 转为十进制字符串 -> Base64编码
  static String randomWechatUIN() {
    // 生成4个随机字节
    final random = Random.secure();
    final buffer = Uint8List(4);
    for (int i = 0; i < 4; i++) {
      buffer[i] = random.nextInt(256);
    }

    // 读取为无符号32位整数（大端序）
    final byteData = ByteData.sublistView(buffer);
    final uint32 = byteData.getUint32(0, Endian.big);

    // 转为十进制字符串，然后Base64编码
    final decimalStr = uint32.toString();
    return base64Encode(utf8.encode(decimalStr));
  }
}
