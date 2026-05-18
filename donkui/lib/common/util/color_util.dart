import 'package:flutter/material.dart';

class ColorUtil {
  static Color fromHex(String hexString) {
    final buffer = StringBuffer();
    if (hexString.length == 6 || hexString.length == 7) buffer.write('ff');
    buffer.write(hexString.replaceFirst('#', ''));
    return Color(int.parse(buffer.toString(), radix: 16));
  }

  // 转换颜色的工具方法
  static int hexToInt(String hex) {
    hex = hex.replaceAll("#", "");
    hex = "FF$hex"; // 添加不透明度
    return int.parse(hex, radix: 16);
  }
}
