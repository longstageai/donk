import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'generated/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale) : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate = _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates = <LocalizationsDelegate<dynamic>>[
    delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
  ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('zh')
  ];

  /// No description provided for @appTitle.
  ///
  /// In zh, this message translates to:
  /// **'Donk'**
  String get appTitle;

  /// No description provided for @settings.
  ///
  /// In zh, this message translates to:
  /// **'设置'**
  String get settings;

  /// No description provided for @generalSettings.
  ///
  /// In zh, this message translates to:
  /// **'通用设置'**
  String get generalSettings;

  /// No description provided for @language.
  ///
  /// In zh, this message translates to:
  /// **'语言'**
  String get language;

  /// No description provided for @languageDesc.
  ///
  /// In zh, this message translates to:
  /// **'选择应用显示语言'**
  String get languageDesc;

  /// No description provided for @languageChinese.
  ///
  /// In zh, this message translates to:
  /// **'中文'**
  String get languageChinese;

  /// No description provided for @languageEnglish.
  ///
  /// In zh, this message translates to:
  /// **'English'**
  String get languageEnglish;

  /// No description provided for @securityProtection.
  ///
  /// In zh, this message translates to:
  /// **'安全防护'**
  String get securityProtection;

  /// No description provided for @securityProtectionDesc.
  ///
  /// In zh, this message translates to:
  /// **'开启后可实时保护AI安全，防范漏洞攻击，拦截恶意指令、技能投毒等风险行为'**
  String get securityProtectionDesc;

  /// No description provided for @knowledgeAutoBuild.
  ///
  /// In zh, this message translates to:
  /// **'知识库自动构建'**
  String get knowledgeAutoBuild;

  /// No description provided for @knowledgeAutoBuildDesc.
  ///
  /// In zh, this message translates to:
  /// **'自动扫描桌面、下载、文档文件夹，支持 txt、md、pdf、docx 格式，构建可搜索的向量知识库'**
  String get knowledgeAutoBuildDesc;

  /// No description provided for @sleepPrevention.
  ///
  /// In zh, this message translates to:
  /// **'休眠阻止'**
  String get sleepPrevention;

  /// No description provided for @sleepPreventionDesc.
  ///
  /// In zh, this message translates to:
  /// **'开启后，电脑将不会进入休眠模式，donk 会保持活跃状态'**
  String get sleepPreventionDesc;

  /// No description provided for @toolPermission.
  ///
  /// In zh, this message translates to:
  /// **'工具权限限制'**
  String get toolPermission;

  /// No description provided for @toolPermissionDesc.
  ///
  /// In zh, this message translates to:
  /// **'开启后，智能体调用工具时会限制按照低权限执行，防止误删关键文件'**
  String get toolPermissionDesc;

  /// No description provided for @close.
  ///
  /// In zh, this message translates to:
  /// **'关闭'**
  String get close;

  /// No description provided for @save.
  ///
  /// In zh, this message translates to:
  /// **'保存'**
  String get save;

  /// No description provided for @cancel.
  ///
  /// In zh, this message translates to:
  /// **'取消'**
  String get cancel;

  /// No description provided for @confirm.
  ///
  /// In zh, this message translates to:
  /// **'确认'**
  String get confirm;

  /// No description provided for @delete.
  ///
  /// In zh, this message translates to:
  /// **'删除'**
  String get delete;

  /// No description provided for @retry.
  ///
  /// In zh, this message translates to:
  /// **'重试'**
  String get retry;

  /// No description provided for @reload.
  ///
  /// In zh, this message translates to:
  /// **'重新加载'**
  String get reload;

  /// No description provided for @edit.
  ///
  /// In zh, this message translates to:
  /// **'编辑'**
  String get edit;

  /// No description provided for @add.
  ///
  /// In zh, this message translates to:
  /// **'添加'**
  String get add;

  /// No description provided for @remove.
  ///
  /// In zh, this message translates to:
  /// **'移除'**
  String get remove;

  /// No description provided for @open.
  ///
  /// In zh, this message translates to:
  /// **'打开'**
  String get open;

  /// No description provided for @copy.
  ///
  /// In zh, this message translates to:
  /// **'复制'**
  String get copy;

  /// No description provided for @search.
  ///
  /// In zh, this message translates to:
  /// **'搜索'**
  String get search;

  /// No description provided for @loading.
  ///
  /// In zh, this message translates to:
  /// **'加载中...'**
  String get loading;

  /// No description provided for @noData.
  ///
  /// In zh, this message translates to:
  /// **'暂无数据'**
  String get noData;

  /// No description provided for @error.
  ///
  /// In zh, this message translates to:
  /// **'错误'**
  String get error;

  /// No description provided for @success.
  ///
  /// In zh, this message translates to:
  /// **'成功'**
  String get success;

  /// No description provided for @failed.
  ///
  /// In zh, this message translates to:
  /// **'失败'**
  String get failed;

  /// No description provided for @operationFailed.
  ///
  /// In zh, this message translates to:
  /// **'操作失败'**
  String get operationFailed;

  /// No description provided for @operationSuccess.
  ///
  /// In zh, this message translates to:
  /// **'操作成功'**
  String get operationSuccess;

  /// No description provided for @confirmDelete.
  ///
  /// In zh, this message translates to:
  /// **'确认删除'**
  String get confirmDelete;

  /// No description provided for @confirmRemove.
  ///
  /// In zh, this message translates to:
  /// **'确认移除'**
  String get confirmRemove;

  /// No description provided for @deleteConfirmMessage.
  ///
  /// In zh, this message translates to:
  /// **'确定要删除吗？此操作不可恢复。'**
  String get deleteConfirmMessage;

  /// No description provided for @removeConfirmMessage.
  ///
  /// In zh, this message translates to:
  /// **'确定要移除吗？此操作不可恢复。'**
  String get removeConfirmMessage;

  /// No description provided for @taskManagement.
  ///
  /// In zh, this message translates to:
  /// **'任务管理'**
  String get taskManagement;

  /// No description provided for @executeConfirmTitle.
  ///
  /// In zh, this message translates to:
  /// **'立即执行'**
  String get executeConfirmTitle;

  /// No description provided for @executeConfirmMessage.
  ///
  /// In zh, this message translates to:
  /// **'确定要立即执行任务吗？'**
  String get executeConfirmMessage;

  /// No description provided for @executeConfirmMessageTask.
  ///
  /// In zh, this message translates to:
  /// **'确定要立即执行任务\"{name}\"吗？'**
  String executeConfirmMessageTask(Object name);

  /// No description provided for @taskExecuted.
  ///
  /// In zh, this message translates to:
  /// **'任务\"{name}\"已开始执行'**
  String taskExecuted(Object name);

  /// No description provided for @executeFailed.
  ///
  /// In zh, this message translates to:
  /// **'执行失败'**
  String get executeFailed;

  /// No description provided for @removeTask.
  ///
  /// In zh, this message translates to:
  /// **'移除任务'**
  String get removeTask;

  /// No description provided for @removeTaskConfirmMessage.
  ///
  /// In zh, this message translates to:
  /// **'确定要移除任务\"{name}\"吗？此操作不可恢复。'**
  String removeTaskConfirmMessage(Object name);

  /// No description provided for @removeFailed.
  ///
  /// In zh, this message translates to:
  /// **'移除失败'**
  String get removeFailed;

  /// No description provided for @runRecords.
  ///
  /// In zh, this message translates to:
  /// **'运行记录'**
  String get runRecords;

  /// No description provided for @deleteRunConfirmMessage.
  ///
  /// In zh, this message translates to:
  /// **'确定要删除这条运行记录吗？\n\n执行ID: {id}\n执行时间: {time}'**
  String deleteRunConfirmMessage(Object id, Object time);

  /// No description provided for @statusDone.
  ///
  /// In zh, this message translates to:
  /// **'已完成'**
  String get statusDone;

  /// No description provided for @statusFailed.
  ///
  /// In zh, this message translates to:
  /// **'失败'**
  String get statusFailed;

  /// No description provided for @statusRunning.
  ///
  /// In zh, this message translates to:
  /// **'执行中'**
  String get statusRunning;

  /// No description provided for @taskStarted.
  ///
  /// In zh, this message translates to:
  /// **'任务已开始执行'**
  String get taskStarted;

  /// No description provided for @taskRemoved.
  ///
  /// In zh, this message translates to:
  /// **'任务已移除'**
  String get taskRemoved;

  /// No description provided for @runRecordDeleted.
  ///
  /// In zh, this message translates to:
  /// **'运行记录已删除'**
  String get runRecordDeleted;

  /// No description provided for @llmSettings.
  ///
  /// In zh, this message translates to:
  /// **'LLM 设置'**
  String get llmSettings;

  /// No description provided for @agentSettings.
  ///
  /// In zh, this message translates to:
  /// **'Agent 设置'**
  String get agentSettings;

  /// No description provided for @agentDetails.
  ///
  /// In zh, this message translates to:
  /// **'Agent 详情'**
  String get agentDetails;

  /// No description provided for @defaultAgentHint.
  ///
  /// In zh, this message translates to:
  /// **'官方默认Agent'**
  String get defaultAgentHint;

  /// No description provided for @idea.
  ///
  /// In zh, this message translates to:
  /// **'灵感'**
  String get idea;

  /// No description provided for @chat.
  ///
  /// In zh, this message translates to:
  /// **'对话'**
  String get chat;

  /// No description provided for @task.
  ///
  /// In zh, this message translates to:
  /// **'任务'**
  String get task;

  /// No description provided for @embeddingSettings.
  ///
  /// In zh, this message translates to:
  /// **'Embedding 设置'**
  String get embeddingSettings;

  /// No description provided for @about.
  ///
  /// In zh, this message translates to:
  /// **'关于我们'**
  String get about;

  /// No description provided for @currentVersion.
  ///
  /// In zh, this message translates to:
  /// **'当前版本'**
  String get currentVersion;

  /// No description provided for @updateAvailable.
  ///
  /// In zh, this message translates to:
  /// **'可更新'**
  String get updateAvailable;

  /// No description provided for @officialWebsite.
  ///
  /// In zh, this message translates to:
  /// **'进入官网'**
  String get officialWebsite;

  /// No description provided for @serviceAgreement.
  ///
  /// In zh, this message translates to:
  /// **'服务协议'**
  String get serviceAgreement;

  /// No description provided for @privacyPolicy.
  ///
  /// In zh, this message translates to:
  /// **'隐私保护协议'**
  String get privacyPolicy;

  /// No description provided for @notifications.
  ///
  /// In zh, this message translates to:
  /// **'消息通知'**
  String get notifications;

  /// No description provided for @markAllRead.
  ///
  /// In zh, this message translates to:
  /// **'全部已读'**
  String get markAllRead;

  /// No description provided for @clearAll.
  ///
  /// In zh, this message translates to:
  /// **'清空全部'**
  String get clearAll;

  /// No description provided for @clearConfirmTitle.
  ///
  /// In zh, this message translates to:
  /// **'确认清空'**
  String get clearConfirmTitle;

  /// No description provided for @clearConfirmMessage.
  ///
  /// In zh, this message translates to:
  /// **'确定要清空所有消息吗？此操作不可恢复。'**
  String get clearConfirmMessage;

  /// No description provided for @justNow.
  ///
  /// In zh, this message translates to:
  /// **'刚刚'**
  String get justNow;

  /// No description provided for @minutesAgo.
  ///
  /// In zh, this message translates to:
  /// **'{count}分钟前'**
  String minutesAgo(Object count);

  /// No description provided for @hoursAgo.
  ///
  /// In zh, this message translates to:
  /// **'{count}小时前'**
  String hoursAgo(Object count);

  /// No description provided for @daysAgo.
  ///
  /// In zh, this message translates to:
  /// **'{count}天前'**
  String daysAgo(Object count);

  /// No description provided for @noMessages.
  ///
  /// In zh, this message translates to:
  /// **'暂无消息'**
  String get noMessages;

  /// No description provided for @websocketDisconnected.
  ///
  /// In zh, this message translates to:
  /// **'WebSocket未连接'**
  String get websocketDisconnected;

  /// No description provided for @reconnect.
  ///
  /// In zh, this message translates to:
  /// **'重新连接'**
  String get reconnect;

  /// No description provided for @tokenStats.
  ///
  /// In zh, this message translates to:
  /// **'用量统计'**
  String get tokenStats;

  /// No description provided for @tokenStatsDesc.
  ///
  /// In zh, this message translates to:
  /// **'仅统计默认大模型的用量数据；不包含自定义模型数据'**
  String get tokenStatsDesc;

  /// No description provided for @tokenUsageDetails.
  ///
  /// In zh, this message translates to:
  /// **'Token使用详情'**
  String get tokenUsageDetails;

  /// No description provided for @provider.
  ///
  /// In zh, this message translates to:
  /// **'提供商'**
  String get provider;

  /// No description provided for @modelName.
  ///
  /// In zh, this message translates to:
  /// **'模型名称'**
  String get modelName;

  /// No description provided for @modelNameHint.
  ///
  /// In zh, this message translates to:
  /// **'如: gpt-4o-mini'**
  String get modelNameHint;

  /// No description provided for @apiKey.
  ///
  /// In zh, this message translates to:
  /// **'API Key'**
  String get apiKey;

  /// No description provided for @apiKeyHint.
  ///
  /// In zh, this message translates to:
  /// **'输入 API Key'**
  String get apiKeyHint;

  /// No description provided for @baseUrl.
  ///
  /// In zh, this message translates to:
  /// **'Base URL'**
  String get baseUrl;

  /// No description provided for @baseUrlHint.
  ///
  /// In zh, this message translates to:
  /// **'可选，留空使用默认地址'**
  String get baseUrlHint;

  /// No description provided for @dimension.
  ///
  /// In zh, this message translates to:
  /// **'向量维度'**
  String get dimension;

  /// No description provided for @dimensionHint.
  ///
  /// In zh, this message translates to:
  /// **'如: 1536'**
  String get dimensionHint;

  /// No description provided for @temperature.
  ///
  /// In zh, this message translates to:
  /// **'温度'**
  String get temperature;

  /// No description provided for @maxTokens.
  ///
  /// In zh, this message translates to:
  /// **'最大 Token 数'**
  String get maxTokens;

  /// No description provided for @maxLoop.
  ///
  /// In zh, this message translates to:
  /// **'最大循环次数'**
  String get maxLoop;

  /// No description provided for @maxLoopHint.
  ///
  /// In zh, this message translates to:
  /// **'如: 10'**
  String get maxLoopHint;

  /// No description provided for @convergeAfter.
  ///
  /// In zh, this message translates to:
  /// **'收敛终止数'**
  String get convergeAfter;

  /// No description provided for @convergeAfterHint.
  ///
  /// In zh, this message translates to:
  /// **'如: 3'**
  String get convergeAfterHint;

  /// No description provided for @timeout.
  ///
  /// In zh, this message translates to:
  /// **'超时时间（秒）'**
  String get timeout;

  /// No description provided for @timeoutHint.
  ///
  /// In zh, this message translates to:
  /// **'如: 300'**
  String get timeoutHint;

  /// No description provided for @dailyTokenLimit.
  ///
  /// In zh, this message translates to:
  /// **'每日 Token 限额'**
  String get dailyTokenLimit;

  /// No description provided for @dailyTokenLimitHint.
  ///
  /// In zh, this message translates to:
  /// **'-1 表示无限制'**
  String get dailyTokenLimitHint;

  /// No description provided for @todayTokenUsed.
  ///
  /// In zh, this message translates to:
  /// **'今日消耗 Token'**
  String get todayTokenUsed;

  /// No description provided for @todayTokenRemaining.
  ///
  /// In zh, this message translates to:
  /// **'今日剩余 Token'**
  String get todayTokenRemaining;

  /// No description provided for @remainingPercent.
  ///
  /// In zh, this message translates to:
  /// **'剩余百分比'**
  String get remainingPercent;

  /// No description provided for @unlimited.
  ///
  /// In zh, this message translates to:
  /// **'无限制'**
  String get unlimited;

  /// No description provided for @wechatConnect.
  ///
  /// In zh, this message translates to:
  /// **'微信连接'**
  String get wechatConnect;

  /// No description provided for @wechatScanQR.
  ///
  /// In zh, this message translates to:
  /// **'请使用微信扫描二维码'**
  String get wechatScanQR;

  /// No description provided for @wechatQRCodeHint.
  ///
  /// In zh, this message translates to:
  /// **'二维码将在一段时间后过期，请尽快扫描'**
  String get wechatQRCodeHint;

  /// No description provided for @wechatScanned.
  ///
  /// In zh, this message translates to:
  /// **'已扫码'**
  String get wechatScanned;

  /// No description provided for @wechatConfirmLogin.
  ///
  /// In zh, this message translates to:
  /// **'请在手机上确认登录'**
  String get wechatConfirmLogin;

  /// No description provided for @wechatConnected.
  ///
  /// In zh, this message translates to:
  /// **'微信已连接'**
  String get wechatConnected;

  /// No description provided for @wechatConnectedDesc.
  ///
  /// In zh, this message translates to:
  /// **'您可以接收和发送微信消息'**
  String get wechatConnectedDesc;

  /// No description provided for @wechatConnecting.
  ///
  /// In zh, this message translates to:
  /// **'正在连接微信'**
  String get wechatConnecting;

  /// No description provided for @wechatConnectFailed.
  ///
  /// In zh, this message translates to:
  /// **'连接失败'**
  String get wechatConnectFailed;

  /// No description provided for @unknownError.
  ///
  /// In zh, this message translates to:
  /// **'未知错误'**
  String get unknownError;

  /// No description provided for @wechatWaitScan.
  ///
  /// In zh, this message translates to:
  /// **'等待扫码登录'**
  String get wechatWaitScan;

  /// No description provided for @wechatScanSuccess.
  ///
  /// In zh, this message translates to:
  /// **'扫码成功'**
  String get wechatScanSuccess;

  /// No description provided for @wechatConnectError.
  ///
  /// In zh, this message translates to:
  /// **'连接出现错误'**
  String get wechatConnectError;

  /// No description provided for @wechatDisconnected.
  ///
  /// In zh, this message translates to:
  /// **'微信未连接'**
  String get wechatDisconnected;

  /// No description provided for @disconnect.
  ///
  /// In zh, this message translates to:
  /// **'断开连接'**
  String get disconnect;

  /// No description provided for @description.
  ///
  /// In zh, this message translates to:
  /// **'描述'**
  String get description;

  /// No description provided for @skillFile.
  ///
  /// In zh, this message translates to:
  /// **'Skill 文件'**
  String get skillFile;

  /// No description provided for @callSettings.
  ///
  /// In zh, this message translates to:
  /// **'调用设置'**
  String get callSettings;

  /// No description provided for @tags.
  ///
  /// In zh, this message translates to:
  /// **'标签'**
  String get tags;

  /// No description provided for @resourceDir.
  ///
  /// In zh, this message translates to:
  /// **'资源目录'**
  String get resourceDir;

  /// No description provided for @rescan.
  ///
  /// In zh, this message translates to:
  /// **'重新扫描'**
  String get rescan;

  /// No description provided for @enableAll.
  ///
  /// In zh, this message translates to:
  /// **'全部启用'**
  String get enableAll;

  /// No description provided for @disableAll.
  ///
  /// In zh, this message translates to:
  /// **'全部禁用'**
  String get disableAll;

  /// No description provided for @refreshList.
  ///
  /// In zh, this message translates to:
  /// **'刷新列表'**
  String get refreshList;

  /// No description provided for @refresh.
  ///
  /// In zh, this message translates to:
  /// **'刷新'**
  String get refresh;

  /// No description provided for @clearMessages.
  ///
  /// In zh, this message translates to:
  /// **'清空消息'**
  String get clearMessages;

  /// No description provided for @tokenUsageStatus.
  ///
  /// In zh, this message translates to:
  /// **'已使用{used}，剩余{percent}%'**
  String tokenUsageStatus(Object percent, Object used);

  /// No description provided for @noSkill.
  ///
  /// In zh, this message translates to:
  /// **'暂无 Skill'**
  String get noSkill;

  /// No description provided for @clickScanToDiscover.
  ///
  /// In zh, this message translates to:
  /// **'点击右上角扫描按钮发现 Skill'**
  String get clickScanToDiscover;

  /// No description provided for @ideaSquare.
  ///
  /// In zh, this message translates to:
  /// **'灵感广场'**
  String get ideaSquare;

  /// No description provided for @loadingFailed.
  ///
  /// In zh, this message translates to:
  /// **'加载失败'**
  String get loadingFailed;

  /// No description provided for @scanFailed.
  ///
  /// In zh, this message translates to:
  /// **'扫描失败'**
  String get scanFailed;

  /// No description provided for @deleteFailed.
  ///
  /// In zh, this message translates to:
  /// **'删除失败'**
  String get deleteFailed;

  /// No description provided for @deleteConfirmMessageSkill.
  ///
  /// In zh, this message translates to:
  /// **'确定要删除 Skill \"{name}\" 吗？\n\n此操作不可恢复！'**
  String deleteConfirmMessageSkill(Object name);

  /// No description provided for @allEnabled.
  ///
  /// In zh, this message translates to:
  /// **'所有 Skill 已处于启用状态'**
  String get allEnabled;

  /// No description provided for @allDisabled.
  ///
  /// In zh, this message translates to:
  /// **'所有 Skill 已处于禁用状态'**
  String get allDisabled;

  /// No description provided for @enabledCount.
  ///
  /// In zh, this message translates to:
  /// **'已启用 {count} 个 Skill'**
  String enabledCount(Object count);

  /// No description provided for @disabledCount.
  ///
  /// In zh, this message translates to:
  /// **'已禁用 {count} 个 Skill'**
  String disabledCount(Object count);

  /// No description provided for @skillDeleted.
  ///
  /// In zh, this message translates to:
  /// **'Skill \"{name}\" 已删除'**
  String skillDeleted(Object name);

  /// No description provided for @skillStarted.
  ///
  /// In zh, this message translates to:
  /// **'Skill 已启动'**
  String get skillStarted;

  /// No description provided for @skillStopped.
  ///
  /// In zh, this message translates to:
  /// **'Skill 已停止'**
  String get skillStopped;

  /// No description provided for @fileNotFound.
  ///
  /// In zh, this message translates to:
  /// **'文件不存在'**
  String get fileNotFound;

  /// No description provided for @openFileFailed.
  ///
  /// In zh, this message translates to:
  /// **'打开文件失败'**
  String get openFileFailed;

  /// No description provided for @filePathCopied.
  ///
  /// In zh, this message translates to:
  /// **'文件路径已复制'**
  String get filePathCopied;

  /// No description provided for @copyFilePath.
  ///
  /// In zh, this message translates to:
  /// **'复制文件路径'**
  String get copyFilePath;

  /// No description provided for @invocationSettings.
  ///
  /// In zh, this message translates to:
  /// **'调用设置'**
  String get invocationSettings;

  /// No description provided for @userInvocable.
  ///
  /// In zh, this message translates to:
  /// **'用户可调用'**
  String get userInvocable;

  /// No description provided for @userInvocableDesc.
  ///
  /// In zh, this message translates to:
  /// **'允许通过斜杠命令调用'**
  String get userInvocableDesc;

  /// No description provided for @userNotInvocableDesc.
  ///
  /// In zh, this message translates to:
  /// **'不允许用户直接调用'**
  String get userNotInvocableDesc;

  /// No description provided for @autoTrigger.
  ///
  /// In zh, this message translates to:
  /// **'自动触发'**
  String get autoTrigger;

  /// No description provided for @autoTriggerEnabled.
  ///
  /// In zh, this message translates to:
  /// **'允许模型自动触发'**
  String get autoTriggerEnabled;

  /// No description provided for @autoTriggerDisabled.
  ///
  /// In zh, this message translates to:
  /// **'已禁用模型自动触发'**
  String get autoTriggerDisabled;

  /// No description provided for @scripts.
  ///
  /// In zh, this message translates to:
  /// **'脚本'**
  String get scripts;

  /// No description provided for @references.
  ///
  /// In zh, this message translates to:
  /// **'参考文档'**
  String get references;

  /// No description provided for @assets.
  ///
  /// In zh, this message translates to:
  /// **'资源文件'**
  String get assets;

  /// No description provided for @exists.
  ///
  /// In zh, this message translates to:
  /// **'存在'**
  String get exists;

  /// No description provided for @notExists.
  ///
  /// In zh, this message translates to:
  /// **'无'**
  String get notExists;

  /// No description provided for @unknownAuthor.
  ///
  /// In zh, this message translates to:
  /// **'未知作者'**
  String get unknownAuthor;

  /// No description provided for @start.
  ///
  /// In zh, this message translates to:
  /// **'启动'**
  String get start;

  /// No description provided for @stop.
  ///
  /// In zh, this message translates to:
  /// **'停止'**
  String get stop;

  /// No description provided for @scanComplete.
  ///
  /// In zh, this message translates to:
  /// **'扫描完成'**
  String get scanComplete;

  /// No description provided for @welcomeTitle.
  ///
  /// In zh, this message translates to:
  /// **'Hi，我是Donk'**
  String get welcomeTitle;

  /// No description provided for @welcomeSubtitle.
  ///
  /// In zh, this message translates to:
  /// **'随时随地，帮您高效干活'**
  String get welcomeSubtitle;

  /// No description provided for @installFirstSkill.
  ///
  /// In zh, this message translates to:
  /// **'安装你的第一个 Skill'**
  String get installFirstSkill;

  /// No description provided for @installFirstSkillDesc.
  ///
  /// In zh, this message translates to:
  /// **'一键教你安装超能力'**
  String get installFirstSkillDesc;

  /// No description provided for @emailManagement.
  ///
  /// In zh, this message translates to:
  /// **'邮件管理'**
  String get emailManagement;

  /// No description provided for @emailManagementDesc.
  ///
  /// In zh, this message translates to:
  /// **'帮你高效处理邮件'**
  String get emailManagementDesc;

  /// No description provided for @organizeDesktop.
  ///
  /// In zh, this message translates to:
  /// **'整理桌面'**
  String get organizeDesktop;

  /// No description provided for @organizeDesktopDesc.
  ///
  /// In zh, this message translates to:
  /// **'还你清爽电脑桌面'**
  String get organizeDesktopDesc;

  /// No description provided for @scheduleManagement.
  ///
  /// In zh, this message translates to:
  /// **'安排日程'**
  String get scheduleManagement;

  /// No description provided for @scheduleManagementDesc.
  ///
  /// In zh, this message translates to:
  /// **'一句话约日程定会议'**
  String get scheduleManagementDesc;

  /// No description provided for @remoteWork.
  ///
  /// In zh, this message translates to:
  /// **'手机远程办公'**
  String get remoteWork;

  /// No description provided for @remoteWorkDesc.
  ///
  /// In zh, this message translates to:
  /// **'随时处理在线任务'**
  String get remoteWorkDesc;

  /// No description provided for @scheduleTaskManagement.
  ///
  /// In zh, this message translates to:
  /// **'日程任务全管理'**
  String get scheduleTaskManagement;

  /// No description provided for @noRunRecords.
  ///
  /// In zh, this message translates to:
  /// **'暂无运行记录'**
  String get noRunRecords;

  /// No description provided for @fileTypeNotSupported.
  ///
  /// In zh, this message translates to:
  /// **'仅支持 pdf、docx、txt、md 文件'**
  String get fileTypeNotSupported;

  /// No description provided for @selectFileFailed.
  ///
  /// In zh, this message translates to:
  /// **'选择文件失败'**
  String get selectFileFailed;

  /// No description provided for @agentCollaboration.
  ///
  /// In zh, this message translates to:
  /// **'Donk 协作'**
  String get agentCollaboration;

  /// No description provided for @agentActivityStatus.
  ///
  /// In zh, this message translates to:
  /// **'{count} 条动态 · 实时同步'**
  String agentActivityStatus(Object count);

  /// No description provided for @realtimeConnected.
  ///
  /// In zh, this message translates to:
  /// **'实时连接已建立'**
  String get realtimeConnected;

  /// No description provided for @realtimeDisconnected.
  ///
  /// In zh, this message translates to:
  /// **'实时连接已断开'**
  String get realtimeDisconnected;

  /// No description provided for @noAgentMessages.
  ///
  /// In zh, this message translates to:
  /// **'暂无 Donk 消息'**
  String get noAgentMessages;

  /// No description provided for @agentActivityHint.
  ///
  /// In zh, this message translates to:
  /// **'Donk 协作动态会实时显示在这里'**
  String get agentActivityHint;

  /// No description provided for @latestMessage.
  ///
  /// In zh, this message translates to:
  /// **'最新消息'**
  String get latestMessage;

  /// No description provided for @copyContent.
  ///
  /// In zh, this message translates to:
  /// **'复制内容'**
  String get copyContent;

  /// No description provided for @contentCopied.
  ///
  /// In zh, this message translates to:
  /// **'内容已复制到剪贴板'**
  String get contentCopied;

  /// No description provided for @secondsAgo.
  ///
  /// In zh, this message translates to:
  /// **'{count}秒前'**
  String secondsAgo(Object count);

  /// No description provided for @sessionStarted.
  ///
  /// In zh, this message translates to:
  /// **'会话已启动'**
  String get sessionStarted;

  /// No description provided for @sessionStartFailed.
  ///
  /// In zh, this message translates to:
  /// **'会话启动失败：{error}'**
  String sessionStartFailed(Object error);

  /// No description provided for @sessionStopped.
  ///
  /// In zh, this message translates to:
  /// **'会话已停止'**
  String get sessionStopped;

  /// No description provided for @sessionStopFailed.
  ///
  /// In zh, this message translates to:
  /// **'会话停止失败：{error}'**
  String sessionStopFailed(Object error);

  /// No description provided for @clearAgentMessagesConfirm.
  ///
  /// In zh, this message translates to:
  /// **'确定清空所有 Donk 协作消息吗？清空后无法恢复。'**
  String get clearAgentMessagesConfirm;

  /// No description provided for @onboardingWindowTitle.
  ///
  /// In zh, this message translates to:
  /// **'Donk 初始化配置'**
  String get onboardingWindowTitle;

  /// No description provided for @minimize.
  ///
  /// In zh, this message translates to:
  /// **'最小化'**
  String get minimize;

  /// No description provided for @maximize.
  ///
  /// In zh, this message translates to:
  /// **'最大化'**
  String get maximize;

  /// No description provided for @restore.
  ///
  /// In zh, this message translates to:
  /// **'还原'**
  String get restore;

  /// No description provided for @previousStep.
  ///
  /// In zh, this message translates to:
  /// **'上一步'**
  String get previousStep;

  /// No description provided for @nextStep.
  ///
  /// In zh, this message translates to:
  /// **'下一步'**
  String get nextStep;

  /// No description provided for @configureLLM.
  ///
  /// In zh, this message translates to:
  /// **'配置 LLM'**
  String get configureLLM;

  /// No description provided for @configureLLMDesc.
  ///
  /// In zh, this message translates to:
  /// **'选择模型厂商并填写必要连接信息'**
  String get configureLLMDesc;

  /// No description provided for @modelConnectionInfo.
  ///
  /// In zh, this message translates to:
  /// **'模型连接信息'**
  String get modelConnectionInfo;

  /// No description provided for @llmProviderDesc.
  ///
  /// In zh, this message translates to:
  /// **'选择后会自动填充默认模型和完整 Base URL'**
  String get llmProviderDesc;

  /// No description provided for @apiKeySaveDesc.
  ///
  /// In zh, this message translates to:
  /// **'密钥仅用于服务端配置保存'**
  String get apiKeySaveDesc;

  /// No description provided for @baseUrlDefaultDesc.
  ///
  /// In zh, this message translates to:
  /// **'已按厂商默认填充，可按需修改'**
  String get baseUrlDefaultDesc;

  /// No description provided for @customApiUrlHint.
  ///
  /// In zh, this message translates to:
  /// **'自定义 API 地址（可选）'**
  String get customApiUrlHint;

  /// No description provided for @requiredFieldsComplete.
  ///
  /// In zh, this message translates to:
  /// **'必填项已完成，可以进入下一步'**
  String get requiredFieldsComplete;

  /// No description provided for @llmRequiredFieldsHint.
  ///
  /// In zh, this message translates to:
  /// **'填写提供商、模型名称和 API Key 后可继续'**
  String get llmRequiredFieldsHint;

  /// No description provided for @llmConfigSaved.
  ///
  /// In zh, this message translates to:
  /// **'LLM 配置保存成功'**
  String get llmConfigSaved;

  /// No description provided for @saveFailed.
  ///
  /// In zh, this message translates to:
  /// **'保存失败: {error}'**
  String saveFailed(Object error);

  /// No description provided for @providerQwen.
  ///
  /// In zh, this message translates to:
  /// **'通义千问'**
  String get providerQwen;

  /// No description provided for @providerDoubao.
  ///
  /// In zh, this message translates to:
  /// **'豆包'**
  String get providerDoubao;

  /// No description provided for @configureEmbedding.
  ///
  /// In zh, this message translates to:
  /// **'配置 Embedding'**
  String get configureEmbedding;

  /// No description provided for @configureEmbeddingDesc.
  ///
  /// In zh, this message translates to:
  /// **'配置向量模型，用于知识库检索与语义匹配'**
  String get configureEmbeddingDesc;

  /// No description provided for @vectorModelConnectionInfo.
  ///
  /// In zh, this message translates to:
  /// **'向量模型连接信息'**
  String get vectorModelConnectionInfo;

  /// No description provided for @embeddingProviderDesc.
  ///
  /// In zh, this message translates to:
  /// **'选择后会自动填充默认模型、完整 Base URL 和向量维度'**
  String get embeddingProviderDesc;

  /// No description provided for @embeddingModelNameHint.
  ///
  /// In zh, this message translates to:
  /// **'例如：text-embedding-3-small'**
  String get embeddingModelNameHint;

  /// No description provided for @dimensionDesc.
  ///
  /// In zh, this message translates to:
  /// **'切换厂商会自动填充默认维度，跨厂商切换通常需要重建向量库'**
  String get dimensionDesc;

  /// No description provided for @embeddingRequiredFieldsHint.
  ///
  /// In zh, this message translates to:
  /// **'填写提供商、模型名称、API Key 和向量维度后可继续'**
  String get embeddingRequiredFieldsHint;

  /// No description provided for @vectorConfigWarningTitle.
  ///
  /// In zh, this message translates to:
  /// **'向量配置确认后不建议轻易更改'**
  String get vectorConfigWarningTitle;

  /// No description provided for @vectorConfigWarningDesc.
  ///
  /// In zh, this message translates to:
  /// **'模型、Base URL 或向量维度变更后，已有知识库向量可能不再兼容，通常需要重新生成或重建索引。'**
  String get vectorConfigWarningDesc;

  /// No description provided for @embeddingConfigSaved.
  ///
  /// In zh, this message translates to:
  /// **'Embedding 配置保存成功'**
  String get embeddingConfigSaved;

  /// No description provided for @connectWeChat.
  ///
  /// In zh, this message translates to:
  /// **'连接微信'**
  String get connectWeChat;

  /// No description provided for @connectWeChatDesc.
  ///
  /// In zh, this message translates to:
  /// **'微信登录为可选项，登录后可接收通知和使用微信消息能力'**
  String get connectWeChatDesc;

  /// No description provided for @connectionFailedWithError.
  ///
  /// In zh, this message translates to:
  /// **'连接失败: {error}'**
  String connectionFailedWithError(Object error);

  /// No description provided for @enterHome.
  ///
  /// In zh, this message translates to:
  /// **'进入首页'**
  String get enterHome;

  /// No description provided for @fetchingQrCode.
  ///
  /// In zh, this message translates to:
  /// **'正在获取二维码'**
  String get fetchingQrCode;

  /// No description provided for @refreshQrCode.
  ///
  /// In zh, this message translates to:
  /// **'重新获取二维码'**
  String get refreshQrCode;

  /// No description provided for @wechatOptionalHint.
  ///
  /// In zh, this message translates to:
  /// **'微信登录为可选项，你也可以稍后在设置中完成连接。'**
  String get wechatOptionalHint;

  /// No description provided for @connected.
  ///
  /// In zh, this message translates to:
  /// **'已连接'**
  String get connected;

  /// No description provided for @connecting.
  ///
  /// In zh, this message translates to:
  /// **'连接中'**
  String get connecting;

  /// No description provided for @waitingForScan.
  ///
  /// In zh, this message translates to:
  /// **'待扫码'**
  String get waitingForScan;

  /// No description provided for @confirming.
  ///
  /// In zh, this message translates to:
  /// **'确认中'**
  String get confirming;

  /// No description provided for @disconnected.
  ///
  /// In zh, this message translates to:
  /// **'未连接'**
  String get disconnected;

  /// No description provided for @wechatLoginSuccessDesc.
  ///
  /// In zh, this message translates to:
  /// **'登录成功，即将自动进入下一步。'**
  String get wechatLoginSuccessDesc;

  /// No description provided for @wechatFetchingQrDesc.
  ///
  /// In zh, this message translates to:
  /// **'正在获取登录二维码，请稍候。'**
  String get wechatFetchingQrDesc;

  /// No description provided for @wechatScanConfirmDesc.
  ///
  /// In zh, this message translates to:
  /// **'请使用微信扫一扫扫描二维码，并在手机上确认登录。'**
  String get wechatScanConfirmDesc;

  /// No description provided for @wechatScannedConfirmDesc.
  ///
  /// In zh, this message translates to:
  /// **'已扫码，请在微信客户端确认登录。'**
  String get wechatScannedConfirmDesc;

  /// No description provided for @wechatConnectErrorDesc.
  ///
  /// In zh, this message translates to:
  /// **'连接失败，可刷新二维码后重新扫码。'**
  String get wechatConnectErrorDesc;

  /// No description provided for @wechatDisconnectedDesc.
  ///
  /// In zh, this message translates to:
  /// **'点击刷新二维码后，使用微信扫码完成登录。'**
  String get wechatDisconnectedDesc;

  /// No description provided for @connectionSuccess.
  ///
  /// In zh, this message translates to:
  /// **'连接成功'**
  String get connectionSuccess;

  /// No description provided for @fetchingQrCodeEllipsis.
  ///
  /// In zh, this message translates to:
  /// **'正在获取二维码...'**
  String get fetchingQrCodeEllipsis;

  /// No description provided for @clickRefreshQrCode.
  ///
  /// In zh, this message translates to:
  /// **'点击刷新获取二维码'**
  String get clickRefreshQrCode;

  /// No description provided for @scanInstructions.
  ///
  /// In zh, this message translates to:
  /// **'扫码说明'**
  String get scanInstructions;

  /// No description provided for @scanInstructionOpenWeChat.
  ///
  /// In zh, this message translates to:
  /// **'打开微信手机客户端'**
  String get scanInstructionOpenWeChat;

  /// No description provided for @scanInstructionTapScan.
  ///
  /// In zh, this message translates to:
  /// **'点击右上角“+”，选择“扫一扫”'**
  String get scanInstructionTapScan;

  /// No description provided for @scanInstructionConfirm.
  ///
  /// In zh, this message translates to:
  /// **'扫描页面中的二维码并在手机上确认登录'**
  String get scanInstructionConfirm;
}

class _AppLocalizationsDelegate extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) => <String>['en', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {


  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en': return AppLocalizationsEn();
    case 'zh': return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.'
  );
}
