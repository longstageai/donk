import 'package:flutter/material.dart';
import 'package:flutter_smart_dialog/flutter_smart_dialog.dart';
import '../../app/conf/colors.dart';
import '../../common/service/setting_service.dart';
import '../../l10n/generated/app_localizations.dart';

class OnboardingStepEmbedding extends StatefulWidget {
  final VoidCallback onCompleted;

  const OnboardingStepEmbedding({super.key, required this.onCompleted});

  @override
  State<OnboardingStepEmbedding> createState() =>
      _OnboardingStepEmbeddingState();
}

class _OnboardingStepEmbeddingState extends State<OnboardingStepEmbedding> {
  bool _isSaving = false;
  String? _errorMessage;

  final _providerController = TextEditingController(text: 'openai');
  final _modelController = TextEditingController(
    text: 'text-embedding-3-small',
  );
  final _apiKeyController = TextEditingController();
  final _baseUrlController = TextEditingController();
  final _dimensionController = TextEditingController(text: '1536');

  final Map<String, Map<String, String>> _providerConfigs = {
    'openai': {
      'label': 'OpenAI',
      'defaultBaseUrl': 'https://api.openai.com/v1/embeddings',
      'defaultModel': 'text-embedding-3-small',
      'defaultDimension': '1536',
    },
    'qwen': {
      'label': 'Qwen',
      'defaultBaseUrl':
          'https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings',
      'defaultModel': 'text-embedding-v3',
      'defaultDimension': '1024',
    },
    'doubao': {
      'label': 'Doubao',
      'defaultBaseUrl':
          'https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal',
      'defaultModel': 'doubao-embedding-vision-251215',
      'defaultDimension': '2048',
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
    _dimensionController.text = config['defaultDimension']!;
  }

  @override
  void dispose() {
    _providerController.dispose();
    _modelController.dispose();
    _apiKeyController.dispose();
    _baseUrlController.dispose();
    _dimensionController.dispose();
    super.dispose();
  }

  bool _isFormValid() {
    return _providerController.text.isNotEmpty &&
        _modelController.text.isNotEmpty &&
        _apiKeyController.text.isNotEmpty &&
        _dimensionController.text.isNotEmpty;
  }

  Future<void> _saveConfig() async {
    setState(() {
      _isSaving = true;
      _errorMessage = null;
    });

    try {
      await SettingService.updateEmbeddingConfig(
        provider: _providerController.text,
        model: _modelController.text,
        apiKey: _apiKeyController.text,
        baseUrl:
            _baseUrlController.text.isEmpty ? null : _baseUrlController.text,
        dimension: int.tryParse(_dimensionController.text) ?? 1536,
      );

      if (mounted) {
        _showToast(AppLocalizations.of(context)!.embeddingConfigSaved);
        widget.onCompleted();
      }
    } catch (e) {
      setState(() {
        _errorMessage = AppLocalizations.of(context)!.saveFailed(e);
      });
    } finally {
      setState(() {
        _isSaving = false;
      });
    }
  }

  void _showToast(String message) {
    SmartDialog.showToast(message);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
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
                    Icons.auto_awesome_mosaic_outlined,
                    color: Colors.white,
                    size: 26,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.configureEmbedding,
                        style: const TextStyle(
                          fontSize: 24,
                          fontWeight: FontWeight.w700,
                          color: Colors.black87,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        l10n.configureEmbeddingDesc,
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
                    Icon(
                      Icons.dataset_outlined,
                      color: AppColors.primary,
                      size: 20,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      l10n.vectorModelConnectionInfo,
                      style: const TextStyle(
                        fontSize: 17,
                        fontWeight: FontWeight.w700,
                        color: Colors.black87,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                _buildVectorConfigWarning(l10n),
                const SizedBox(height: 20),
                _buildFieldGroup(
                  label: l10n.provider,
                  required: true,
                  description: l10n.embeddingProviderDesc,
                  child: _buildProviderDropdown(l10n),
                ),
                _buildFieldGroup(
                  label: l10n.modelName,
                  required: true,
                  child: _buildTextField(
                    controller: _modelController,
                    hintText: l10n.embeddingModelNameHint,
                    prefixIcon: Icons.memory_outlined,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                _buildFieldGroup(
                  label: l10n.apiKey,
                  required: true,
                  description: l10n.apiKeySaveDesc,
                  child: _buildTextField(
                    controller: _apiKeyController,
                    hintText: l10n.apiKeyHint,
                    prefixIcon: Icons.key_outlined,
                    obscureText: true,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                _buildFieldGroup(
                  label: l10n.baseUrl,
                  description: l10n.baseUrlDefaultDesc,
                  child: _buildTextField(
                    controller: _baseUrlController,
                    hintText: l10n.customApiUrlHint,
                    prefixIcon: Icons.link_outlined,
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                _buildFieldGroup(
                  label: l10n.dimension,
                  required: true,
                  description: l10n.dimensionDesc,
                  bottomSpacing: 0,
                  child: _buildTextField(
                    controller: _dimensionController,
                    hintText: l10n.dimensionHint,
                    prefixIcon: Icons.straighten_outlined,
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
                  isFormValid
                      ? l10n.requiredFieldsComplete
                      : l10n.embeddingRequiredFieldsHint,
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
                      : Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Text(
                            l10n.nextStep,
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          const SizedBox(width: 8),
                          const Icon(Icons.arrow_forward, size: 18),
                        ],
                      ),
            ),
          ),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _buildVectorConfigWarning(AppLocalizations l10n) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.amber.shade50,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.amber.shade200),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.warning_amber_rounded,
            size: 20,
            color: Colors.amber.shade800,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.vectorConfigWarningTitle,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w700,
                    color: Colors.amber.shade900,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  l10n.vectorConfigWarningDesc,
                  style: TextStyle(
                    fontSize: 12,
                    height: 1.4,
                    color: Colors.amber.shade900,
                  ),
                ),
              ],
            ),
          ),
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

  Widget _buildProviderDropdown(AppLocalizations l10n) {
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
                        _providerLabel(l10n, entry.key),
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
              final config = _providerConfigs[value]!;
              _baseUrlController.text = config['defaultBaseUrl']!;
              _modelController.text = config['defaultModel']!;
              _dimensionController.text = config['defaultDimension']!;
            });
          },
        ),
      ),
    );
  }

  String _providerLabel(AppLocalizations l10n, String provider) {
    switch (provider) {
      case 'qwen':
        return l10n.providerQwen;
      case 'doubao':
        return l10n.providerDoubao;
      default:
        return _providerConfigs[provider]?['label'] ?? provider;
    }
  }
}
