import 'dart:convert';
import 'dart:typed_data';

/// 微信 iLink Bot SDK 核心类型定义

// ==================== 常量定义 ====================

class WeChatConstants {
  // API端点
  static const String defaultBaseURL = 'https://ilinkai.weixin.qq.com';
  static const String cdnBaseURL = 'https://novac2c.cdn.weixin.qq.com/c2c';

  // 版本信息
  static const String channelVersion = '0.1.0';
  static const String iLinkAppID = 'bot';
  static const String iLinkClientVer = '256'; // 0x00010000 for 0.1.0 = 256

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

// ==================== 枚举类型 ====================

/// 消息发送者类型
enum MessageType {
  user(1),
  bot(2);

  final int value;
  const MessageType(this.value);

  factory MessageType.fromInt(int value) {
    return MessageType.values.firstWhere(
      (e) => e.value == value,
      orElse: () => MessageType.user,
    );
  }
}

/// 消息状态
enum MessageState {
  new_(0),
  generating(1),
  finish(2);

  final int value;
  const MessageState(this.value);

  factory MessageState.fromInt(int value) {
    return MessageState.values.firstWhere(
      (e) => e.value == value,
      orElse: () => MessageState.new_,
    );
  }
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

  factory MessageItemType.fromInt(int value) {
    return MessageItemType.values.firstWhere(
      (e) => e.value == value,
      orElse: () => MessageItemType.text,
    );
  }
}

/// 媒体类型（用于上传）
enum MediaType {
  image(1),
  video(2),
  file(3),
  voice(4);

  final int value;
  const MediaType(this.value);

  factory MediaType.fromInt(int value) {
    return MediaType.values.firstWhere(
      (e) => e.value == value,
      orElse: () => MediaType.file,
    );
  }
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

/// 二维码状态
enum QRCodeStatus {
  wait('wait'),
  scaned('scaned'),
  confirmed('confirmed'),
  expired('expired'),
  scanedButRedirect('scaned_but_redirect');

  final String value;
  const QRCodeStatus(this.value);

  factory QRCodeStatus.fromString(String value) {
    return QRCodeStatus.values.firstWhere(
      (e) => e.value == value,
      orElse: () => QRCodeStatus.wait,
    );
  }
}

/// 日志级别
enum LogLevel {
  debug(0),
  info(1),
  warn(2),
  error(3);

  final int value;
  const LogLevel(this.value);
}

// ==================== 数据模型 ====================

/// 凭证信息
class Credentials {
  final String token;
  final String baseUrl;
  final String accountId;
  final String userId;
  final DateTime? savedAt;

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
    token: json['token'] as String,
    baseUrl: json['baseUrl'] as String,
    accountId: json['accountId'] as String,
    userId: json['userId'] as String,
    savedAt:
        json['savedAt'] != null
            ? DateTime.parse(json['savedAt'] as String)
            : null,
  );

  String toJsonString() => jsonEncode(toJson());

  factory Credentials.fromJsonString(String jsonString) {
    final json = jsonDecode(jsonString) as Map<String, dynamic>;
    return Credentials.fromJson(json);
  }
}

/// 二维码响应
class QRCodeResponse {
  final String qrcode;
  final String qrcodeImgContent;

  QRCodeResponse({required this.qrcode, required this.qrcodeImgContent});

  factory QRCodeResponse.fromJson(Map<String, dynamic> json) => QRCodeResponse(
    qrcode: json['qrcode'] as String,
    qrcodeImgContent: json['qrcode_img_content'] as String,
  );
}

/// 二维码状态响应
class QRCodeStatusResponse {
  final QRCodeStatus status;
  final String? botToken;
  final String? ilinkBotId;
  final String? ilinkUserId;
  final String? baseurl;
  final String? redirectHost;

  QRCodeStatusResponse({
    required this.status,
    this.botToken,
    this.ilinkBotId,
    this.ilinkUserId,
    this.baseurl,
    this.redirectHost,
  });

  factory QRCodeStatusResponse.fromJson(Map<String, dynamic> json) =>
      QRCodeStatusResponse(
        status: QRCodeStatus.fromString(json['status'] as String),
        botToken: json['bot_token'] as String?,
        ilinkBotId: json['ilink_bot_id'] as String?,
        ilinkUserId: json['ilink_user_id'] as String?,
        baseurl: json['baseurl'] as String?,
        redirectHost: json['redirect_host'] as String?,
      );

  bool get isConfirmed => status == QRCodeStatus.confirmed;
  bool get isExpired => status == QRCodeStatus.expired;
  bool get needRedirect => status == QRCodeStatus.scanedButRedirect;
}

/// 文本消息项
class TextItem {
  final String text;

  TextItem({required this.text});

  Map<String, dynamic> toJson() => {'text': text};

  factory TextItem.fromJson(Map<String, dynamic> json) =>
      TextItem(text: json['text'] as String);
}

/// 图片消息项
class ImageItem {
  final CDNMedia media;
  final int? midSize;

  ImageItem({required this.media, this.midSize});

  Map<String, dynamic> toJson() => {
    'media': media.toJson(),
    if (midSize != null) 'mid_size': midSize,
  };

