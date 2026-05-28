// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Donk';

  @override
  String get settings => 'Settings';

  @override
  String get generalSettings => 'General Settings';

  @override
  String get language => 'Language';

  @override
  String get languageDesc => 'Select application language';

  @override
  String get languageChinese => '中文';

  @override
  String get languageEnglish => 'English';

  @override
  String get securityProtection => 'Security Protection';

  @override
  String get securityProtectionDesc => 'Enable real-time AI security protection, prevent vulnerability attacks, and block malicious instructions';

  @override
  String get knowledgeAutoBuild => 'Knowledge Base Auto Build';

  @override
  String get knowledgeAutoBuildDesc => 'Automatically scan desktop, downloads, and documents folders, supporting txt, md, pdf, docx formats';

  @override
  String get sleepPrevention => 'Sleep Prevention';

  @override
  String get sleepPreventionDesc => 'When enabled, the computer will not enter sleep mode, keeping donk active';

  @override
  String get toolPermission => 'Tool Permission Limit';

  @override
  String get toolPermissionDesc => 'When enabled, agent tools will run with low privileges to prevent accidental deletion of critical files';

  @override
  String get close => 'Close';

  @override
  String get save => 'Save';

  @override
  String get cancel => 'Cancel';

  @override
  String get confirm => 'Confirm';

  @override
  String get delete => 'Delete';

  @override
  String get retry => 'Retry';

  @override
  String get reload => 'Reload';

  @override
  String get edit => 'Edit';

  @override
  String get add => 'Add';

  @override
  String get remove => 'Remove';

  @override
  String get open => 'Open';

  @override
  String get copy => 'Copy';

  @override
  String get search => 'Search';

  @override
  String get loading => 'Loading...';

  @override
  String get noData => 'No Data';

  @override
  String get error => 'Error';

  @override
  String get success => 'Success';

  @override
  String get failed => 'Failed';

  @override
  String get operationFailed => 'Operation Failed';

  @override
  String get operationSuccess => 'Operation Successful';

  @override
  String get confirmDelete => 'Confirm Delete';

  @override
  String get confirmRemove => 'Confirm Remove';

  @override
  String get deleteConfirmMessage => 'Are you sure you want to delete? This action cannot be undone.';

  @override
  String get removeConfirmMessage => 'Are you sure you want to remove? This action cannot be undone.';

  @override
  String get taskManagement => 'Task Management';

  @override
  String get executeConfirmTitle => 'Execute Now';

  @override
  String get executeConfirmMessage => 'Are you sure you want to execute the task immediately?';

  @override
  String executeConfirmMessageTask(Object name) {
    return 'Are you sure you want to execute task \"$name\" immediately?';
  }

  @override
  String taskExecuted(Object name) {
    return 'Task \"$name\" has started execution';
  }

  @override
  String get executeFailed => 'Execution failed';

  @override
  String get removeTask => 'Remove Task';

  @override
  String removeTaskConfirmMessage(Object name) {
    return 'Are you sure you want to remove task \"$name\"? This action cannot be undone.';
  }

  @override
  String get removeFailed => 'Remove failed';

  @override
  String get runRecords => 'Run Records';

  @override
  String deleteRunConfirmMessage(Object id, Object time) {
    return 'Are you sure you want to delete this run record?\n\nRun ID: $id\nExecution Time: $time';
  }

  @override
  String get statusDone => 'Completed';

  @override
  String get statusFailed => 'Failed';

  @override
  String get statusRunning => 'Running';

  @override
  String get taskStarted => 'Task has started';

  @override
  String get taskRemoved => 'Task removed';

  @override
  String get runRecordDeleted => 'Run record deleted';

  @override
  String get llmSettings => 'LLM Settings';

  @override
  String get agentSettings => 'Agent Settings';

  @override
  String get agentDetails => 'Agent Details';

  @override
  String get defaultAgentHint => 'Default Agent';

  @override
  String get idea => 'Ideas';

  @override
  String get chat => 'Chat';

  @override
  String get task => 'Tasks';

  @override
  String get embeddingSettings => 'Embedding Settings';

  @override
  String get about => 'About';

  @override
  String get currentVersion => 'Current Version';

  @override
  String get updateAvailable => 'Update Available';

  @override
  String get officialWebsite => 'Official Website';

  @override
  String get serviceAgreement => 'Service Agreement';

  @override
  String get privacyPolicy => 'Privacy Policy';

  @override
  String get notifications => 'Notifications';

  @override
  String get markAllRead => 'Mark All Read';

  @override
  String get clearAll => 'Clear All';

  @override
  String get clearConfirmTitle => 'Confirm Clear';

  @override
  String get clearConfirmMessage => 'Are you sure you want to clear all messages? This action cannot be undone.';

  @override
  String get justNow => 'Just now';

  @override
  String minutesAgo(Object count) {
    return '$count minutes ago';
  }

  @override
  String hoursAgo(Object count) {
    return '$count hours ago';
  }

  @override
  String daysAgo(Object count) {
    return '$count days ago';
  }

  @override
  String get noMessages => 'No Messages';

  @override
  String get websocketDisconnected => 'WebSocket Disconnected';

  @override
  String get reconnect => 'Reconnect';

  @override
  String get tokenStats => 'Token Statistics';

  @override
  String get tokenStatsDesc => 'Only counts default LLM usage; custom models not included';

  @override
  String get tokenUsageDetails => 'Token Usage Details';

  @override
  String get provider => 'Provider';

  @override
  String get modelName => 'Model Name';

  @override
  String get modelNameHint => 'e.g., gpt-4o-mini';

  @override
  String get apiKey => 'API Key';

  @override
  String get apiKeyHint => 'Enter API Key';

  @override
  String get baseUrl => 'Base URL';

  @override
  String get baseUrlHint => 'Optional, leave empty for default';

  @override
  String get dimension => 'Dimension';

  @override
  String get dimensionHint => 'e.g., 1536';

  @override
  String get temperature => 'Temperature';

  @override
  String get maxTokens => 'Max Tokens';

  @override
  String get maxLoop => 'Max Loop Count';

  @override
  String get maxLoopHint => 'e.g., 10';

  @override
  String get convergeAfter => 'Converge After';

  @override
  String get convergeAfterHint => 'e.g., 3';

  @override
  String get timeout => 'Timeout (seconds)';

  @override
  String get timeoutHint => 'e.g., 300';

  @override
  String get dailyTokenLimit => 'Daily Token Limit';

  @override
  String get dailyTokenLimitHint => '-1 means unlimited';

  @override
  String get todayTokenUsed => 'Today\'s Token Used';

  @override
  String get todayTokenRemaining => 'Today\'s Token Remaining';

  @override
  String get remainingPercent => 'Remaining Percentage';

  @override
  String get unlimited => 'Unlimited';

  @override
  String get wechatConnect => 'WeChat Connect';

  @override
  String get wechatScanQR => 'Scan QR code with WeChat';

  @override
  String get wechatQRCodeHint => 'The QR code will expire soon, please scan it quickly';

  @override
  String get wechatScanned => 'Scanned';

  @override
  String get wechatConfirmLogin => 'Please confirm login on your phone';

  @override
  String get wechatConnected => 'WeChat Connected';

  @override
  String get wechatConnectedDesc => 'You can receive and send WeChat messages';

  @override
  String get wechatConnecting => 'Connecting WeChat';

  @override
  String get wechatConnectFailed => 'Connection Failed';

  @override
  String get unknownError => 'Unknown error';

  @override
  String get wechatWaitScan => 'Waiting for QR Scan';

  @override
  String get wechatScanSuccess => 'Scan Successful';

  @override
  String get wechatConnectError => 'Connection Error';

  @override
  String get wechatDisconnected => 'WeChat Disconnected';

  @override
  String get disconnect => 'Disconnect';

  @override
  String get description => 'Description';

  @override
  String get skillFile => 'Skill File';

  @override
  String get callSettings => 'Call Settings';

  @override
  String get tags => 'Tags';

  @override
  String get resourceDir => 'Resource Directory';

  @override
  String get rescan => 'Rescan';

  @override
  String get enableAll => 'Enable All';

  @override
  String get disableAll => 'Disable All';

  @override
  String get refreshList => 'Refresh List';

  @override
  String get refresh => 'Refresh';

  @override
  String get clearMessages => 'Clear Messages';

  @override
  String tokenUsageStatus(Object percent, Object used) {
    return 'Used $used, Remaining $percent%';
  }

  @override
  String get noSkill => 'No Skill';

  @override
  String get clickScanToDiscover => 'Click scan button to discover Skills';

  @override
  String get ideaSquare => 'Idea Square';

  @override
  String get loadingFailed => 'Loading failed';

  @override
  String get scanFailed => 'Scan failed';

  @override
  String get deleteFailed => 'Delete failed';

  @override
  String deleteConfirmMessageSkill(Object name) {
    return 'Are you sure you want to delete Skill \"$name\"?\n\nThis action cannot be undone!';
  }

  @override
  String get allEnabled => 'All Skills are already enabled';

  @override
  String get allDisabled => 'All Skills are already disabled';

  @override
  String enabledCount(Object count) {
    return 'Enabled $count Skills';
  }

  @override
  String disabledCount(Object count) {
    return 'Disabled $count Skills';
  }

  @override
  String skillDeleted(Object name) {
    return 'Skill \"$name\" deleted';
  }

  @override
  String get skillStarted => 'Skill started';

  @override
  String get skillStopped => 'Skill stopped';

  @override
  String get fileNotFound => 'File not found';

  @override
  String get openFileFailed => 'Failed to open file';

  @override
  String get filePathCopied => 'File path copied';

  @override
  String get copyFilePath => 'Copy file path';

  @override
  String get invocationSettings => 'Invocation Settings';

  @override
  String get userInvocable => 'User Invocable';

  @override
  String get userInvocableDesc => 'Allow invocation via slash commands';

  @override
  String get userNotInvocableDesc => 'Do not allow user direct invocation';

  @override
  String get autoTrigger => 'Auto Trigger';

  @override
  String get autoTriggerEnabled => 'Allow model auto trigger';

  @override
  String get autoTriggerDisabled => 'Model auto trigger disabled';

  @override
  String get scripts => 'Scripts';

  @override
  String get references => 'References';

  @override
  String get assets => 'Assets';

  @override
  String get exists => 'Exists';

  @override
  String get notExists => 'None';

  @override
  String get unknownAuthor => 'Unknown Author';

  @override
  String get start => 'Start';

  @override
  String get stop => 'Stop';

  @override
  String get scanComplete => 'Scan complete';

  @override
  String get welcomeTitle => 'Hi, I\'m Donk';

  @override
  String get welcomeSubtitle => 'Help you work efficiently anytime, anywhere';

  @override
  String get installFirstSkill => 'Install your first Skill';

  @override
  String get installFirstSkillDesc => 'One-click to install superpowers';

  @override
  String get emailManagement => 'Email Management';

  @override
  String get emailManagementDesc => 'Help you handle emails efficiently';

  @override
  String get organizeDesktop => 'Organize Desktop';

  @override
  String get organizeDesktopDesc => 'Give you a clean desktop';

  @override
  String get scheduleManagement => 'Schedule Management';

  @override
  String get scheduleManagementDesc => 'Schedule meetings with one sentence';

  @override
  String get remoteWork => 'Remote Work';

  @override
  String get remoteWorkDesc => 'Handle tasks online anytime';

  @override
  String get scheduleTaskManagement => 'Schedule & Task Management';

  @override
  String get noRunRecords => 'No run records';

  @override
  String get fileTypeNotSupported => 'Only pdf, docx, txt, md files are supported';

  @override
  String get selectFileFailed => 'Failed to select file';

  @override
  String get agentCollaboration => 'Agent Collaboration';

  @override
  String agentActivityStatus(Object count) {
    return '$count updates · Live sync';
  }

  @override
  String get realtimeConnected => 'Live connection established';

  @override
  String get realtimeDisconnected => 'Live connection disconnected';

  @override
  String get noAgentMessages => 'No Agent Messages';

  @override
  String get agentActivityHint => 'Agent collaboration updates will appear here in real time';

  @override
  String get latestMessage => 'Latest';

  @override
  String get copyContent => 'Copy Content';

  @override
  String get contentCopied => 'Content copied to clipboard';

  @override
  String secondsAgo(Object count) {
    return '${count}s ago';
  }

  @override
  String get sessionStarted => 'Session started';

  @override
  String sessionStartFailed(Object error) {
    return 'Failed to start session: $error';
  }

  @override
  String get sessionStopped => 'Session stopped';

  @override
  String sessionStopFailed(Object error) {
    return 'Failed to stop session: $error';
  }

  @override
  String get clearAgentMessagesConfirm => 'Clear all Agent collaboration messages? This action cannot be undone.';

  @override
  String get onboardingWindowTitle => 'Donk Initial Setup';

  @override
  String get minimize => 'Minimize';

  @override
  String get maximize => 'Maximize';

  @override
  String get restore => 'Restore';

  @override
  String get previousStep => 'Previous';

  @override
  String get nextStep => 'Next';

  @override
  String get configureLLM => 'Configure LLM';

  @override
  String get configureLLMDesc => 'Select a model provider and enter the required connection information';

  @override
  String get modelConnectionInfo => 'Model Connection Information';

  @override
  String get llmProviderDesc => 'Selecting a provider automatically fills the default model and full Base URL';

  @override
  String get apiKeySaveDesc => 'The key is only used for server-side configuration storage';

  @override
  String get baseUrlDefaultDesc => 'Filled with the provider default. You can change it if needed';

  @override
  String get customApiUrlHint => 'Custom API address (optional)';

  @override
  String get requiredFieldsComplete => 'Required fields are complete. You can continue to the next step';

  @override
  String get llmRequiredFieldsHint => 'Enter provider, model name, and API Key to continue';

  @override
  String get llmConfigSaved => 'LLM configuration saved';

  @override
  String saveFailed(Object error) {
    return 'Save failed: $error';
  }

  @override
  String get providerQwen => 'Qwen';

  @override
  String get providerDoubao => 'Doubao';

  @override
  String get configureEmbedding => 'Configure Embedding';

  @override
  String get configureEmbeddingDesc => 'Configure the vector model for knowledge-base retrieval and semantic matching';

  @override
  String get vectorModelConnectionInfo => 'Vector Model Connection Information';

  @override
  String get embeddingProviderDesc => 'Selecting a provider automatically fills the default model, full Base URL, and vector dimension';

  @override
  String get embeddingModelNameHint => 'e.g., text-embedding-3-small';

  @override
  String get dimensionDesc => 'Switching providers fills the default dimension automatically. Switching across providers usually requires rebuilding the vector database';

  @override
  String get embeddingRequiredFieldsHint => 'Enter provider, model name, API Key, and vector dimension to continue';

  @override
  String get vectorConfigWarningTitle => 'Do not change vector configuration lightly after confirmation';

  @override
  String get vectorConfigWarningDesc => 'After the model, Base URL, or vector dimension changes, existing knowledge-base vectors may no longer be compatible and usually need to be regenerated or re-indexed.';

  @override
  String get embeddingConfigSaved => 'Embedding configuration saved';

  @override
  String get connectWeChat => 'Connect WeChat';

  @override
  String get connectWeChatDesc => 'WeChat login is optional. After login, you can receive notifications and use WeChat messaging capabilities';

  @override
  String connectionFailedWithError(Object error) {
    return 'Connection failed: $error';
  }

  @override
  String get enterHome => 'Enter Home';

  @override
  String get fetchingQrCode => 'Fetching QR code';

  @override
  String get refreshQrCode => 'Refresh QR code';

  @override
  String get wechatOptionalHint => 'WeChat login is optional. You can also connect it later in Settings.';

  @override
  String get connected => 'Connected';

  @override
  String get connecting => 'Connecting';

  @override
  String get waitingForScan => 'Waiting';

  @override
  String get confirming => 'Confirming';

  @override
  String get disconnected => 'Disconnected';

  @override
  String get wechatLoginSuccessDesc => 'Login succeeded. Entering the next step automatically.';

  @override
  String get wechatFetchingQrDesc => 'Fetching the login QR code. Please wait.';

  @override
  String get wechatScanConfirmDesc => 'Use WeChat to scan the QR code and confirm login on your phone.';

  @override
  String get wechatScannedConfirmDesc => 'Scanned. Please confirm login in WeChat.';

  @override
  String get wechatConnectErrorDesc => 'Connection failed. Refresh the QR code and scan again.';

  @override
  String get wechatDisconnectedDesc => 'Click refresh QR code, then scan with WeChat to log in.';

  @override
  String get connectionSuccess => 'Connected successfully';

  @override
  String get fetchingQrCodeEllipsis => 'Fetching QR code...';

  @override
  String get clickRefreshQrCode => 'Click refresh to get a QR code';

  @override
  String get scanInstructions => 'Scan Instructions';

  @override
  String get scanInstructionOpenWeChat => 'Open the WeChat mobile app';

  @override
  String get scanInstructionTapScan => 'Tap \"+\" in the upper-right corner and choose \"Scan\"';

  @override
  String get scanInstructionConfirm => 'Scan the QR code on this page and confirm login on your phone';
}
