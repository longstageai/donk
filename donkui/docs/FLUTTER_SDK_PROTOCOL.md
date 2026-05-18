# WeChat iLink Bot SDK - Flutter 版本协议文档

本文档基于Go SDK源码分析整理，为Flutter版本SDK开发提供完整的协议参考。

参考文章：https://yarrow.ren/posts/wechat-ilink-llm-bridge-guide/

---

## 目录

1. [架构概览](#架构概览)
2. [核心常量与配置](#核心常量与配置)
3. [HTTP API 协议](#http-api-协议)
4. [认证流程（QR码登录）](#认证流程qr码登录)
5. [消息协议](#消息协议)
6. [媒体处理（CDN上传下载）](#媒体处理cdn上传下载)
7. [加密算法](#加密算法)
8. [Flutter SDK 架构建议](#flutter-sdk-架构建议)

---

## 架构概览

### 服务端点

| 端点 | URL |
|------|-----|
| API Base URL | `https://ilinkai.weixin.qq.com` |
| CDN Base URL | `https://novac2c.cdn.weixin.qq.com/c2c` |

### 模块划分

```
wechatbot-flutter/
├── lib/
│   ├── wechatbot.dart          # 主入口
│   ├── src/
│   │   ├── bot.dart            # Bot客户端
│   │   ├── types.dart          # 数据类型定义
│   │   ├── protocol/
│   │   │   ├── api.dart        # HTTP API调用
│   │   │   └── headers.dart    # 请求头生成
│   │   ├── auth/
│   │   │   └── login.dart      # 登录流程
│   │   └── crypto/
│   │       └── aes.dart        # AES加密解密
│   └── examples/
│       └── echo_bot.dart       # 示例机器人
```

---

## 核心常量与配置

### 常量定义

```dart
class WeChatConstants {
  // API端点
  static const String defaultBaseURL = 'https://ilinkai.weixin.qq.com';
  static const String cdnBaseURL = 'https://novac2c.cdn.weixin.qq.com/c2c';
  
  // 版本信息
  static const String channelVersion = '0.1.0';
  static const String iLinkAppID = 'bot';
  static const String iLinkClientVer = '256'; // 0x00MMNNPP for 0.1.0 = 256
  
  // Bot类型
  static const String botType = '3';
  
  // 超时配置
  static const Duration getUpdatesTimeout = Duration(seconds: 45);
  static const Duration apiTimeout = Duration(seconds: 15);
  static const Duration cdnTimeout = Duration(seconds: 60);
  
  // 重试配置
  static const int maxCDNRetries = 3;
  static const int maxQRRefreshCount = 3;
}
```

### 枚举类型

```dart
/// 消息发送者类型
enum MessageType {
  user(1),
  bot(2);
  
  final int value;
  const MessageType(this.value);
}

/// 消息状态
enum MessageState {
  new_(0),
  generating(1),
  finish(2);
  
  final int value;
  const MessageState(this.value);
}

/// 消息内容类型
enum MessageItemType {
  text(1),
  image(2),
  voice(3),
  file(4),
  video(5);
  
  final int value;
  const MessageItemType(this.value);
}

/// 媒体类型（用于上传）
enum MediaType {
  image(1),
  video(2),
  file(3),
  voice(4);
  
  final int value;
  const MediaType(this.value);
}

/// 内容类型（解析后的消息类型）
enum ContentType {
  text('text'),
  image('image'),
  voice('voice'),
  file('file'),
  video('video');
  
  final String value;
  const ContentType(this.value);
}
```

---

## HTTP API 协议

### 请求头规范

所有请求必须携带以下Header：

#### 公共Header（所有请求）

```dart
Map<String, String> commonHeaders() {
  return {
    'iLink-App-Id': 'bot',
    'iLink-App-ClientVersion': '256',
  };
}
```

#### 认证Header（POST请求）

```dart
Map<String, String> authHeaders(String token) {
  return {
    ...commonHeaders(),
    'Content-Type': 'application/json',
    'AuthorizationType': 'ilink_bot_token',
    'Authorization': 'Bearer $token',
    'X-WECHAT-UIN': randomWechatUIN(),
  };
}
```

#### X-WECHAT-UIN 生成算法

```dart
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

String randomWechatUIN() {
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
```

### API端点列表

| 端点 | 方法 | 路径 | 超时 | 描述 |
|------|------|------|------|------|
| 获取二维码 | GET | `/ilink/bot/get_bot_qrcode?bot_type=3` | 15s | 获取登录二维码 |
| 查询扫码状态 | GET | `/ilink/bot/get_qrcode_status?qrcode={qrcode}` | 35s | 长轮询扫码状态 |
| 获取消息 | POST | `/ilink/bot/getupdates` | 45s | 长轮询获取新消息 |
| 发送消息 | POST | `/ilink/bot/sendmessage` | 15s | 发送消息 |
| 获取配置 | POST | `/ilink/bot/getconfig` | 15s | 获取typing_ticket |
| 发送输入状态 | POST | `/ilink/bot/sendtyping` | 10s | 发送"正在输入"状态 |
| 获取上传URL | POST | `/ilink/bot/getuploadurl` | 15s | 获取CDN上传参数 |

---

## 认证流程（QR码登录）

### 流程图

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   开始登录   │────▶│  获取QR码        │────▶│  轮询扫码状态    │
└─────────────┘     └─────────────────┘     └─────────────────┘
                                                      │
                              ┌───────────────────────┼───────────────────────┐
                              ▼                       ▼                       ▼
                        ┌─────────┐            ┌──────────┐           ┌───────────┐
                        │  wait   │            │ scanned  │           │ confirmed │
                        │(等待扫码)│            │(已扫码)   │           │ (已确认)   │
                        └────┬────┘            └────┬─────┘           └─────┬─────┘
                             │                      │                       │
                             │                      ▼                       │
                             │               ┌────────────┐                 │
                             │               │scaned_but  │                 │
                             │               │_redirect   │─────────────────┘
                             │               │(IDC重定向)  │
                             │               └────────────┘
                             ▼
                        ┌─────────┐
                        │ expired │
                        │(已过期)  │
                        └────┬────┘
                             │
                             └────────────────▶ 重新获取QR码（最多3次）
```

### 1. 获取二维码

**请求：**
```http
GET /ilink/bot/get_bot_qrcode?bot_type=3
iLink-App-Id: bot
iLink-App-ClientVersion: 256
```

**响应：**
```json
{
  "qrcode": "string",           // 二维码标识符
  "qrcode_img_content": "string" // 二维码图片URL
}
```

### 2. 轮询扫码状态

**请求：**
```http
GET /ilink/bot/get_qrcode_status?qrcode={url_encoded_qrcode}
iLink-App-Id: bot
iLink-App-ClientVersion: 256
```

**响应：**
```json
{
  "status": "wait|scaned|confirmed|expired|scaned_but_redirect",
  "bot_token": "string",       // 仅在confirmed时返回
  "ilink_bot_id": "string",    // 仅在confirmed时返回
  "ilink_user_id": "string",   // 仅在confirmed时返回
  "baseurl": "string",         // 可选，服务器地址
  "redirect_host": "string"    // 仅在scaned_but_redirect时返回
}
```

**状态说明：**

| 状态 | 含义 | 处理 |
|------|------|------|
| `wait` | 等待用户扫码 | 继续轮询（2秒间隔） |
| `scaned` | 用户已扫码，等待确认 | 继续轮询，可显示提示 |
| `confirmed` | 登录确认成功 | 获取token，结束登录 |
| `expired` | 二维码已过期 | 重新获取二维码（最多3次） |
| `scaned_but_redirect` | 需要IDC重定向 | 切换到redirect_host继续轮询 |

### 3. 凭证存储结构

```dart
class Credentials {
  final String token;        // Bot Token
  final String baseUrl;      // API基础URL
  final String accountId;    // Bot ID
  final String userId;       // 用户ID
  final DateTime? savedAt;   // 保存时间

  Credentials({
    required this.token,
    required this.baseUrl,
    required this.accountId,
    required this.userId,
    this.savedAt,
  });

  Map<String, dynamic> toJson() => {
    'token': token,
    'baseUrl': baseUrl,
    'accountId': accountId,
    'userId': userId,
    'savedAt': savedAt?.toIso8601String(),
  };

  factory Credentials.fromJson(Map<String, dynamic> json) => Credentials(
    token: json['token'],
    baseUrl: json['baseUrl'],
    accountId: json['accountId'],
    userId: json['userId'],
    savedAt: json['savedAt'] != null ? DateTime.parse(json['savedAt']) : null,
  );
}
```

---

## 消息协议

### 接收消息（getupdates）

**请求：**
```http
POST /ilink/bot/getupdates
Content-Type: application/json
AuthorizationType: ilink_bot_token
Authorization: Bearer {token}
X-WECHAT-UIN: {random_uin}

{
  "get_updates_buf": "string",  // 游标，首次为空
  "base_info": {
    "channel_version": "0.1.0"
  }
}
```

**响应：**
```json
{
  "ret": 0,
  "msgs": [
    {
      "seq": 123456789,
      "message_id": 987654321,
      "from_user_id": "wxid_xxx",
      "to_user_id": "wxid_bot",
      "client_id": "uuid",
      "create_time_ms": 1714392000000,
      "message_type": 1,
      "message_state": 2,
      "context_token": "token_string",
      "item_list": [
        {
          "type": 1,
          "text_item": {
            "text": "Hello"
          }
        }
      ]
    }
  ],
  "get_updates_buf": "next_cursor"
}
```

### WireMessage 数据结构

```dart
class WireMessage {
  final int? seq;
  final int? messageId;
  final String fromUserId;
  final String toUserId;
  final String clientId;
  final int createTimeMs;
  final MessageType messageType;
  final MessageState messageState;
  final String contextToken;
  final List<MessageItem> itemList;

  WireMessage({
    this.seq,
    this.messageId,
    required this.fromUserId,
    required this.toUserId,
    required this.clientId,
    required this.createTimeMs,
    required this.messageType,
    required this.messageState,
    required this.contextToken,
    required this.itemList,
  });
}

class MessageItem {
  final MessageItemType type;
  final TextItem? textItem;
  final ImageItem? imageItem;
  final VoiceItem? voiceItem;
  final FileItem? fileItem;
  final VideoItem? videoItem;
  final RefMessage? refMsg;

  MessageItem({
    required this.type,
    this.textItem,
    this.imageItem,
    this.voiceItem,
    this.fileItem,
    this.videoItem,
    this.refMsg,
  });
}
```

### 发送消息（sendmessage）

**文本消息请求：**
```http
POST /ilink/bot/sendmessage
Authorization: Bearer {token}

{
  "msg": {
    "from_user_id": "",
    "to_user_id": "wxid_xxx",
    "client_id": "uuid",
    "message_type": 2,
    "message_state": 2,
    "context_token": "token_string",
    "item_list": [
      {
        "type": 1,
        "text_item": {
          "text": "Hello back!"
        }
      }
    ]
  },
  "base_info": {
    "channel_version": "0.1.0"
  }
}
```

**媒体消息请求：**
```json
{
  "msg": {
    "from_user_id": "",
    "to_user_id": "wxid_xxx",
    "client_id": "uuid",
    "message_type": 2,
    "message_state": 2,
    "context_token": "token_string",
    "item_list": [
      {
        "type": 2,
        "image_item": {
          "media": {
            "encrypt_query_param": "...",
            "aes_key": "...",
            "encrypt_type": 1
          },
          "mid_size": 12345
        }
      }
    ]
  },
  "base_info": {
    "channel_version": "0.1.0"
  }
}
```

### 输入状态（sendtyping）

**获取配置：**
```http
POST /ilink/bot/getconfig

{
  "ilink_user_id": "wxid_xxx",
  "context_token": "token_string",
  "base_info": {"channel_version": "0.1.0"}
}
```

**响应：**
```json
{
  "typing_ticket": "ticket_string"
}
```

**发送输入状态：**
```http
POST /ilink/bot/sendtyping

{
  "ilink_user_id": "wxid_xxx",
  "typing_ticket": "ticket_string",
  "status": 1,  // 1=开始输入, 2=停止输入
  "base_info": {"channel_version": "0.1.0"}
}
```

---

## 媒体处理（CDN上传下载）

### CDN媒体引用结构

```dart
class CDNMedia {
  final String encryptQueryParam;
  final String aesKey;
  final int? encryptType;
  final String? fullUrl;

  CDNMedia({
    required this.encryptQueryParam,
    required this.aesKey,
    this.encryptType,
    this.fullUrl,
  });
}
```

### 上传流程

```
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐
│  准备文件    │───▶│ 生成AES密钥并加密 │───▶│ 获取上传URL      │───▶│ 上传到CDN    │
│             │    │                 │    │                 │    │             │
└─────────────┘    └─────────────────┘    └─────────────────┘    └──────┬──────┘
                                                                         │
                                                                         ▼
                                                              ┌─────────────────┐
                                                              │ 获取加密查询参数  │
                                                              │ x-encrypted-param│
                                                              └─────────────────┘
```

### 1. 获取上传URL

**请求：**
```http
POST /ilink/bot/getuploadurl

{
  "filekey": "hex_string",
  "media_type": 1,           // 1=image, 2=video, 3=file, 4=voice
  "to_user_id": "wxid_xxx",
  "rawsize": 10240,
  "rawfilemd5": "md5_hex",
  "filesize": 10240,         // 加密后大小
  "no_need_thumb": true,
  "aeskey": "hex_string",
  "base_info": {"channel_version": "0.1.0"}
}
```

**响应：**
```json
{
  "upload_param": "string",
  "thumb_upload_param": "string",
  "upload_full_url": "string"
}
```

### 2. 上传到CDN

**请求：**
```http
POST https://novac2c.cdn.weixin.qq.com/c2c/upload?encrypted_query_param={param}&filekey={filekey}
Content-Type: application/octet-stream

[加密后的文件字节]
```

**响应Header：**
```
x-encrypted-param: {download_param}
```

### 3. 从CDN下载

**请求：**
```http
GET https://novac2c.cdn.weixin.qq.com/c2c/download?encrypted_query_param={param}
```

**响应：**
```
[加密后的文件字节]
```

**解密流程：**
```dart
// 1. 下载加密数据
// 2. 使用AES-128-ECB解密
// 3. 去除PKCS7填充
```

---

## 加密算法

### AES-128-ECB 加解密

```dart
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';
import 'package:pointycastle/pointycastle.dart';

class AESCrypto {
  /// AES-128-ECB 加密（带PKCS7填充）
  static Uint8List encrypt(Uint8List plaintext, Uint8List key) {
    if (key.length != 16) {
      throw ArgumentError('AES key must be 16 bytes, got ${key.length}');
    }
    
    final padded = _pkcs7Pad(plaintext, 16);
    final cipher = PaddedBlockCipher('AES/ECB/PKCS7');
    
    cipher.init(
      true,
      PaddedBlockCipherParameters(
        KeyParameter(key),
        null,
      ),
    );
    
    return cipher.process(padded);
  }
  
  /// AES-128-ECB 解密（去除PKCS7填充）
  static Uint8List decrypt(Uint8List ciphertext, Uint8List key) {
    if (key.length != 16) {
      throw ArgumentError('AES key must be 16 bytes, got ${key.length}');
    }
    if (ciphertext.length % 16 != 0) {
      throw ArgumentError('Ciphertext length must be multiple of 16');
    }
    
    final cipher = PaddedBlockCipher('AES/ECB/PKCS7');
    
    cipher.init(
      false,
      PaddedBlockCipherParameters(
        KeyParameter(key),
        null,
      ),
    );
    
    return cipher.process(ciphertext);
  }
  
  /// 生成随机16字节AES密钥
  static Uint8List generateKey() {
    final random = Random.secure();
    return Uint8List.fromList(
      List.generate(16, (_) => random.nextInt(256)),
    );
  }
  
  /// PKCS7填充
  static Uint8List _pkcs7Pad(Uint8List data, int blockSize) {
    final padding = blockSize - (data.length % blockSize);
    final result = Uint8List(data.length + padding);
    result.setAll(0, data);
    result.fillRange(data.length, result.length, padding);
    return result;
  }
}
```

### AES Key 编解码

```dart
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
      decoded = base64Decode(encoded.replaceAll('-', '+').replaceAll('_', '/'));
    }
    
    if (decoded.length == 16) {
      return decoded;
    }
    
    // base64编码的hex字符串
    if (decoded.length == 32 && hexPattern.hasMatch(utf8.decode(decoded))) {
      return _hexDecode(utf8.decode(decoded));
    }
    
    throw ArgumentError('Invalid AES key format');
  }
  
  /// 编码为hex字符串（用于getuploadurl）
  static String encodeHex(Uint8List key) {
    return _hexEncode(key);
  }
  
  /// 编码为base64(hex)（用于CDNMedia.aes_key）
  static String encodeBase64(Uint8List key) {
    return base64Encode(utf8.encode(_hexEncode(key)));
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
```

---

## Flutter SDK 架构建议

### 核心类设计

```dart
/// Bot客户端主类
class WeChatBot {
  final WeChatOptions options;
  final _protocol = ProtocolClient();
  final _handlers = <MessageHandler>[];
  final _contextTokens = <String, String>{};
  Credentials? _credentials;
  String? _cursor;
  bool _stopped = false;
  
  WeChatBot({this.options = const WeChatOptions()});
  
  /// 登录
  Future<Credentials> login({bool force = false});
  
  /// 注册消息处理器
  void onMessage(MessageHandler handler);
  
  /// 回复消息
  Future<void> reply(IncomingMessage msg, String text);
  
  /// 发送消息
  Future<void> send(String userId, String text);
  
  /// 发送输入状态
  Future<void> sendTyping(String userId);
  
  /// 停止输入状态
  Future<void> stopTyping(String userId);
  
  /// 回复多媒体内容
  Future<void> replyContent(IncomingMessage msg, SendContent content);
  
  /// 下载媒体
  Future<DownloadedMedia?> download(IncomingMessage msg);
  
  /// 启动消息轮询
  Future<void> run();
  
  /// 停止消息轮询
  void stop();
}

/// 配置选项
class WeChatOptions {
  final String? baseUrl;
  final String? credPath;
  final LogLevel logLevel;
  final void Function(String url)? onQrUrl;
  final void Function()? onScanned;
  final void Function()? onExpired;
  final void Function(Object error)? onError;
  
  const WeChatOptions({
    this.baseUrl,
    this.credPath,
    this.logLevel = LogLevel.info,
    this.onQrUrl,
    this.onScanned,
    this.onExpired,
    this.onError,
  });
}

/// 消息处理器类型
typedef MessageHandler = void Function(IncomingMessage msg);
```

### 依赖建议

```yaml
# pubspec.yaml
dependencies:
  http: ^1.2.0
  pointycastle: ^3.7.4
  uuid: ^4.3.3
  path_provider: ^2.1.2
  
dev_dependencies:
  flutter_test:
    sdk: flutter
  mockito: ^5.4.4
```

### 使用示例

```dart
import 'package:wechatbot/wechatbot.dart';

void main() async {
  final bot = WeChatBot(
    options: WeChatOptions(
      onQrUrl: (url) => print('请扫码: $url'),
      onError: (e) => print('错误: $e'),
    ),
  );
  
  // 登录
  final creds = await bot.login();
  print('已登录: ${creds.accountId}');
  
  // 注册消息处理器
  bot.onMessage((msg) async {
    print('收到消息: ${msg.text}');
    
    // 显示输入状态
    await bot.sendTyping(msg.userId);
    
    // 回复
    await bot.reply(msg, 'Echo: ${msg.text}');
  });
  
  // 启动轮询
  await bot.run();
}
```

---

## 错误处理

### API错误码

| 错误码 | 含义 | 处理 |
|--------|------|------|
| `-14` | Session过期 | 重新登录 |
| 其他 | 通用错误 | 根据错误消息处理 |

### 错误类设计

```dart
class APIError implements Exception {
  final String message;
  final int httpStatus;
  final int errCode;
  
  APIError({
    required this.message,
    required this.httpStatus,
    required this.errCode,
  });
  
  bool get isSessionExpired => errCode == -14;
  
  @override
  String toString() => 'APIError: $message (http=$httpStatus, errcode=$errCode)';
}
```

---

## 注意事项

1. **Context Token管理**：每个用户会话需要维护context_token，首次回复后SDK会自动缓存
2. **长轮询超时**：getupdates接口使用45秒超时，这是正常行为
3. **消息分片**：文本消息超过4000字符需要分片发送
4. **文件扩展名**：发送文件时根据扩展名自动分类（图片/视频/普通文件）
5. **CDN上传重试**：上传失败时自动重试最多3次

---

*文档基于Go SDK v0.1.0版本整理*
