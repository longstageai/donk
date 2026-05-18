import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:path/path.dart' as p;

import '../../common/model/skill.dart';
import '../../common/service/skill_service.dart';

/// Skill 详情页
/// 展示 Skill 的完整信息，支持打开 Skill 文件
class SkillDetailPage extends StatefulWidget {
  final Skill skill;

  const SkillDetailPage({super.key, required this.skill});

  @override
  State<SkillDetailPage> createState() => _SkillDetailPageState();
}

class _SkillDetailPageState extends State<SkillDetailPage> {
  late Skill _skill;

  @override
  void initState() {
    super.initState();
    _skill = widget.skill;
  }

  Future<void> _toggleSkillEnabled() async {
    final originalEnabled = _skill.enabled;
    final newEnabled = !originalEnabled;

    setState(() {
      _skill = _skill.copyWith(enabled: newEnabled);
    });

    try {
      if (originalEnabled) {
        await SkillService.disableSkill(_skill.name);
      } else {
        await SkillService.enableSkill(_skill.name);
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(newEnabled ? 'Skill 已启动' : 'Skill 已停止')),
        );
      }
    } catch (e) {
      setState(() {
        _skill = _skill.copyWith(enabled: originalEnabled);
      });
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('操作失败: $e')));
      }
    }
  }

  String get _skillRootPath {
    if (p.isAbsolute(_skill.path)) {
      return _skill.path;
    }
    return p.join(
      Directory.current.path,
      'donkserv',
      _skill.path,
    );
  }

  String get _skillFilePath => p.join(_skillRootPath, 'SKILL.md');

  Future<void> _openSkillFile() async {
    try {
      final filePath = _skillFilePath;
      final file = File(filePath);

      if (!await file.exists()) {
        if (mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('文件不存在: $filePath')));
        }
        return;
      }

      if (Platform.isWindows) {
        await Process.run('explorer', [filePath]);
      } else if (Platform.isMacOS) {
        await Process.run('open', [filePath]);
      } else if (Platform.isLinux) {
        await Process.run('xdg-open', [filePath]);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('打开文件失败: $e')));
      }
    }
  }

  void _copyPath() {
    Clipboard.setData(ClipboardData(text: _skillFilePath));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('文件路径已复制')),
    );
  }

  Color _getTagColor(String tag) {
    final colors = [
      Colors.blue,
      Colors.green,
      Colors.orange,
      Colors.purple,
      Colors.teal,
      Colors.red,
      Colors.indigo,
    ];
    return colors[tag.hashCode % colors.length];
  }

  IconData _getSkillIcon(String skillName) {
    final iconMap = {
      'browser': Icons.language,
      'canvas': Icons.palette,
      'cloud': Icons.cloud,
      'content': Icons.factory,
      'docx': Icons.description,
      'email': Icons.email,
      'pdf': Icons.picture_as_pdf,
      'image': Icons.image,
      'video': Icons.video_library,
      'audio': Icons.audiotrack,
      'search': Icons.search,
      'data': Icons.storage,
      'file': Icons.folder,
      'text': Icons.text_fields,
      'chat': Icons.chat,
      'calendar': Icons.calendar_today,
      'task': Icons.check_circle,
      'note': Icons.note,
      'code': Icons.code,
      'api': Icons.api,
      'web': Icons.web,
      'download': Icons.download,
      'upload': Icons.upload,
      'sync': Icons.sync,
      'backup': Icons.backup,
      'security': Icons.security,
      'setting': Icons.settings,
      'tool': Icons.build,
      'test': Icons.bug_report,
      'debug': Icons.developer_mode,
    };

    for (final entry in iconMap.entries) {
      if (skillName.toLowerCase().contains(entry.key)) {
        return entry.value;
      }
    }
    return Icons.extension;
  }

  @override
  Widget build(BuildContext context) {
    final accentColor = _getTagColor(
      _skill.tags.isNotEmpty ? _skill.tags.first : _skill.name,
    );

    return Scaffold(
      backgroundColor: const Color(0xFFFAFAFA),
      appBar: AppBar(
        backgroundColor: const Color(0xFFFAFAFA),
        elevation: 0,
        centerTitle: false,
        leading: IconButton(
          icon: const Icon(Icons.arrow_back, color: Colors.black87),
          onPressed: () => Navigator.of(context).pop(),
        ),
        title: const Text(
          'Skill 详情',
          style: TextStyle(
            color: Colors.black87,
            fontSize: 18,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 980),
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(24, 12, 24, 28),
            child: Column(
              children: [
                _buildHeroCard(accentColor),
                const SizedBox(height: 18),
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      flex: 6,
                      child: Column(
                        children: [
                          _buildDescriptionCard(),
                          const SizedBox(height: 18),
                          _buildFileCard(),
                        ],
                      ),
                    ),
                    const SizedBox(width: 18),
                    Expanded(
                      flex: 4,
                      child: Column(
                        children: [
                          _buildStatusCard(),
                          const SizedBox(height: 18),
                          if (_skill.tags.isNotEmpty) ...[
                            _buildTagsCard(),
                            const SizedBox(height: 18),
                          ],
                          _buildAssetsCard(),
                        ],
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildHeroCard(Color accentColor) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(28),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(22),
        border: Border.all(color: const Color(0xFFEFEFEF)),
      ),
      child: Row(
        children: [
          Container(
            width: 82,
            height: 82,
            decoration: BoxDecoration(
              color: accentColor.withAlpha(18),
              borderRadius: BorderRadius.circular(24),
            ),
            child: Icon(
              _getSkillIcon(_skill.name),
              size: 40,
              color: accentColor,
            ),
          ),
          const SizedBox(width: 22),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Flexible(
                      child: Text(
                        _skill.name,
                        style: const TextStyle(
                          fontSize: 26,
                          fontWeight: FontWeight.w700,
                          color: Color(0xFF1F1F1F),
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    const SizedBox(width: 12),
                    _buildPill(
                      text: _skill.enabled ? '已启动' : '已停止',
                      color: _skill.enabled
                          ? const Color(0xFF07C160)
                          : Colors.grey,
                      icon: _skill.enabled
                          ? Icons.play_circle_fill
                          : Icons.pause_circle_filled,
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    _buildMetaText(Icons.sell_outlined, 'v${_skill.version}'),
                    const SizedBox(width: 18),
                    _buildMetaText(
                      Icons.person_outline,
                      _skill.author.isNotEmpty ? _skill.author : '未知作者',
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(width: 22),
          SizedBox(
            height: 44,
            child: FilledButton.icon(
              onPressed: _toggleSkillEnabled,
              icon: Icon(_skill.enabled ? Icons.stop : Icons.play_arrow),
              label: Text(_skill.enabled ? '停止' : '启动'),
              style: FilledButton.styleFrom(
                backgroundColor: _skill.enabled
                    ? const Color(0xFFFF9800)
                    : const Color(0xFF07C160),
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(horizontal: 22),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildDescriptionCard() {
    return _buildPanel(
      title: '描述',
      icon: Icons.notes_outlined,
      child: Text(
        _skill.description.isNotEmpty ? _skill.description : '暂无描述',
        style: const TextStyle(
          fontSize: 15,
          height: 1.7,
          color: Color(0xFF555555),
        ),
      ),
    );
  }

  Widget _buildFileCard() {
    return _buildPanel(
      title: 'Skill 文件',
      icon: Icons.article_outlined,
      action: TextButton.icon(
        onPressed: _openSkillFile,
        icon: const Icon(Icons.open_in_new, size: 16),
        label: const Text('打开'),
        style: TextButton.styleFrom(foregroundColor: Colors.blue),
      ),
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: const Color(0xFFF8F8F8),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: const Color(0xFFEDEDED)),
        ),
        child: Row(
          children: [
            const Icon(Icons.article_outlined, color: Colors.blueGrey),
            const SizedBox(width: 12),
            Expanded(
              child: SelectableText(
                _skillFilePath,
                style: const TextStyle(
                  fontSize: 13,
                  color: Color(0xFF444444),
                  fontFamily: 'Consolas',
                ),
              ),
            ),
            IconButton(
              onPressed: _copyPath,
              icon: const Icon(Icons.copy, size: 18),
              tooltip: '复制文件路径',
              color: Colors.grey.shade600,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStatusCard() {
    return _buildPanel(
      title: '调用设置',
      icon: Icons.tune_outlined,
      child: Column(
        children: [
          _buildSettingTile(
            '用户可调用',
            _skill.userInvocable ? '允许通过斜杠命令调用' : '不允许用户直接调用',
            _skill.userInvocable,
          ),
          const SizedBox(height: 12),
          _buildSettingTile(
            '自动触发',
            _skill.disableModelInvocation ? '已禁用模型自动触发' : '允许模型自动触发',
            !_skill.disableModelInvocation,
          ),
        ],
      ),
    );
  }

  Widget _buildTagsCard() {
    return _buildPanel(
      title: '标签',
      icon: Icons.label_outline,
      child: Wrap(
        spacing: 8,
        runSpacing: 8,
        children: _skill.tags.map((tag) {
          final color = _getTagColor(tag);
          return Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
            decoration: BoxDecoration(
              color: color.withAlpha(14),
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              tag,
              style: TextStyle(
                fontSize: 13,
                color: color,
                fontWeight: FontWeight.w600,
              ),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildAssetsCard() {
    final items = [
      _DirectoryInfo('scripts', Icons.terminal, _skill.hasScripts),
      _DirectoryInfo('references', Icons.menu_book_outlined, _skill.hasReferences),
      _DirectoryInfo('assets', Icons.perm_media_outlined, _skill.hasAssets),
    ];

    return _buildPanel(
      title: '资源目录',
      icon: Icons.folder_copy_outlined,
      child: Column(
        children: items.map((item) {
          return Padding(
            padding: const EdgeInsets.only(bottom: 10),
            child: Row(
              children: [
                Container(
                  width: 36,
                  height: 36,
                  decoration: BoxDecoration(
                    color: item.exists
                        ? Colors.blue.withAlpha(12)
                        : Colors.grey.withAlpha(12),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Icon(
                    item.icon,
                    size: 18,
                    color: item.exists ? Colors.blue : Colors.grey,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    item.name,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: Color(0xFF333333),
                    ),
                  ),
                ),
                _buildPill(
                  text: item.exists ? '存在' : '无',
                  color: item.exists ? Colors.blue : Colors.grey,
                ),
              ],
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildPanel({
    required String title,
    required IconData icon,
    required Widget child,
    Widget? action,
  }) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: const Color(0xFFEFEFEF)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 20, color: const Color(0xFF666666)),
              const SizedBox(width: 8),
              Text(
                title,
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w700,
                  color: Color(0xFF222222),
                ),
              ),
              const Spacer(),
              if (action != null) action,
            ],
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }

  Widget _buildSettingTile(String title, String subtitle, bool enabled) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFFFAFAFA),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          Icon(
            enabled ? Icons.check_circle : Icons.remove_circle,
            color: enabled ? const Color(0xFF07C160) : Colors.grey,
            size: 22,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    color: Color(0xFF333333),
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  subtitle,
                  style: TextStyle(
                    fontSize: 12,
                    color: Colors.grey.shade600,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPill({
    required String text,
    required Color color,
    IconData? icon,
  }) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withAlpha(16),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 14, color: color),
            const SizedBox(width: 4),
          ],
          Text(
            text,
            style: TextStyle(
              fontSize: 12,
              color: color,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMetaText(IconData icon, String text) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 15, color: Colors.grey.shade500),
        const SizedBox(width: 5),
        Text(
          text,
          style: TextStyle(
            fontSize: 13,
            color: Colors.grey.shade600,
            fontWeight: FontWeight.w500,
          ),
        ),
      ],
    );
  }
}

class _DirectoryInfo {
  final String name;
  final IconData icon;
  final bool exists;

  const _DirectoryInfo(this.name, this.icon, this.exists);
}
