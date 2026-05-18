import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../home_controller.dart';
import 'chat_list.dart';

/// 首页主体内容组件
/// 包含欢迎标题和功能卡片区域，或对话列表
class HomeBody extends StatefulWidget {
  const HomeBody({super.key});

  @override
  State<HomeBody> createState() => _HomeBodyState();
}

class _HomeBodyState extends State<HomeBody> {
  final controller = Get.find<HomeController>();

  /// 构建欢迎标题区域
  /// 显示主标题"Hi，我是donk"和副标题"随时随地，帮您高效干活"
  Widget title() {
    return Container(
      margin: EdgeInsets.only(top: 40),
      child: Column(
        children: [
          /// 主标题
          const Text(
            'Hi，我是Donk',
            style: TextStyle(
              fontSize: 36,
              fontWeight: FontWeight.bold,
              color: Colors.black87,
            ),
          ),

          /// 副标题
          const Text(
            '随时随地，帮您高效干活',
            style: TextStyle(fontSize: 18, color: Colors.grey),
          ),
        ],
      ),
    );
  }

  /// 构建功能卡片区域
  /// 横向排列5个功能卡片：安装Skill、邮件管理、整理桌面、安排日程、手机远程办公
  Widget body() {
    /// 每个卡片的固定宽度
    double cw = 160.0;
    return Container(
      padding: EdgeInsets.only(bottom: 10),
      child: Row(
        /// 水平方向均匀分布
        mainAxisAlignment: MainAxisAlignment.spaceBetween,

        /// 垂直方向底部对齐
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          /// 安装你的第一个Skill卡片
          _buildFeatureCard(
            icon: Icons.laptop,
            iconBgColor: const Color(0xFFE8E0FF),
            iconColor: const Color(0xFF8B5CF6),
            title: '安装你的第一个Skill',
            description: '一键教你安装超能力',
            cardBgColor: const Color(0xFFF9F5FF),
            cardWidth: cw,
          ),

          /// 邮件管理卡片
          _buildFeatureCard(
            icon: Icons.email,
            iconBgColor: const Color(0xFFFFE8E0),
            iconColor: const Color(0xFFF97316),
            title: '邮件管理',
            description: '帮你高效处理邮件',
            cardBgColor: const Color(0xFFFFF5F0),
            cardWidth: cw,
          ),

          /// 整理桌面卡片
          _buildFeatureCard(
            icon: Icons.cleaning_services,
            iconBgColor: const Color(0xFFE0FFF0),
            iconColor: const Color(0xFF10B981),
            title: '整理桌面',
            description: '还你清爽电脑桌面',
            cardBgColor: const Color(0xFFF0FFF5),
            cardWidth: cw,
          ),

          /// 安排日程卡片
          _buildFeatureCard(
            icon: Icons.calendar_today,
            iconBgColor: const Color(0xFFFFF0E8),
            iconColor: const Color(0xFFEF4444),
            title: '安排日程',
            description: '一句话约日程定会议',
            cardBgColor: const Color(0xFFFFF5F0),
            cardWidth: cw,
          ),

          /// 手机远程办公卡片
          _buildFeatureCard(
            icon: Icons.phone_android,
            iconBgColor: const Color(0xFFE0F4FF),
            iconColor: const Color(0xFF3B82F6),
            title: '手机远程办公',
            description: '随时处理在线任务',
            cardBgColor: const Color(0xFFF0F8FF),
            cardWidth: cw,
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Obx(() {
      // 如果有聊天消息，显示对话列表
      if (controller.hasChatMessages) {
        return const ChatList();
      }

      // 否则显示欢迎页面
      return Column(
        /// 垂直方向居中对齐
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          /// 欢迎标题区域
          title(),
          const SizedBox(height: 20),

          /// 功能卡片区域（占据剩余空间）
          Expanded(child: body()),
        ],
      );
    });
  }

  /// 构建功能卡片组件
  /// [icon] 图标
  /// [iconBgColor] 图标背景色
  /// [iconColor] 图标颜色
  /// [title] 卡片标题
  /// [description] 卡片描述
  /// [cardBgColor] 卡片背景色
  /// [cardWidth] 卡片宽度
  Widget _buildFeatureCard({
    required IconData icon,
    required Color iconBgColor,
    required Color iconColor,
    required String title,
    required String description,
    required Color cardBgColor,
    required double cardWidth,
  }) {
    /// 根据卡片宽度计算图标尺寸
    final iconSize = cardWidth * 0.5;

    /// 根据卡片宽度计算内边距
    final innerPadding = cardWidth * 0.15;

    /// 鼠标悬停时显示点击手势
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: () {},
        child: Container(
          /// 卡片装饰样式
          decoration: BoxDecoration(
            color: cardBgColor,
            borderRadius: BorderRadius.circular(10),
          ),

          /// 固定高度180
          height: 180,
          width: cardWidth,
          padding: EdgeInsets.all(10),
          child: Column(
            /// 内容居中对齐
            crossAxisAlignment: CrossAxisAlignment.center,
            mainAxisSize: MainAxisSize.min,
            children: [
              /// 卡片标题
              Text(
                title,
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.bold,
                  color: Colors.black87,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              SizedBox(height: innerPadding * 0.5),

              /// 卡片描述
              Text(
                description,
                style: TextStyle(fontSize: 12, color: Colors.grey),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              SizedBox(height: innerPadding * 0.5),

              /// 图标容器
              Container(
                width: 80,
                height: 80,
                decoration: BoxDecoration(
                  color: iconBgColor,
                  borderRadius: BorderRadius.circular(iconSize * 0.25),
                ),
                child: Icon(icon, size: 80, color: iconColor),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
