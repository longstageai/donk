import 'package:donk/app/layout/app_dialog.dart';
import 'package:flutter/material.dart';
import '../../common/service/setting_service.dart';

/// Agent 配置页面
class AgentConfigPage extends StatefulWidget {
  const AgentConfigPage({super.key});

  @override
  State<AgentConfigPage> createState() => _AgentConfigPageState();
}

class _AgentConfigPageState extends State<AgentConfigPage> {
  /// 是否正在加载
  bool _isLoading = true;

  /// 是否正在保存
  bool _isSaving = false;

  /// 错误信息
  String? _errorMessage;

  /// 表单控制器
  final _nameController = TextEditingController();
  final _maxLoopController = TextEditingController();
  final _convergeAfterController = TextEditingController();
  final _timeoutController = TextEditingController();
  final _dailyTokenLimitController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _loadConfig();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _maxLoopController.dispose();
    _convergeAfterController.dispose();
    _timeoutController.dispose();
    _dailyTokenLimitController.dispose();
    super.dispose();
  }

  /// 加载配置
  Future<void> _loadConfig() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final data = await SettingService.getAgentConfig();
      _nameController.text = data['name'] ?? 'donk';
      _maxLoopController.text = (data['max_loop'] ?? 3).toString();
      _convergeAfterController.text = (data['converge_after'] ?? 3).toString();
      _timeoutController.text = (data['timeout'] ?? 300).toString();
      _dailyTokenLimitController.text =
          (data['daily_token_limit'] ?? -1).toString();
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
      await SettingService.updateAgentConfig(
        name: _nameController.text,
        maxLoop: int.tryParse(_maxLoopController.text) ?? 10,
        convergeAfter: int.tryParse(_convergeAfterController.text) ?? 3,
        timeout: int.tryParse(_timeoutController.text) ?? 300,
        dailyTokenLimit: int.tryParse(_dailyTokenLimitController.text) ?? -1,
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
                'Agent 配置',
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
                    // _buildTextField(
                    //   label: 'Agent 名称',
                    //   controller: _nameController,
                    //   hintText: '如: donk',
                    // ),
                    _buildTextField(
                      label: '最大循环次数',
                      controller: _maxLoopController,
                      hintText: '如: 10',
                      keyboardType: TextInputType.number,
                      helperText: 'Agent 执行任务时的最大循环次数',
                    ),
                    _buildTextField(
                      label: '收敛终止数',
                      controller: _convergeAfterController,
                      hintText: '如: 3',
                      keyboardType: TextInputType.number,
                      helperText: '连续无工具调用次数达到此值时终止',
                    ),
                    _buildTextField(
                      label: '超时时间（秒）',
                      controller: _timeoutController,
                      hintText: '如: 300',
                      keyboardType: TextInputType.number,
                      helperText: '单次任务执行的超时时间',
                    ),
                    _buildTextField(
                      label: '每日 Token 限额',
                      controller: _dailyTokenLimitController,
                      hintText: '-1 表示无限制',
                      keyboardType: TextInputType.number,
                      helperText: '-1 表示不限制每日 Token 使用量',
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
    String? helperText,
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
              helperText: helperText,
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
}