  factory ImageItem.fromJson(Map<String, dynamic> json) => ImageItem(
    media: CDNMedia.fromJson(json['media'] as Map<String, dynamic>),
    midSize: json['mid_size'] as int?,
  );
}

/// 语音消息项
class VoiceItem {
  final CDNMedia media;
  final int? duration;
  final String? recognitionResult;

  VoiceItem({required this.media, this.duration, this.recognitionResult});

  Map<String, dynamic> toJson() => {
    'media': media.toJson(),
    if (duration != null) 'duration': duration,
    if (recognitionResult != null) 'recognition_result': recognitionResult,
  };

  factory VoiceItem.fromJson(Map<String, dynamic> json) => VoiceItem(
    media: CDNMedia.fromJson(json['media'] as Map<String, dynamic>),
    duration: json['duration'] as int?,
    recognitionResult:
        json['text'] as String? ?? json['recognition_result'] as String?,
  );
}

/// 文件消息项
class FileItem {
  final CDNMedia media;
  final String fileName;
  final int? fileSize;

  FileItem({required this.media, required this.fileName, this.fileSize});

  Map<String, dynamic> toJson() => {
    'media': media.toJson(),
    'file_name': fileName,
    if (fileSize != null) 'file_size': fileSize,
  };

  factory FileItem.fromJson(Map<String, dynamic> json) => FileItem(
    media: CDNMedia.fromJson(json['media'] as Map<String, dynamic>),
    fileName: json['file_name'] as String,
    fileSize: json['file_size'] as int?,
  );
}

/// 视频消息项
class VideoItem {
  final CDNMedia media;
  final int? duration;

  VideoItem({required this.media, this.duration});

  Map<String, dynamic> toJson() => {
    'media': media.toJson(),
    if (duration != null) 'duration': duration,
  };

  factory VideoItem.fromJson(Map<String, dynamic> json) => VideoItem(
    media: CDNMedia.fromJson(json['media'] as Map<String, dynamic>),
    duration: json['duration'] as int?,
  );
}

/// 引用消息
class RefMessage {
  final String fromUserId;
  final int messageId;
  final String clientId;

  RefMessage({
    required this.fromUserId,
    required this.messageId,
    required this.clientId,
  });

  Map<String, dynamic> toJson() => {
    'from_user_id': fromUserId,
    'message_id': messageId,
    'client_id': clientId,
  };

  factory RefMessage.fromJson(Map<String, dynamic> json) => RefMessage(
    fromUserId: json['from_user_id'] as String,
    messageId: json['message_id'] as int,
    clientId: json['client_id'] as String,
  );
}

/// 消息项
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

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{'type': type.value};
    switch (type) {
      case MessageItemType.text:
        if (textItem != null) json['text_item'] = textItem!.toJson();
        break;
      case MessageItemType.image:
        if (imageItem != null) json['image_item'] = imageItem!.toJson();
        break;
      case MessageItemType.voice:
        if (voiceItem != null) json['voice_item'] = voiceItem!.toJson();
        break;
      case MessageItemType.file:
        if (fileItem != null) json['file_item'] = fileItem!.toJson();
        break;
      case MessageItemType.video:
        if (videoItem != null) json['video_item'] = videoItem!.toJson();
        break;
    }
    if (refMsg != null) json['ref_msg'] = refMsg!.toJson();
    return json;
  }

  factory MessageItem.fromJson(Map<String, dynamic> json) => MessageItem(
    type: MessageItemType.fromInt(json['type'] as int),
    textItem:
        json['text_item'] != null
            ? TextItem.fromJson(json['text_item'] as Map<String, dynamic>)
            : null,
    imageItem:
        json['image_item'] != null
            ? ImageItem.fromJson(json['image_item'] as Map<String, dynamic>)
            : null,
    voiceItem:
        json['voice_item'] != null
            ? VoiceItem.fromJson(json['voice_item'] as Map<String, dynamic>)
            : null,
    fileItem:
        json['file_item'] != null
            ? FileItem.fromJson(json['file_item'] as Map<String, dynamic>)
            : null,
    videoItem:
        json['video_item'] != null
            ? VideoItem.fromJson(json['video_item'] as Map<String, dynamic>)
            : null,
    refMsg:
        json['ref_msg'] != null
            ? RefMessage.fromJson(json['ref_msg'] as Map<String, dynamic>)
            : null,
  );
}

/// CDN 媒体引用
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

  Map<String, dynamic> toJson() => {
    'encrypt_query_param': encryptQueryParam,
    'aes_key': aesKey,
    if (encryptType != null) 'encrypt_type': encryptType,
    if (fullUrl != null) 'full_url': fullUrl,
  };

  factory CDNMedia.fromJson(Map<String, dynamic> json) => CDNMedia(
    encryptQueryParam: json['encrypt_query_param'] as String,
    aesKey: json['aes_key'] as String,
    encryptType: json['encrypt_type'] as int?,
    fullUrl: json['full_url'] as String?,
  );
}

