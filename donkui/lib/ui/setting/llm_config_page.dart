import 'package:donk/app/layout/app_dialog.dart';
import 'package:flutter/material.dart';
import '../../common/service/setting_service.dart';

/// LLM 配置页面
class LLMConfigPage extends StatefulWidget {
  const LLMConfigPage({super.key});

  @override
  State<LLMConfigPage> createState() => _LLMConfigPageState();
}

class _LLMConfigPageState extends State<LLMConfigPage> {
  /// 是否正在加载
  bool _isLoading = true;

  /// 是否正在保存
  bool _isSaving = false;

  /// 错误信息
  String? _errorMessage;

  /// 表单控制器
  final _providerController = TextEditingController();
  final _modelController = TextEditingController();
  final _apiKeyController = TextEditingController();
  final _baseUrlController = TextEditingController();

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
    _loadConfig();
  }

  @override
  void dispose() {
    _providerController.dispose();
    _modelController.dispose();
    _apiKeyController.dispose();
    _baseUrlController.dispose();
    super.dispose();
  }

  /// 加载配置
  Future<void> _loadConfig() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final data = await SettingService.getConfig();
      final provider = data['llm_provider'] ?? 'openai';
      final config = _providerConfigs[provider] ?? _providerConfigs['openai']!;
      _providerController.text = provider;
      _modelController.text = data['llm_model'] ?? config['defaultModel']!;
      _apiKeyController.text = data['llm_api_key'] ?? '';
      _baseUrlController.text =
          data['llm_base_url'] ?? config['defaultBaseUrl']!;
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _errorMessage = '加载配置失败: $e';
      });
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  /// 保存配置
  Future<void> _saveConfig() async {
    setState(() {
      _isSaving = true;
    });

    try {
      await SettingService.updateLLMConfig(
        provider: _providerController.text,
        model: _modelController.text,
        apiKey: _apiKeyController.text.isEmpty ? null : _apiKeyController.text,
        baseUrl:
            _baseUrlController.text.isEmpty ? null : _baseUrlController.text,
      );
      _showToast('保存成功');
    } catch (e) {
      _showToast('保存失败: $e');
    } finally {
      if (mounted) {
        setState(() {
          _isSaving = false;
        });
      }
    }
  }

  /// 显示提示
  void _showToast(String message) {
    // 使用 Overlay 显示提示，避免 Navigator 上下文问题
    final overlay = Overlay.of(context);
    final overlayEntry = OverlayEntry(
      builder:
          (context) => Positioned(
            bottom: 50,
            left: 0,
            right: 0,
            child: Center(
              child: Material(
                color: Colors.transparent,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 24,
                    vertical: 12,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.black87,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    message,
                    style: const TextStyle(color: Colors.white, fontSize: 14),
                  ),
                ),
              ),
            ),
          ),
    );

    overlay.insert(overlayEntry);
    Future.delayed(const Duration(seconds: 2), () {
      overlayEntry.remove();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          /// 顶部标题栏
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'LLM 配置',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                  color: Colors.black87,
                ),
              ),
              Row(
                children: [
                  /// 保存按钮
                  if (!_isLoading)
                    ElevatedButton(
                      onPressed: _isSaving ? null : _saveConfig,
                      style: ElevatedButton.styleFrom(
                        backgroundColor: const Color(0xFF07C160),
                        foregroundColor: Colors.white,
                        padding: const EdgeInsets.symmetric(
                          horizontal: 20,
                          vertical: 10,
                        ),
                      ),
                      child:
                          _isSaving
                              ? const SizedBox(
                                width: 16,
                                height: 16,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  color: Colors.white,
                                ),
                              )
                              : const Text('保存'),
                    ),
                  const SizedBox(width: 12),

                  /// 关闭按钮
                  MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: GestureDetector(
                      onTap: () => AppDialog.dismiss(),
                      child: const Icon(
                        Icons.close,
                        size: 20,
                        color: Colors.grey,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
          const SizedBox(height: 24),

          /// 加载中
          if (_isLoading)
            const Expanded(child: Center(child: CircularProgressIndicator()))
          else if (_errorMessage != null)
            Expanded(
              child: Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(
                      _errorMessage!,
                      style: const TextStyle(color: Colors.red),
                    ),
                    const SizedBox(height: 16),
                    ElevatedButton(
                      onPressed: _loadConfig,
                      child: const Text('重新加载'),
                    ),
                  ],
                ),
              ),
            )
          else
            Expanded(
              child: SingleChildScrollView(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildDropdownField(
                      label: '提供商',
                      value:
                          _providerConfigs.containsKey(_providerController.text)
                              ? _providerController.text
                              : 'openai',
                      items: _providerConfigs.keys.toList(),
                      onChanged: (value) {
                        if (value != null) {
                          setState(() {
                            _providerController.text = value;
                            final config = _providerConfigs[value]!;
                            _baseUrlController.text = config['defaultBaseUrl']!;
                            _modelController.text = config['defaultModel']!;
                          });
                        }
                      },
                    ),
                    _buildTextField(
                      label: '模型名称',
                      controller: _modelController,
                      hintText: '如: gpt-4o-mini',
                    ),
                    _buildTextField(
                      label: 'API Key',
                      controller: _apiKeyController,
                      hintText: '输入 API Key',
                      isPassword: true,
                    ),
                    _buildTextField(
                      label: 'Base URL',
                      controller: _baseUrlController,
                      hintText: '可选，留空使用默认地址',
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }

  /// 构建文本输入框
  Widget _buildTextField({
    required String label,
    required TextEditingController controller,
    String? hintText,
    bool isPassword = false,
    TextInputType? keyboardType,
  }) {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: Colors.black87,
            ),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: controller,
            obscureText: isPassword,
            keyboardType: keyboardType,
            decoration: InputDecoration(
              hintText: hintText,
              filled: true,
              fillColor: const Color(0xFFF5F5F5),
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 12,
                vertical: 12,
              ),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide.none,
              ),
            ),
            style: const TextStyle(fontSize: 14),
          ),
        ],
      ),
    );
  }

  /// 构建下拉选择框
  Widget _buildDropdownField({
    required String label,
    required String value,
    required List<String> items,
    required ValueChanged<String?> onChanged,
  }) {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: Colors.black87,
            ),
          ),
          const SizedBox(height: 8),
          _CustomDropdown(value: value, items: items, onChanged: onChanged),
        ],
      ),
    );
  }
}

