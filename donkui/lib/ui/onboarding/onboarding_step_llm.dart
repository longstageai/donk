import 'package:flutter/material.dart';
import 'package:flutter_smart_dialog/flutter_smart_dialog.dart';
import '../../app/conf/colors.dart';
import '../../common/service/setting_service.dart';

/// LLM 配置步骤
class OnboardingStepLLM extends StatefulWidget {
  final VoidCallback onCompleted;

  const OnboardingStepLLM({super.key, required this.onCompleted});

  @override
  State<OnboardingStepLLM> createState() => _OnboardingStepLLMState();
}

class _OnboardingStepLLMState extends State<OnboardingStepLLM> {
  /// 是否正在保存
  bool _isSaving = false;

  /// 错误信息
  String? _errorMessage;

  /// 表单控制器
  final _providerController = TextEditingController(text: 'openai');
  final _modelController = TextEditingController(text: 'gpt-4o-mini');
  final _apiKeyController = TextEditingController();
  final _baseUrlController = TextEditingController();
  final _dailyTokenLimitController = TextEditingController(text: '-1');

  /// 提供商配置信息
  final Map<String, Map<String, String>> _providerConfigs = {
    'openai': {
      'label': 'OpenAI',
      'defaultBaseUrl': 'https://api.openai.com/v1/chat/completions',
      'defaultModel': 'gpt-4o-mini',
    },
    'deepseek': {
      'label': 'DeepSeek',
      'defaultBaseUrl': 'https://api.deepseek.com/v1/chat/completions',
      'defaultModel': 'deepseek-chat',
    },
    'qwen': {
      'label': '通义千问',
      'defaultBaseUrl':
          'https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions',
      'defaultModel': 'qwen-turbo',
    },
    'doubao': {
      'label': '豆包',
      'defaultBaseUrl':
          'https://ark.cn-beijing.volces.com/api/v3/chat/completions',
      'defaultModel': 'doubao-seed-1-8-251228',
    },
  };

  @override
  void initState() {
    super.initState();
    _initDefaultValues();
  }

  void _initDefaultValues() {
    final defaultProvider = 'openai';
    final config = _providerConfigs[defaultProvider]!;
    _baseUrlController.text = config['defaultBaseUrl']!;
    _modelController.text = config['defaultModel']!;
  }

  @override
  void dispose() {
    _providerController.dispose();
    _modelController.dispose();
    _apiKeyController.dispose();
    _baseUrlController.dispose();
    _dailyTokenLimitController.dispose();
    super.dispose();
  }

  bool _isFormValid() {
    return _providerController.text.isNotEmpty &&
        _modelController.text.isNotEmpty &&
        _apiKeyController.text.isNotEmpty;
  }

  /// 保存配置
  Future<void> _saveConfig() async {
    setState(() {
      _isSaving = true;
      _errorMessage = null;
    });

    try {
      await SettingService.updateLLMConfig(
        provider: _providerController.text,
        model: _modelController.text,
        apiKey: _apiKeyController.text,
        baseUrl:
            _baseUrlController.text.isEmpty ? null : _baseUrlController.text,
      );

      await SettingService.updateAgentConfig(
        name: 'donk',
        maxLoop: 5,
        convergeAfter: 3,
        timeout: 300,
        dailyTokenLimit: int.tryParse(_dailyTokenLimitController.text) ?? -1,
        historyMaxEntries: 100,
        historyMaxDays: 30,
      );

      if (mounted) {
        _showToast('LLM 配置保存成功');
        widget.onCompleted();
      }
    } catch (e) {
      setState(() {
        _errorMessage = '保存失败: $e';
      });
    } finally {
      setState(() {
        _isSaving = false;
      });
    }
  }

  /// 显示提示
  void _showToast(String message) {
    SmartDialog.showToast(message);
  }