/// 原始消息（Wire格式）
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

  Map<String, dynamic> toJson() => {
    if (seq != null) 'seq': seq,
    if (messageId != null) 'message_id': messageId,
    'from_user_id': fromUserId,
    'to_user_id': toUserId,
    'client_id': clientId,
    'create_time_ms': createTimeMs,
    'message_type': messageType.value,
    'message_state': messageState.value,
    'context_token': contextToken,
    'item_list': itemList.map((e) => e.toJson()).toList(),
  };

  factory WireMessage.fromJson(Map<String, dynamic> json) => WireMessage(
    seq: json['seq'] as int?,
    messageId: json['message_id'] as int?,
    fromUserId: json['from_user_id'] as String,
    toUserId: json['to_user_id'] as String,
    clientId: json['client_id'] as String,
    createTimeMs: json['create_time_ms'] as int,
    messageType: MessageType.fromInt(json['message_type'] as int),
    messageState: MessageState.fromInt(json['message_state'] as int),
    contextToken: json['context_token'] as String,
    itemList:
        (json['item_list'] as List<dynamic>)
            .map((e) => MessageItem.fromJson(e as Map<String, dynamic>))
            .toList(),
  );
}

/// 接收到的消息（简化格式）
class IncomingMessage {
  final String id;
  final String fromUserId;
  final String toUserId;
  final DateTime timestamp;
  final ContentType contentType;
  final String? text;
  final CDNMedia? media;
  final String contextToken;
  final WireMessage rawMessage;

  IncomingMessage({
    required this.id,
    required this.fromUserId,
    required this.toUserId,
    required this.timestamp,
    required this.contentType,
    this.text,
    this.media,
    required this.contextToken,
    required this.rawMessage,
  });

  bool get isText => contentType == ContentType.text;
  bool get isImage => contentType == ContentType.image;
  bool get isVoice => contentType == ContentType.voice;
  bool get isFile => contentType == ContentType.file;
  bool get isVideo => contentType == ContentType.video;
}

/// 发送内容
abstract class SendContent {
  ContentType get contentType;
  Map<String, dynamic> toJson();
}

/// 文本发送内容
class TextContent implements SendContent {
  final String text;

  TextContent(this.text);

  @override
  ContentType get contentType => ContentType.text;

  @override
  Map<String, dynamic> toJson() => {'text': text};
}

/// 媒体发送内容
class MediaContent implements SendContent {
  @override
  final ContentType contentType;
  final CDNMedia media;
  final int? size;

  MediaContent({required this.contentType, required this.media, this.size});

  @override
  Map<String, dynamic> toJson() => {
    'media': media.toJson(),
    if (size != null) 'size': size,
  };
}

/// 下载的媒体
class DownloadedMedia {
  final ContentType contentType;
  final Uint8List data;
  final String? fileName;
  final int? duration;
  final String? recognitionResult;

  DownloadedMedia({
    required this.contentType,
    required this.data,
    this.fileName,
    this.duration,
    this.recognitionResult,
  });
}

/// 获取更新响应
class GetUpdatesResponse {
  final int ret;
  final List<WireMessage> msgs;
  final String? nextCursor;
  final int? errcode;
  final String? errmsg;

  GetUpdatesResponse({
    required this.ret,
    required this.msgs,
    this.nextCursor,
    this.errcode,
    this.errmsg,
  });

  factory GetUpdatesResponse.fromJson(Map<String, dynamic> json) =>
      GetUpdatesResponse(
        ret: json['ret'] as int? ?? 0,
        msgs:
            (json['msgs'] as List<dynamic>?)
                ?.map((e) => WireMessage.fromJson(e as Map<String, dynamic>))
                .toList() ??
            [],
        nextCursor: json['get_updates_buf'] as String?,
        errcode: json['errcode'] as int?,
        errmsg: json['errmsg'] as String?,
      );

  bool get isSuccess => ret == 0;
  bool get isSessionTimeout => errcode == -14;
}

/// 获取上传URL响应
class GetUploadURLResponse {
  final String uploadParam;
  final String? thumbUploadParam;
  final String? uploadFullUrl;

  GetUploadURLResponse({
    required this.uploadParam,
    this.thumbUploadParam,
    this.uploadFullUrl,
  });

  factory GetUploadURLResponse.fromJson(Map<String, dynamic> json) =>
      GetUploadURLResponse(
        uploadParam: json['upload_param'] as String,
        thumbUploadParam: json['thumb_upload_param'] as String?,
        uploadFullUrl: json['upload_full_url'] as String?,
      );
}

/// 获取配置响应
class GetConfigResponse {
  final String typingTicket;

  GetConfigResponse({required this.typingTicket});

  factory GetConfigResponse.fromJson(Map<String, dynamic> json) =>
      GetConfigResponse(typingTicket: json['typing_ticket'] as String);
}

/// 基础信息
class BaseInfo {
  final String channelVersion;

  BaseInfo({this.channelVersion = WeChatConstants.channelVersion});

  Map<String, dynamic> toJson() => {'channel_version': channelVersion};
}

/// SDK 配置选项
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

/// 消息处理器类型定义
typedef MessageHandler = Future<void> Function(IncomingMessage message);