/// 自定义下拉组件（不使用 Navigator）
class _CustomDropdown extends StatefulWidget {
  final String value;
  final List<String> items;
  final ValueChanged<String?> onChanged;

  const _CustomDropdown({
    required this.value,
    required this.items,
    required this.onChanged,
  });

  @override
  State<_CustomDropdown> createState() => _CustomDropdownState();
}

class _CustomDropdownState extends State<_CustomDropdown> {
  bool _isOpen = false;
  OverlayEntry? _overlayEntry;
  final LayerLink _layerLink = LayerLink();

  void _toggleDropdown() {
    if (_isOpen) {
      _closeDropdown();
    } else {
      _openDropdown();
    }
  }

  void _openDropdown() {
    _overlayEntry = _createOverlayEntry();
    Overlay.of(context).insert(_overlayEntry!);
    setState(() => _isOpen = true);
  }

  void _closeDropdown() {
    _overlayEntry?.remove();
    _overlayEntry = null;
    if (mounted) {
      setState(() => _isOpen = false);
    }
  }

  OverlayEntry _createOverlayEntry() {
    final renderBox = context.findRenderObject() as RenderBox;
    final size = renderBox.size;

    return OverlayEntry(
      builder:
          (context) => GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: _closeDropdown,
            child: SizedBox.expand(
              child: Stack(
                children: [
                  Positioned(
                    width: size.width,
                    child: CompositedTransformFollower(
                      link: _layerLink,
                      showWhenUnlinked: false,
                      offset: Offset(0, size.height + 4),
                      child: Material(
                        elevation: 4,
                        borderRadius: BorderRadius.circular(8),
                        child: Container(
                          decoration: BoxDecoration(
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(8),
                            boxShadow: [
                              BoxShadow(
                                color: Colors.black.withValues(alpha: 0.1),
                                blurRadius: 8,
                                offset: const Offset(0, 4),
                              ),
                            ],
                          ),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children:
                                widget.items.map((item) {
                                  final isSelected = item == widget.value;
                                  return InkWell(
                                    onTap: () {
                                      widget.onChanged(item);
                                      _closeDropdown();
                                    },
                                    child: Container(
                                      width: double.infinity,
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 16,
                                        vertical: 12,
                                      ),
                                      decoration: BoxDecoration(
                                        color:
                                            isSelected
                                                ? const Color(
                                                  0xFF07C160,
                                                ).withValues(alpha: 0.1)
                                                : null,
                                        borderRadius: BorderRadius.circular(8),
                                      ),
                                      child: Text(
                                        item,
                                        style: TextStyle(
                                          fontSize: 14,
                                          color:
                                              isSelected
                                                  ? const Color(0xFF07C160)
                                                  : Colors.black87,
                                          fontWeight:
                                              isSelected
                                                  ? FontWeight.w500
                                                  : FontWeight.normal,
                                        ),
                                      ),
                                    ),
                                  );
                                }).toList(),
                          ),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
    );
  }

  @override
  void dispose() {
    _overlayEntry?.remove();
    _overlayEntry = null;
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return CompositedTransformTarget(
      link: _layerLink,
      child: GestureDetector(
        onTap: _toggleDropdown,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
          decoration: BoxDecoration(
            color: const Color(0xFFF5F5F5),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                widget.value,
                style: const TextStyle(fontSize: 14, color: Colors.black87),
              ),
              Icon(
                _isOpen ? Icons.arrow_drop_up : Icons.arrow_drop_down,
                color: Colors.black54,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
