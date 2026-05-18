import 'package:flutter/material.dart';
import '../../common/model/skill.dart';
import '../../common/service/skill_service.dart';
import 'skill_detail_page.dart';

/// 灵感广场页面
/// 展示各种AI应用场景和灵感卡片，对接 Skill API 动态加载 Skill 列表
class IdeaView extends StatefulWidget {
  const IdeaView({super.key});

  @override
  State<IdeaView> createState() => _IdeaViewState();
}

class _IdeaViewState extends State<IdeaView> {
  /// Skill 列表
  List<Skill> _skills = [];

  /// 是否正在加载
  bool _isLoading = true;

  /// 错误信息
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _loadSkills();
  }

  /// 加载 Skill 列表
  Future<void> _loadSkills() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final response = await SkillService.getSkills();
      setState(() {
        _skills = response.data;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _errorMessage = '加载 Skill 失败: $e';
        _isLoading = false;
      });
    }
  }

  /// 切换 Skill 启用状态
  Future<void> _toggleSkillEnabled(Skill skill) async {
    // 乐观更新：先更新本地状态
    final originalEnabled = skill.enabled;
    final newEnabled = !originalEnabled;

    setState(() {
      final index = _skills.indexWhere((s) => s.name == skill.name);
      if (index != -1) {
        _skills[index] = _skills[index].copyWith(enabled: newEnabled);
      }
    });

    try {
      if (originalEnabled) {
        await SkillService.disableSkill(skill.name);
      } else {
        await SkillService.enableSkill(skill.name);
      }
    } catch (e) {
      // 失败时回滚状态
      setState(() {
        final index = _skills.indexWhere((s) => s.name == skill.name);
        if (index != -1) {
          _skills[index] = _skills[index].copyWith(enabled: originalEnabled);
        }
      });
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('操作失败: $e')));
      }
    }
  }

  /// 重新扫描 Skill
  Future<void> _rescanSkills() async {
    setState(() {
      _isLoading = true;
    });

    try {
      await SkillService.rescanSkills();
      await _loadSkills();
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('扫描完成')));
      }
    } catch (e) {
      setState(() {
        _errorMessage = '扫描失败: $e';
        _isLoading = false;
      });
    }
  }

  /// 全部启用 Skill
  Future<void> _enableAllSkills() async {
    final disabledSkills = _skills.where((s) => !s.enabled).toList();
    if (disabledSkills.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(
            const SnackBar(content: Text('所有 Skill 已处于启用状态')));
      }
      return;
    }

    // 乐观更新：先更新本地状态
    setState(() {
      for (var i = 0; i < _skills.length; i++) {
        if (!_skills[i].enabled) {
          _skills[i] = _skills[i].copyWith(enabled: true);
        }
      }
    });

    try {
      for (final skill in disabledSkills) {
        await SkillService.enableSkill(skill.name);
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('已启用 ${disabledSkills.length} 个 Skill')),
        );
      }
    } catch (e) {
      // 失败时重新加载列表
      await _loadSkills();
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('操作失败: $e')));
      }
    }
  }

  /// 全部禁用 Skill
  Future<void> _disableAllSkills() async {
    final enabledSkills = _skills.where((s) => s.enabled).toList();
    if (enabledSkills.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(
            const SnackBar(content: Text('所有 Skill 已处于禁用状态')));
      }
      return;
    }

    // 乐观更新：先更新本地状态
    setState(() {
      for (var i = 0; i < _skills.length; i++) {
        if (_skills[i].enabled) {
          _skills[i] = _skills[i].copyWith(enabled: false);
        }
      }
    });

    try {
      for (final skill in enabledSkills) {
        await SkillService.disableSkill(skill.name);
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('已禁用 ${enabledSkills.length} 个 Skill')),
        );
      }
    } catch (e) {
      // 失败时重新加载列表
      await _loadSkills();
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('操作失败: $e')));
      }
    }
  }

  /// 删除 Skill
  Future<void> _deleteSkill(Skill skill) async {
    // 显示确认对话框
    final confirmed = await showDialog<bool>(
      context: context,
      builder:
          (context) =>
          AlertDialog(
            title: const Text('确认删除'),
            content: Text(
                '确定要删除 Skill "${skill.name}" 吗？\n\n此操作不可恢复！'),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(context).pop(false),
                child: const Text('取消'),
              ),
              TextButton(
                onPressed: () => Navigator.of(context).pop(true),
                style: TextButton.styleFrom(foregroundColor: Colors.red),
                child: const Text('删除'),
              ),
            ],
          ),
    );

    if (confirmed != true) return;

    try {
      await SkillService.deleteSkill(skill.name);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Skill "${skill.name}" 已删除')));
      }
      // 重新加载列表
      await _loadSkills();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('删除失败: $e')));
      }
    }
  }

  /// 获取标签对应的颜色
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

  /// 获取 Skill 对应的图标
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
    return Scaffold(
      backgroundColor: Colors.white,
      body: Container(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [

            /// 页面标题
            _buildHeader(),
            const SizedBox(height: 20),

            /// Skill 卡片网格
            Expanded(child: _buildSkillGrid()),
          ],
        ),
      ),
    );
  }

  /// 构建页面标题
  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        const Text(
          '灵感广场',
          style: TextStyle(
            fontSize: 20,
            fontWeight: FontWeight.bold,
            color: Colors.black87,
          ),
        ),
        Row(
          children: [
            IconButton(
              onPressed:
              _isLoading || _skills.isEmpty ? null : _enableAllSkills,
              icon: const Icon(
                Icons.play_circle_outline,
                size: 20,
                color: Colors.green,
              ),
              tooltip: '全部开启',
            ),
            IconButton(
              onPressed:
              _isLoading || _skills.isEmpty ? null : _disableAllSkills,
              icon: const Icon(
                Icons.pause_circle_outline,
                size: 18,
                color: Colors.orange,
              ),
              tooltip: '全部禁用',
            ),

            /// 刷新按钮
            IconButton(
              onPressed: _isLoading ? null : _loadSkills,
              icon: const Icon(Icons.refresh, size: 20),
              tooltip: '刷新列表',
            ),

            /// 扫描按钮
            IconButton(
              onPressed: _isLoading ? null : _rescanSkills,
              icon: const Icon(Icons.scanner, size: 20),
              tooltip: '重新扫描',
            ),
          ],
        ),
      ],
    );
  }

  /// 构建 Skill 卡片网格
  Widget _buildSkillGrid() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_errorMessage != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 48, color: Colors.red),
            const SizedBox(height: 16),
            Text(
              _errorMessage!,
              style: const TextStyle(color: Colors.red),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            ElevatedButton(onPressed: _loadSkills, child: const Text('重试')),
          ],
        ),
      );
    }

    if (_skills.isEmpty) {
      return _buildEmptyState();
    }

    return GridView.builder(
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 3,
        crossAxisSpacing: 16,
        mainAxisSpacing: 16,
        childAspectRatio: 1.4,
      ),
      itemCount: _skills.length,
      itemBuilder: (context, index) {
        final skill = _skills[index];
        return _buildSkillCard(skill: skill);
      },
    );
  }

  /// 构建空状态界面
  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.extension_outlined, size: 64, color: Colors.grey.shade300),
          const SizedBox(height: 16),
          Text(
            '暂无 Skill',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w500,
              color: Colors.grey.shade600,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            '点击右上角扫描按钮发现 Skill',
            style: TextStyle(fontSize: 14, color: Colors.grey.shade400),
          ),
          const SizedBox(height: 24),
          ElevatedButton.icon(
            onPressed: _rescanSkills,
            icon: const Icon(Icons.scanner, size: 18),
            label: const Text('重新扫描'),
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFF333333),
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// 构建单个 Skill 卡片
  Widget _buildSkillCard({required Skill skill}) {
    final iconColor = _getTagColor(
      skill.tags.isNotEmpty ? skill.tags.first : skill.name,
    );

    return InkWell(
      onTap: () => _navigateToSkillDetail(skill),
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: const Color(0xFFF8F8F8),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0xFFEEEEEE)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [

            /// 图标和开关
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: iconColor.withAlpha(20),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(
                    _getSkillIcon(skill.name),
                    size: 20,
                    color: iconColor,
                  ),
                ),
                Row(
                  children: [

                    /// 启用/禁用开关
                    /// 开关
                    Transform.scale(

                      /// 缩放比例，使开关变小
                      scale: 0.8,
                      child: Switch(
                        value: skill.enabled,
                        onChanged: (value) => _toggleSkillEnabled(skill),

                        /// 激活颜色
                        activeColor: Colors.white,
                        activeTrackColor: const Color(0xFF07C160),

                        /// 非激活颜色
                        inactiveThumbColor: Colors.white,
                        inactiveTrackColor: Colors.grey.shade300,
                      ),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 12),

            /// 标题
            Text(
              skill.name,
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.bold,
                color: Colors.black87,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 4),

            /// 描述
            Expanded(
              child: Text(
                skill.description,
                style: const TextStyle(fontSize: 12, color: Colors.grey),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            const SizedBox(height: 8),

            /// 标签和版本
            Row(
              children: [
                if (skill.tags.isNotEmpty)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 6,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: iconColor.withAlpha(20),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      skill.tags.first,
                      style: TextStyle(fontSize: 10, color: iconColor),
                    ),
                  ),
                const Spacer(),
                Text(
                  'v${skill.version}',
                  style: const TextStyle(fontSize: 10, color: Colors.grey),
                ),
                SizedBox(width: 10),

                /// 删除按钮
                IconButton(
                  onPressed: () => _deleteSkill(skill),
                  icon: const Icon(Icons.delete_outline, size: 18),
                  color: Colors.red,
                  tooltip: '删除 Skill',
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(
                      minWidth: 32, minHeight: 32),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  /// 跳转到 Skill 详情页
  void _navigateToSkillDetail(Skill skill) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (context) => SkillDetailPage(skill: skill),
      ),
    );
  }
}