  @override
  Widget build(BuildContext context) {
    final isFormValid = _isFormValid();

    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  AppColors.primary.withAlpha(28),
                  AppColors.primary.withAlpha(8),
                ],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: AppColors.primary.withAlpha(30)),
            ),
            child: Row(
              children: [
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: AppColors.primary,
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: const Icon(
                    Icons.smart_toy_outlined,
                    color: Colors.white,
                    size: 26,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        '配置 LLM',
                        style: TextStyle(
                          fontSize: 24,
                          fontWeight: FontWeight.w700,
                          color: Colors.black87,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        '选择模型厂商并填写必要连接信息',
                        style: TextStyle(
                          fontSize: 14,
                          height: 1.4,
                          color: Colors.grey.shade700,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          if (_errorMessage != null)
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(14),
              margin: const EdgeInsets.only(bottom: 16),
              decoration: BoxDecoration(
                color: Colors.red.shade50,
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: Colors.red.shade100),
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.error_outline,
                    color: Colors.red.shade400,
                    size: 20,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      _errorMessage!,
                      style: TextStyle(
                        color: Colors.red.shade700,
                        fontSize: 14,
                        height: 1.4,
                      ),
                    ),
                  ),
                ],
              ),
            ),

          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: Colors.grey.shade200),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withAlpha(10),
                  blurRadius: 18,
                  offset: const Offset(0, 8),
                ),
              ],
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.tune, color: AppColors.primary, size: 20),
                    const SizedBox(width: 8),
                    const Text(
                      '模型连接信息',
                      style: TextStyle(
                        fontSize: 17,
                        fontWeight: FontWeight.w700,
                        color: Colors.black87,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 20),
                _buildFieldGroup(
                  label: '提供商',
                  required: true,
                  description: '选择后会自动填充默认模型和完整 Base URL',
                  child: _buildProviderDropdown(),
                ),
                _buildFieldGroup(
                  label: '模型名称',
                  required: true,
                  child: _buildTextField(
                    controller: _modelController,
                    hintText: '例如：gpt-4o-mini',
                    prefixIcon: Icons.memory_outlined,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                _buildFieldGroup(
                  label: 'API Key',
                  required: true,
                  description: '密钥仅用于服务端配置保存',
                  child: _buildTextField(
                    controller: _apiKeyController,
                    hintText: '输入您的 API Key',
                    prefixIcon: Icons.key_outlined,
                    obscureText: true,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                _buildFieldGroup(
                  label: 'Base URL',
                  description: '已按厂商默认填充，可按需修改',
                  child: _buildTextField(
                    controller: _baseUrlController,
                    hintText: '自定义 API 地址（可选）',
                    prefixIcon: Icons.link_outlined,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                _buildFieldGroup(
                  label: '每日 Token 消耗限制',
                  description: '-1 表示不限制',
                  bottomSpacing: 0,
                  child: _buildTextField(
                    controller: _dailyTokenLimitController,
                    hintText: '-1 表示不限制',
                    prefixIcon: Icons.speed_outlined,
                    keyboardType: TextInputType.number,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 18),

          Row(
            children: [
              Icon(
                isFormValid ? Icons.check_circle_outline : Icons.info_outline,
                size: 18,
                color:
                    isFormValid ? Colors.green.shade600 : Colors.grey.shade500,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  isFormValid ? '必填项已完成，可以进入下一步' : '填写提供商、模型名称和 API Key 后可继续',
                  style: TextStyle(
                    fontSize: 13,
                    color:
                        isFormValid
                            ? Colors.green.shade700
                            : Colors.grey.shade600,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),

          SizedBox(
            width: double.infinity,
            height: 52,
            child: ElevatedButton(
              onPressed: (_isSaving || !isFormValid) ? null : _saveConfig,
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.primary,
                foregroundColor: Colors.white,
                disabledBackgroundColor: Colors.grey.shade200,
                disabledForegroundColor: Colors.grey.shade500,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(16),
                ),
                elevation: 0,
              ),
              child:
                  _isSaving
                      ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(
                            Colors.white,
                          ),
                        ),
                      )
                      : const Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Text(
                            '下一步',
                            style: TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          SizedBox(width: 8),
                          Icon(Icons.arrow_forward, size: 18),
                        ],
                      ),
            ),
          ),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _buildFieldGroup({
    required String label,
    required Widget child,
    bool required = false,
    String? description,
    double bottomSpacing = 22,
  }) {
    return Padding(
      padding: EdgeInsets.only(bottom: bottomSpacing),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(
                label,
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                  color: Colors.black87,
                ),
              ),
              if (required) ...[
                const SizedBox(width: 4),
                Text(
                  '*',
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    color: Colors.red.shade400,
                  ),
                ),
              ],
            ],
          ),
          if (description != null) ...[
            const SizedBox(height: 4),
            Text(
              description,
              style: TextStyle(
                fontSize: 12,
                height: 1.35,
                color: Colors.grey.shade600,
              ),
            ),
          ],
          const SizedBox(height: 10),
          child,
        ],
      ),
    );
  }

  /// 构建文本输入框
  Widget _buildTextField({
    required TextEditingController controller,
    required String hintText,
    bool obscureText = false,
    TextInputType? keyboardType,
    IconData? prefixIcon,
    Function(String)? onChanged,
  }) {
    return TextField(
      controller: controller,
      obscureText: obscureText,
      keyboardType: keyboardType,
      onChanged: onChanged,
      style: const TextStyle(fontSize: 14, color: Colors.black87),
      decoration: InputDecoration(
        hintText: hintText,
        prefixIcon:
            prefixIcon == null
                ? null
                : Icon(prefixIcon, size: 20, color: Colors.grey.shade500),
        hintStyle: TextStyle(color: AppColors.textHint, fontSize: 14),
        filled: true,
        fillColor: Colors.grey.shade50,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: BorderSide(color: Colors.grey.shade200),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: BorderSide(color: Colors.grey.shade200),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: BorderSide(color: AppColors.primary, width: 1.5),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 15,
        ),
      ),
    );
  }

  /// 构建提供商下拉选择
  Widget _buildProviderDropdown() {
    return Container(
      decoration: BoxDecoration(
        color: Colors.grey.shade50,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.grey.shade200),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<String>(
          value: _providerController.text,
          isExpanded: true,
          icon: Icon(
            Icons.keyboard_arrow_down_rounded,
            color: AppColors.textSecondary,
          ),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 5),
          items:
              _providerConfigs.entries.map((entry) {
                return DropdownMenuItem<String>(
                  value: entry.key,
                  child: Row(
                    children: [
                      Icon(
                        Icons.hub_outlined,
                        size: 18,
                        color: AppColors.primary,
                      ),
                      const SizedBox(width: 10),
                      Text(
                        entry.value['label']!,
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                  ),
                );
              }).toList(),
          onChanged: (value) {
            setState(() {
              _providerController.text = value!;
              // 自动填充默认配置
              final config = _providerConfigs[value]!;
              _baseUrlController.text = config['defaultBaseUrl']!;
              _modelController.text = config['defaultModel']!;
            });
          },
        ),
      ),
    );
  }
}
