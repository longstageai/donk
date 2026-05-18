import 'dart:io';

import 'package:donk/app/init/app.dart';
import 'package:flutter/cupertino.dart';

Future<void> main() async {
  final shouldContinue = await App.init();
  if (!shouldContinue) {
    // 已有实例在运行，退出当前程序
    exit(0);
  }
  runApp(const App());
}
