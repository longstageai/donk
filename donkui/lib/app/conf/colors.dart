import 'dart:ui';

import '../../common/util/color_util.dart';

abstract class AppColors {
  static Color backgroundColor = ColorUtil.fromHex("#ebebeb");
  static Color backgroundColor1 = ColorUtil.fromHex("#ffffff");
  static Color backgroundColor2 = ColorUtil.fromHex("#ededed");

  static Color choice = ColorUtil.fromHex("#1c212a");

  static Color c1 = ColorUtil.fromHex("#1e1e25");

  // 主题色
  static Color primary = ColorUtil.fromHex("#1890ff");

  // 文字颜色
  static Color textPrimary = ColorUtil.fromHex("#262626");
  static Color textSecondary = ColorUtil.fromHex("#595959");
  static Color textHint = ColorUtil.fromHex("#bfbfbf");




  // 分割线颜色
  static Color divider = ColorUtil.fromHex("#e8e8e8");
}
