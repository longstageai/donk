import 'package:donk/app/conf/colors.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'component/home_body.dart';
import 'home_controller.dart';
import 'component/home_bottom.dart';
import 'component/home_drawer.dart';
import 'component/home_header.dart';

/// 首页视图
/// 包含顶部导航栏、中间内容区域和底部输入框组件
class HomeView extends StatefulWidget {
  const HomeView({super.key});

  @override
  State<HomeView> createState() => _HomeViewState();
}

class _HomeViewState extends State<HomeView> {
  /// HomeController 控制器（从全局依赖中获取）
  final HomeController controller = Get.find();

  /// Scaffold状态键，用于控制抽屉开关
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();

  /// 切换右侧抽屉的显示/隐藏状态
  void _toggleDrawer() {
    if (_scaffoldKey.currentState!.isEndDrawerOpen) {
      _scaffoldKey.currentState!.closeEndDrawer();
    } else {
      _scaffoldKey.currentState!.openEndDrawer();
    }
  }

  @override
  Widget build(BuildContext context) {
    return TransparentDrawerTheme(
      child: Scaffold(
        key: _scaffoldKey,
        backgroundColor: AppColors.backgroundColor1,
        body: Padding(
          padding: const EdgeInsets.only(right: 25.0, left: 25.0, bottom: 20),
          child: Column(
            children: [
              /// 构建顶部导航栏
              /// 包含龙虾管家按钮、已使用资源显示和安全检查按钮
              HomeHeader(onTap: _toggleDrawer),

              /// 构建中间内容区域
              Expanded(child: HomeBody()),

              /// 构建底部输入框区域
              HomeBottom(),
            ],
          ),
        ),

        // 右侧抽屉
        endDrawer: Drawer(
          width: 300,
          elevation: 0,
          shape: RoundedRectangleBorder(),
          child: Container(
            decoration: BoxDecoration(
              color: Colors.white,
              border: Border.all(color: AppColors.backgroundColor, width: 1),
            ),
            child: HomeDrawer(),
          ),
        ),
      ),
    );
  }
}

/// 透明遮罩抽屉主题
/// 将抽屉的遮罩层设为透明
class TransparentDrawerTheme extends StatelessWidget {
  final Widget child;

  const TransparentDrawerTheme({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return DrawerTheme(
      data: DrawerThemeData(scrimColor: Colors.transparent, elevation: 0),
      child: child,
    );
  }
}
