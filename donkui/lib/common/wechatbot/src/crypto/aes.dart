import 'dart:convert';
import 'dart:io';
import 'dart:math';
import 'dart:typed_data';
import 'package:encrypt/encrypt.dart';
import 'package:crypto/crypto.dart' as crypto;

/// AES-128-ECB 加密/解密工具类
/// 用于微信 iLink CDN 媒体文件的加解密
class AESCrypto {
  /// AES-128-ECB 加密（带PKCS7填充）
  static Uint8List encrypt(Uint8List plaintext, Uint8List key) {
    if (key.length != 16) {
      throw ArgumentError('AES key must be 16 bytes, got ${key.length}');
    }

    final padded = _pkcs7Pad(plaintext, 16);
    final encrypter = Encrypter(
      AES(
        Key(key),
        mode: AESMode.ecb,
        padding: null, // 我们手动处理填充
      ),
    );

    final encrypted = encrypter.encryptBytes(padded);
    return Uint8List.fromList(encrypted.bytes);
  }

  /// AES-128-ECB 解密（去除PKCS7填充）
  static Uint8List decrypt(Uint8List ciphertext, Uint8List key) {
    if (key.length != 16) {
      throw ArgumentError('AES key must be 16 bytes, got ${key.length}');
    }
    if (ciphertext.length % 16 != 0) {
      throw ArgumentError('Ciphertext length must be multiple of 16');
    }

    final encrypter = Encrypter(
      AES(
        Key(key),
        mode: AESMode.ecb,
        padding: null, // 我们手动处理填充
      ),
    );

    final decrypted = encrypter.decryptBytes(Encrypted(ciphertext));

    return _pkcs7Unpad(Uint8List.fromList(decrypted));
  }

  /// 生成随机16字节AES密钥
  static Uint8List generateKey() {
    final random = Random.secure();
    return Uint8List.fromList(List.generate(16, (_) => random.nextInt(256)));
  }

  /// PKCS7填充
  static Uint8List _pkcs7Pad(Uint8List data, int blockSize) {
    final padding = blockSize - (data.length % blockSize);
    final result = Uint8List(data.length + padding);
    result.setAll(0, data);
    result.fillRange(data.length, result.length, padding);
    return result;
  }

  /// PKCS7去除填充
  static Uint8List _pkcs7Unpad(Uint8List data) {
    if (data.isEmpty) return data;
    final padding = data.last;
    if (padding > 16 || padding == 0) return data;
    // 验证填充
    for (int i = data.length - padding; i < data.length; i++) {
      if (data[i] != padding) return data;
    }
    return Uint8List.sublistView(data, 0, data.length - padding);
  }

  /// 计算MD5
  static String md5(Uint8List data) {
    return crypto.md5.convert(data).toString();
  }

  /// 计算文件的MD5
  static Future<String> md5File(String filePath) async {
    final file = File(filePath);
    final bytes = await file.readAsBytes();
    return md5(bytes);
  }
}

/// AES Key 编解码工具类
class AESKeyCodec {
  /// 解码AES Key（支持多种格式）
  /// - 直接hex字符串（32字符）
  /// - base64编码的原始16字节
  /// - base64编码的hex字符串
  static Uint8List decode(String encoded) {
    // 直接hex字符串
    final hexPattern = RegExp(r'^[0-9a-fA-F]{32}$');
    if (hexPattern.hasMatch(encoded)) {
      return _hexDecode(encoded);
    }

    // 尝试base64解码
    Uint8List decoded;
    try {
      decoded = base64Decode(encoded);
    } catch (e) {
      // 尝试URL-safe base64
      final normalized = encoded.replaceAll('-', '+').replaceAll('_', '/');
      decoded = base64Decode(normalized);
    }

    if (decoded.length == 16) {
      return decoded;
    }

    // base64编码的hex字符串
    if (decoded.length == 32) {
      final hexStr = utf8.decode(decoded);
      if (hexPattern.hasMatch(hexStr)) {
        return _hexDecode(hexStr);
      }
    }

    throw ArgumentError('Invalid AES key format: $encoded');
  }

  /// 编码为hex字符串（用于getuploadurl）
  static String encodeHex(Uint8List key) {
    return _hexEncode(key);
  }

  /// 编码为base64(hex)（用于CDNMedia.aes_key）
  static String encodeBase64(Uint8List key) {
    return base64Encode(utf8.encode(_hexEncode(key)));
  }

  /// 编码为base64(原始字节)
  static String encodeBase64Raw(Uint8List key) {
    return base64Encode(key);
  }

  static String _hexEncode(Uint8List data) {
    return data.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  }

  static Uint8List _hexDecode(String hex) {
    final result = Uint8List(hex.length ~/ 2);
    for (int i = 0; i < hex.length; i += 2) {
      result[i ~/ 2] = int.parse(hex.substring(i, i + 2), radix: 16);
    }
    return result;
  }
}
