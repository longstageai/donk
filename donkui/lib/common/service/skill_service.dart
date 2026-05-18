import 'dart:convert';
import 'package:donk/app/conf/config.dart' as app_config;
import '../client/http_client.dart';
import '../model/skill.dart';

/// Skill 服务类
/// 封装 Skill 管理系统的所有 HTTP API 调用
class SkillService {
  static final String _baseUrl = app_config.apiBaseUrl;
  static final HttpClientSingleton _http = HttpClientSingleton.instance;

  /// 获取 Skill 列表
  /// 获取所有 Skill 的基本信息（包括启用状态）
  static Future<SkillListResponse> getSkills() async {
    final body = await _http.get('$_baseUrl/skills');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillListResponse.fromJson(json);
  }

  /// 获取 Skill 详情
  /// [name] Skill 名称
  static Future<Skill> getSkillDetail(String name) async {
    final body = await _http.get('$_baseUrl/skills/$name');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return Skill.fromJson(json);
  }

  /// 启用 Skill
  /// [name] Skill 名称
  /// 启用后 Agent 可以加载和使用该 Skill
  static Future<SkillActionResponse> enableSkill(String name) async {
    final body = await _http.post('$_baseUrl/skills/$name/enable');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillActionResponse.fromJson(json);
  }

  /// 禁用 Skill
  /// [name] Skill 名称
  /// 禁用后 Agent 无法加载该 Skill
  static Future<SkillActionResponse> disableSkill(String name) async {
    final body = await _http.post('$_baseUrl/skills/$name/disable');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillActionResponse.fromJson(json);
  }

  /// 删除 Skill
  /// [name] Skill 名称
  /// 警告：此操作不可恢复！会同时删除文件系统目录和数据库记录
  static Future<SkillActionResponse> deleteSkill(String name) async {
    final body = await _http.delete('$_baseUrl/skills/$name');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillActionResponse.fromJson(json);
  }

  /// 重新扫描文件系统
  /// 扫描 data/skills 目录，将新发现的 Skill 同步到数据库（默认启用）
  static Future<SkillActionResponse> rescanSkills() async {
    final body = await _http.post('$_baseUrl/skills/rescan');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillActionResponse.fromJson(json);
  }

  /// 获取 Skill 指令
  /// [name] Skill 名称
  /// 获取 Skill 的完整指令内容（Markdown 格式）
  static Future<SkillInstructionsResponse> getSkillInstructions(
    String name,
  ) async {
    final body = await _http.get('$_baseUrl/skills/$name/instructions');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillInstructionsResponse.fromJson(json);
  }

  /// 获取脚本列表
  /// [name] Skill 名称
  /// 获取指定 Skill 的 scripts 目录下的所有脚本文件
  static Future<SkillScriptsResponse> getSkillScripts(String name) async {
    final body = await _http.get('$_baseUrl/skills/$name/scripts');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillScriptsResponse.fromJson(json);
  }

  /// 获取脚本内容
  /// [name] Skill 名称
  /// [script] 脚本文件名
  static Future<SkillScriptContentResponse> getScriptContent(
    String name,
    String script,
  ) async {
    final body = await _http.get('$_baseUrl/skills/$name/scripts/$script');
    final json = jsonDecode(body) as Map<String, dynamic>;
    return SkillScriptContentResponse.fromJson(json);
  }
}
