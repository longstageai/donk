// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => 'Donk';

  @override
  String get settings => '设置';

  @override
  String get generalSettings => '通用设置';

  @override
  String get language => '语言';

  @override
  String get languageDesc => '选择应用显示语言';

  @override
  String get languageChinese => '中文';

  @override
  String get languageEnglish => 'English';

  @override
  String get securityProtection => '安全防护';

  @override
  String get securityProtectionDesc => '开启后可实时保护AI安全，防范漏洞攻击，拦截恶意指令、技能投毒等风险行为';

  @override
  String get knowledgeAutoBuild => '知识库自动构建';

  @override
  String get knowledgeAutoBuildDesc => '自动扫描桌面、下载、文档文件夹，支持 txt、md、pdf、docx 格式，构建可搜索的向量知识库';

  @override
  String get sleepPrevention => '休眠阻止';

  @override
  String get sleepPreventionDesc => '开启后，电脑将不会进入休眠模式，donk 会保持活跃状态';

  @override
  String get toolPermission => '工具权限限制';

  @override
  String get toolPermissionDesc => '开启后，智能体调用工具时会限制按照低权限执行，防止误删关键文件';

  @override
  String get close => '关闭';

  @override
  String get save => '保存';

  @override
  String get cancel => '取消';

  @override
  String get confirm => '确认';

  @override
  String get delete => '删除';

  @override
  String get retry => '重试';

  @override
  String get reload => '重新加载';

  @override
  String get edit => '编辑';

  @override
  String get add => '添加';

  @override
  String get remove => '移除';

  @override
  String get open => '打开';

  @override
  String get copy => '复制';

  @override
  String get search => '搜索';

  @override
  String get loading => '加载中...';

  @override
  String get noData => '暂无数据';

  @override
  String get error => '错误';

  @override
  String get success => '成功';

  @override
  String get failed => '失败';

  @override
  String get operationFailed => '操作失败';

  @override
  String get operationSuccess => '操作成功';

  @override
  String get confirmDelete => '确认删除';

  @override
  String get confirmRemove => '确认移除';

  @override
  String get deleteConfirmMessage => '确定要删除吗？此操作不可恢复。';

  @override
  String get removeConfirmMessage => '确定要移除吗？此操作不可恢复。';

  @override
  String get taskManagement => '任务管理';

  @override
  String get executeConfirmTitle => '立即执行';

  @override
  String get executeConfirmMessage => '确定要立即执行任务吗？';

  @override
  String executeConfirmMessageTask(Object name) {
    return '确定要立即执行任务\"$name\"吗？';
  }

  @override
  String taskExecuted(Object name) {
    return '任务\"$name\"已开始执行';
  }

  @override
  String get executeFailed => '执行失败';

  @override
  String get removeTask => '移除任务';

  @override
  String removeTaskConfirmMessage(Object name) {
    return '确定要移除任务\"$name\"吗？此操作不可恢复。';
  }

  @override
  String get removeFailed => '移除失败';

  @override
  String get runRecords => '运行记录';

  @override
  String deleteRunConfirmMessage(Object id, Object time) {
    return '确定要删除这条运行记录吗？\n\n执行ID: $id\n执行时间: $time';
  }

  @override
  String get statusDone => '已完成';

  @override
  String get statusFailed => '失败';

  @override
  String get statusRunning => '执行中';

  @override
  String get taskStarted => '任务已开始执行';

  @override
  String get taskRemoved => '任务已移除';

  @override
  String get runRecordDeleted => '运行记录已删除';

  @override
  String get llmSettings => 'LLM 设置';

  @override
  String get agentSettings => 'Agent 设置';

  @override
  String get agentDetails => 'Agent 详情';

  @override
  String get defaultAgentHint => '官方默认Agent';

  @override
  String get idea => '灵感';

  @override
  String get chat => '对话';

  @override
  String get task => '任务';

  @override
  String get embeddingSettings => 'Embedding 设置';

  @override
  String get about => '关于我们';

  @override
  String get currentVersion => '当前版本';

  @override
  String get updateAvailable => '可更新';

  @override
  String get officialWebsite => '进入官网';

  @override
  String get serviceAgreement => '服务协议';

  @override
  String get privacyPolicy => '隐私保护协议';

  @override
  String get notifications => '消息通知';

  @override
  String get markAllRead => '全部已读';

  @override
  String get clearAll => '清空全部';

  @override
  String get clearConfirmTitle => '确认清空';

  @override
  String get clearConfirmMessage => '确定要清空所有消息吗？此操作不可恢复。';

  @override
  String get justNow => '刚刚';

  @override
  String minutesAgo(Object count) {
    return '$count分钟前';
  }

  @override
  String hoursAgo(Object count) {
    return '$count小时前';
  }

  @override
  String daysAgo(Object count) {
    return '$count天前';
  }

  @override
  String get noMessages => '暂无消息';

  @override
  String get websocketDisconnected => 'WebSocket未连接';

  @override
  String get reconnect => '重新连接';

  @override
  String get tokenStats => '用量统计';

  @override
  String get tokenStatsDesc => '仅统计默认大模型的用量数据；不包含自定义模型数据';

  @override
  String get tokenUsageDetails => 'Token使用详情';

  @override
  String get provider => '提供商';

  @override
  String get modelName => '模型名称';

  @override
  String get modelNameHint => '如: gpt-4o-mini';

  @override
  String get apiKey => 'API Key';

  @override
  String get apiKeyHint => '输入 API Key';

  @override
  String get baseUrl => 'Base URL';

  @override
  String get baseUrlHint => '可选，留空使用默认地址';

  @override
  String get dimension => '向量维度';

  @override
  String get dimensionHint => '如: 1536';

  @override
  String get temperature => '温度';

  @override
  String get maxTokens => '最大 Token 数';

  @override
  String get maxLoop => '最大循环次数';

  @override
  String get maxLoopHint => '如: 10';

  @override
  String get convergeAfter => '收敛终止数';

  @override
  String get convergeAfterHint => '如: 3';

  @override
  String get timeout => '超时时间（秒）';

  @override
  String get timeoutHint => '如: 300';

  @override
  String get dailyTokenLimit => '每日 Token 限额';

  @override
  String get dailyTokenLimitHint => '-1 表示无限制';

  @override
  String get todayTokenUsed => '今日消耗 Token';

  @override
  String get todayTokenRemaining => '今日剩余 Token';

  @override
  String get remainingPercent => '剩余百分比';

  @override
  String get unlimited => '无限制';

  @override
  String get wechatConnect => '微信连接';

  @override
  String get wechatScanQR => '请使用微信扫描二维码';

  @override
  String get wechatQRCodeHint => '二维码将在一段时间后过期，请尽快扫描';

  @override
  String get wechatScanned => '已扫码';

  @override
  String get wechatConfirmLogin => '请在手机上确认登录';

  @override
  String get wechatConnected => '微信已连接';

  @override
  String get wechatConnectedDesc => '您可以接收和发送微信消息';

  @override
  String get wechatConnecting => '正在连接微信';

  @override
  String get wechatConnectFailed => '连接失败';

  @override
  String get unknownError => '未知错误';

  @override
  String get wechatWaitScan => '等待扫码登录';

  @override
  String get wechatScanSuccess => '扫码成功';

  @override
  String get wechatConnectError => '连接出现错误';

  @override
  String get wechatDisconnected => '微信未连接';

  @override
  String get disconnect => '断开连接';

  @override
  String get description => '描述';

  @override
  String get skillFile => 'Skill 文件';

  @override
  String get callSettings => '调用设置';

  @override
  String get tags => '标签';

  @override
  String get resourceDir => '资源目录';

  @override
  String get rescan => '重新扫描';

  @override
  String get enableAll => '全部启用';

  @override
  String get disableAll => '全部禁用';

  @override
  String get refreshList => '刷新列表';

  @override
  String get refresh => '刷新';

  @override
  String get clearMessages => '清空消息';

  @override
  String tokenUsageStatus(Object percent, Object used) {
    return '已使用$used，剩余$percent%';
  }

  @override
  String get noSkill => '暂无 Skill';

  @override
  String get clickScanToDiscover => '点击右上角扫描按钮发现 Skill';

  @override
  String get ideaSquare => '灵感广场';

  @override
  String get loadingFailed => '加载失败';

  @override
  String get scanFailed => '扫描失败';

  @override
  String get deleteFailed => '删除失败';

  @override
  String deleteConfirmMessageSkill(Object name) {
    return '确定要删除 Skill \"$name\" 吗？\n\n此操作不可恢复！';
  }

  @override
  String get allEnabled => '所有 Skill 已处于启用状态';

  @override
  String get allDisabled => '所有 Skill 已处于禁用状态';

  @override
  String enabledCount(Object count) {
    return '已启用 $count 个 Skill';
  }

  @override
  String disabledCount(Object count) {
    return '已禁用 $count 个 Skill';
  }

  @override
  String skillDeleted(Object name) {
    return 'Skill \"$name\" 已删除';
  }

  @override
  String get skillStarted => 'Skill 已启动';

  @override
  String get skillStopped => 'Skill 已停止';

  @override
  String get fileNotFound => '文件不存在';

  @override
  String get openFileFailed => '打开文件失败';

  @override
  String get filePathCopied => '文件路径已复制';

  @override
  String get copyFilePath => '复制文件路径';

  @override
  String get invocationSettings => '调用设置';

  @override
  String get userInvocable => '用户可调用';

  @override
  String get userInvocableDesc => '允许通过斜杠命令调用';

  @override
  String get userNotInvocableDesc => '不允许用户直接调用';

  @override
  String get autoTrigger => '自动触发';

  @override
  String get autoTriggerEnabled => '允许模型自动触发';

  @override
  String get autoTriggerDisabled => '已禁用模型自动触发';

  @override
  String get scripts => '脚本';

  @override
  String get references => '参考文档';

  @override
  String get assets => '资源文件';

  @override
  String get exists => '存在';

  @override
  String get notExists => '无';

  @override
  String get unknownAuthor => '未知作者';

  @override
  String get start => '启动';

  @override
  String get stop => '停止';

  @override
  String get scanComplete => '扫描完成';

  @override
  String get welcomeTitle => 'Hi，我是Donk';

  @override
  String get welcomeSubtitle => '随时随地，帮您高效干活';

  @override
  String get installFirstSkill => '安装你的第一个 Skill';

  @override
  String get installFirstSkillDesc => '一键教你安装超能力';

  @override
  String get emailManagement => '邮件管理';

  @override
  String get emailManagementDesc => '帮你高效处理邮件';

  @override
  String get organizeDesktop => '整理桌面';

  @override
  String get organizeDesktopDesc => '还你清爽电脑桌面';

  @override
  String get scheduleManagement => '安排日程';

  @override
  String get scheduleManagementDesc => '一句话约日程定会议';

  @override
  String get remoteWork => '手机远程办公';

  @override
  String get remoteWorkDesc => '随时处理在线任务';

  @override
  String get scheduleTaskManagement => '日程任务全管理';

  @override
  String get noRunRecords => '暂无运行记录';

  @override
  String get fileTypeNotSupported => '仅支持 pdf、docx、txt、md 文件';

  @override
  String get selectFileFailed => '选择文件失败';

  @override
  String get agentCollaboration => 'Donk 协作';

  @override
  String agentActivityStatus(Object count) {
    return '$count 条动态 · 实时同步';
  }

  @override
  String get realtimeConnected => '实时连接已建立';

  @override
  String get realtimeDisconnected => '实时连接已断开';

  @override
  String get noAgentMessages => '暂无 Donk 消息';

  @override
  String get agentActivityHint => 'Donk 协作动态会实时显示在这里';

  @override
  String get latestMessage => '最新消息';

  @override
  String get copyContent => '复制内容';

  @override
  String get contentCopied => '内容已复制到剪贴板';

  @override
  String secondsAgo(Object count) {
    return '$count秒前';
  }

  @override
  String get sessionStarted => '会话已启动';

  @override
  String sessionStartFailed(Object error) {
    return '会话启动失败：$error';
  }

  @override
  String get sessionStopped => '会话已停止';

  @override
  String sessionStopFailed(Object error) {
    return '会话停止失败：$error';
  }

  @override
  String get clearAgentMessagesConfirm => '确定清空所有 Donk 协作消息吗？清空后无法恢复。';

  @override
  String get onboardingWindowTitle => 'Donk 初始化配置';

  @override
  String get minimize => '最小化';

  @override
  String get maximize => '最大化';

  @override
  String get restore => '还原';

  @override
  String get previousStep => '上一步';

  @override
  String get nextStep => '下一步';

  @override
  String get configureLLM => '配置 LLM';

  @override
  String get configureLLMDesc => '选择模型厂商并填写必要连接信息';

  @override
  String get modelConnectionInfo => '模型连接信息';

  @override
  String get llmProviderDesc => '选择后会自动填充默认模型和完整 Base URL';

  @override
  String get apiKeySaveDesc => '密钥仅用于服务端配置保存';

  @override
  String get baseUrlDefaultDesc => '已按厂商默认填充，可按需修改';

  @override
  String get customApiUrlHint => '自定义 API 地址（可选）';

  @override
  String get requiredFieldsComplete => '必填项已完成，可以进入下一步';

  @override
  String get llmRequiredFieldsHint => '填写提供商、模型名称和 API Key 后可继续';

  @override
  String get llmConfigSaved => 'LLM 配置保存成功';

  @override
  String saveFailed(Object error) {
    return '保存失败: $error';
  }

  @override
  String get providerQwen => '通义千问';

  @override
  String get providerDoubao => '豆包';

  @override
  String get configureEmbedding => '配置 Embedding';

  @override
  String get configureEmbeddingDesc => '配置向量模型，用于知识库检索与语义匹配';

  @override
  String get vectorModelConnectionInfo => '向量模型连接信息';

  @override
  String get embeddingProviderDesc => '选择后会自动填充默认模型、完整 Base URL 和向量维度';

  @override
  String get embeddingModelNameHint => '例如：text-embedding-3-small';

  @override
  String get dimensionDesc => '切换厂商会自动填充默认维度，跨厂商切换通常需要重建向量库';

  @override
  String get embeddingRequiredFieldsHint => '填写提供商、模型名称、API Key 和向量维度后可继续';

  @override
  String get vectorConfigWarningTitle => '向量配置确认后不建议轻易更改';

  @override
  String get vectorConfigWarningDesc => '模型、Base URL 或向量维度变更后，已有知识库向量可能不再兼容，通常需要重新生成或重建索引。';

  @override
  String get embeddingConfigSaved => 'Embedding 配置保存成功';

  @override
  String get connectWeChat => '连接微信';

  @override
  String get connectWeChatDesc => '微信登录为可选项，登录后可接收通知和使用微信消息能力';

  @override
  String connectionFailedWithError(Object error) {
    return '连接失败: $error';
  }

  @override
  String get enterHome => '进入首页';

  @override
  String get fetchingQrCode => '正在获取二维码';

  @override
  String get refreshQrCode => '重新获取二维码';

  @override
  String get wechatOptionalHint => '微信登录为可选项，你也可以稍后在设置中完成连接。';

  @override
  String get connected => '已连接';

  @override
  String get connecting => '连接中';

  @override
  String get waitingForScan => '待扫码';

  @override
  String get confirming => '确认中';

  @override
  String get disconnected => '未连接';

  @override
  String get wechatLoginSuccessDesc => '登录成功，即将自动进入下一步。';

  @override
  String get wechatFetchingQrDesc => '正在获取登录二维码，请稍候。';

  @override
  String get wechatScanConfirmDesc => '请使用微信扫一扫扫描二维码，并在手机上确认登录。';

  @override
  String get wechatScannedConfirmDesc => '已扫码，请在微信客户端确认登录。';

  @override
  String get wechatConnectErrorDesc => '连接失败，可刷新二维码后重新扫码。';

  @override
  String get wechatDisconnectedDesc => '点击刷新二维码后，使用微信扫码完成登录。';

  @override
  String get connectionSuccess => '连接成功';

  @override
  String get fetchingQrCodeEllipsis => '正在获取二维码...';

  @override
  String get clickRefreshQrCode => '点击刷新获取二维码';

  @override
  String get scanInstructions => '扫码说明';

  @override
  String get scanInstructionOpenWeChat => '打开微信手机客户端';

  @override
  String get scanInstructionTapScan => '点击右上角“+”，选择“扫一扫”';

  @override
  String get scanInstructionConfirm => '扫描页面中的二维码并在手机上确认登录';
}
