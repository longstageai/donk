/// 微信 iLink Bot SDK - Flutter 版本
///
/// 基于微信 iLink 协议的 Bot API SDK
/// 支持二维码登录、消息收发、媒体文件处理等功能
///
/// 使用示例:
/// ```dart
/// import 'package:donk/common/wechatbot/wechatbot.dart';
///
/// void main() async {
///   // 创建 Bot 实例
///   final bot = WeChatBot(
///     options: WeChatOptions(
///       onQrUrl: (url) => print('请扫描二维码: $url'),
///       onScanned: () => print('已扫码，等待确认'),
///       onExpired: () => print('二维码已过期'),
///     ),
///   );
///
///   // 登录
///   await bot.login();
///
///   // 注册消息处理器
///   bot.onMessage((msg) async {
///     print('收到消息: ${msg.text}');
///     await bot.reply(msg, '收到: ${msg.text}');
///   });
///
///   // 启动消息轮询
///   await bot.run();
/// }
/// ```

library;

// 导出核心类型
export 'src/types.dart';

// 导出 Bot 客户端
export 'src/bot.dart';

// 导出协议客户端
export 'src/protocol/api.dart';
export 'src/protocol/headers.dart';

// 导出认证模块
export 'src/auth/login.dart';

// 导出加密模块
export 'src/crypto/aes.dart';
