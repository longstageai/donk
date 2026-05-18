/// Skill 数据模型
/// 用于表示 Skill 的基本信息
class Skill {
  /// Skill 名称（唯一标识）
  final String name;

  /// Skill 描述
  final String description;

  /// 版本号
  final String version;

  /// 作者
  final String author;

  /// 标签列表
  final List<String> tags;

  /// 是否启用
  final bool enabled;

  /// 是否允许用户通过斜杠命令调用
  final bool userInvocable;

  /// 是否禁止自动触发
  final bool disableModelInvocation;

  /// Skill 目录路径
  final String path;

  /// 是否包含 scripts 目录
  final bool hasScripts;

  /// 是否包含 references 目录
  final bool hasReferences;

  /// 是否包含 assets 目录
  final bool hasAssets;

  Skill({
    required this.name,
    required this.description,
    required this.version,
    required this.author,
    required this.tags,
    required this.enabled,
    required this.userInvocable,
    required this.disableModelInvocation,
    required this.path,
    required this.hasScripts,
    required this.hasReferences,
    required this.hasAssets,
  });

  /// 从 JSON 数据创建 Skill 实例
  factory Skill.fromJson(Map<String, dynamic> json) {
    return Skill(
      name: json['name'] as String? ?? '',
      description: json['description'] as String? ?? '',
      version: json['version'] as String? ?? '',
      author: json['author'] as String? ?? '',
      tags:
          (json['tags'] as List<dynamic>?)
              ?.map((e) => e as String? ?? '')
              .toList() ??
          [],
      enabled: json['enabled'] as bool? ?? false,
      userInvocable: json['user_invocable'] as bool? ?? false,
      disableModelInvocation:
          json['disable_model_invocation'] as bool? ?? false,
      path: json['path'] as String? ?? '',
      hasScripts: json['has_scripts'] as bool? ?? false,
      hasReferences: json['has_references'] as bool? ?? false,
      hasAssets: json['has_assets'] as bool? ?? false,
    );
  }

  /// 将 Skill 实例转换为 JSON 数据
  Map<String, dynamic> toJson() {
    return {
      'name': name,
      'description': description,
      'version': version,
      'author': author,
      'tags': tags,
      'enabled': enabled,
      'user_invocable': userInvocable,
      'disable_model_invocation': disableModelInvocation,
      'path': path,
      'has_scripts': hasScripts,
      'has_references': hasReferences,
      'has_assets': hasAssets,
    };
  }

  /// 复制当前实例并修改指定字段
  Skill copyWith({
    String? name,
    String? description,
    String? version,
    String? author,
    List<String>? tags,
    bool? enabled,
    bool? userInvocable,
    bool? disableModelInvocation,
    String? path,
    bool? hasScripts,
    bool? hasReferences,
    bool? hasAssets,
  }) {
    return Skill(
      name: name ?? this.name,
      description: description ?? this.description,
      version: version ?? this.version,
      author: author ?? this.author,
      tags: tags ?? this.tags,
      enabled: enabled ?? this.enabled,
      userInvocable: userInvocable ?? this.userInvocable,
      disableModelInvocation:
          disableModelInvocation ?? this.disableModelInvocation,
      path: path ?? this.path,
      hasScripts: hasScripts ?? this.hasScripts,
      hasReferences: hasReferences ?? this.hasReferences,
      hasAssets: hasAssets ?? this.hasAssets,
    );
  }
}

/// Skill 列表响应模型
class SkillListResponse {
  /// Skill 列表
  final List<Skill> data;

  /// 总数
  final int total;

  SkillListResponse({required this.data, required this.total});

  /// 从 JSON 数据创建 SkillListResponse 实例
  factory SkillListResponse.fromJson(Map<String, dynamic> json) {
    return SkillListResponse(
      data:
          (json['data'] as List<dynamic>)
              .map((e) => Skill.fromJson(e as Map<String, dynamic>))
              .toList(),
      total: json['total'] as int,
    );
  }
}

/// Skill 指令响应模型
class SkillInstructionsResponse {
  /// 指令内容（Markdown 格式）
  final String instructions;

  SkillInstructionsResponse({required this.instructions});

  /// 从 JSON 数据创建 SkillInstructionsResponse 实例
  factory SkillInstructionsResponse.fromJson(Map<String, dynamic> json) {
    return SkillInstructionsResponse(
      instructions: json['instructions'] as String,
    );
  }
}

/// Skill 脚本列表响应模型
class SkillScriptsResponse {
  /// 脚本文件列表
  final List<String> scripts;

  SkillScriptsResponse({required this.scripts});

  /// 从 JSON 数据创建 SkillScriptsResponse 实例
  factory SkillScriptsResponse.fromJson(Map<String, dynamic> json) {
    return SkillScriptsResponse(
      scripts:
          (json['scripts'] as List<dynamic>).map((e) => e as String).toList(),
    );
  }
}

/// Skill 脚本内容响应模型
class SkillScriptContentResponse {
  /// 脚本名称
  final String name;

  /// 脚本内容
  final String content;

  SkillScriptContentResponse({required this.name, required this.content});

  /// 从 JSON 数据创建 SkillScriptContentResponse 实例
  factory SkillScriptContentResponse.fromJson(Map<String, dynamic> json) {
    return SkillScriptContentResponse(
      name: json['name'] as String,
      content: json['content'] as String,
    );
  }
}

/// 操作响应模型
class SkillActionResponse {
  /// 响应消息
  final String message;

  SkillActionResponse({required this.message});

  /// 从 JSON 数据创建 SkillActionResponse 实例
  factory SkillActionResponse.fromJson(Map<String, dynamic> json) {
    return SkillActionResponse(message: json['message'] as String);
  }
}

/// Skill 错误响应模型
class SkillErrorResponse {
  /// 错误信息
  final String error;

  SkillErrorResponse({required this.error});

  /// 从 JSON 数据创建 SkillErrorResponse 实例
  factory SkillErrorResponse.fromJson(Map<String, dynamic> json) {
    return SkillErrorResponse(error: json['error'] as String);
  }
}
